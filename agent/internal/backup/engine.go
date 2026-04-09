package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/smcsoluciones/backup-system/agent/internal/backup/acl"
	"github.com/smcsoluciones/backup-system/agent/internal/backup/manifest"
	"github.com/smcsoluciones/backup-system/agent/internal/backup/scanner"
	"github.com/smcsoluciones/backup-system/agent/internal/backup/vss"
	"github.com/smcsoluciones/backup-system/agent/internal/compress"
	"github.com/smcsoluciones/backup-system/agent/internal/config"
	"github.com/smcsoluciones/backup-system/agent/internal/crypto"
	"github.com/smcsoluciones/backup-system/agent/internal/destination"
	"github.com/smcsoluciones/backup-system/agent/internal/reporter"
	"github.com/smcsoluciones/backup-system/agent/internal/retry"
	"github.com/smcsoluciones/backup-system/agent/internal/throttle"
)

const (
	// Files larger than chunkThreshold are split into 512 MB chunks.
	chunkThreshold = 512 * 1024 * 1024
	chunkSize      = 512 * 1024 * 1024
)

// Engine orchestrates a full or incremental backup job.
type Engine struct {
	cfg    *config.Config
	dest   destination.Writer
	rep    *reporter.Reporter
	log    *zap.Logger
	nodeID string
}

// New creates a new backup Engine.
func New(cfg *config.Config, dest destination.Writer, rep *reporter.Reporter, log *zap.Logger, nodeID string) *Engine {
	return &Engine{
		cfg:    cfg,
		dest:   dest,
		rep:    rep,
		log:    log,
		nodeID: nodeID,
	}
}

// RunResult holds the outcome of a backup run.
type RunResult struct {
	JobID        string
	ManifestPath string
	TotalFiles   int64
	ChangedFiles int64
	TotalBytes   int64
	ChangedBytes int64
	Errors       []string
	StartedAt    time.Time
	FinishedAt   time.Time
}

// Run executes a backup job. It:
//  1. Runs the pre-script (if configured); aborts on non-zero exit
//  2. Creates VSS snapshots per volume (Windows only)
//  3. Scans each source directory for changed files
//  4. Compresses + encrypts each file, writes to destination
//  5. Optionally verifies each written file by decrypting and hashing
//  6. Saves an encrypted manifest
//  7. Reports progress to the server
//  8. Runs the post-script (always, even on failure)
func (e *Engine) Run(ctx context.Context, jobID string) (*RunResult, error) {
	if jobID == "" {
		jobID = uuid.New().String()
	}
	startedAt := time.Now().UTC()
	result := &RunResult{JobID: jobID, StartedAt: startedAt}

	backupCfg := e.cfg.Backup
	passphrase := backupCfg.EncryptionPassphrase

	sourcePaths := e.cfg.EffectiveSourcePaths()

	// ── 0. Pre-script ─────────────────────────────────────────────────────────
	if backupCfg.PreScript != "" {
		e.log.Info("engine: running pre-script", zap.String("script", backupCfg.PreScript))
		if err := runScript(ctx, backupCfg.PreScript); err != nil {
			return nil, fmt.Errorf("engine: pre-script: %w", err)
		}
	}
	// Post-script always executes when Run returns (success or failure).
	if backupCfg.PostScript != "" {
		defer func() {
			e.log.Info("engine: running post-script", zap.String("script", backupCfg.PostScript))
			if err := runScript(context.Background(), backupCfg.PostScript); err != nil {
				e.log.Warn("engine: post-script error", zap.Error(err))
			}
		}()
	}

	e.log.Info("backup started",
		zap.String("job_id", jobID),
		zap.Strings("sources", sourcePaths),
		zap.Bool("incremental", backupCfg.Incremental),
		zap.Bool("use_vss", backupCfg.UseVSS),
		zap.Float64("throttle_mbps", backupCfg.ThrottleMbps),
		zap.Bool("verify_after_backup", backupCfg.VerifyAfterBackup),
	)

	// ── 1. VSS snapshots (one per unique volume) ──────────────────────────────
	snaps := map[string]*vss.Snapshot{}
	if backupCfg.UseVSS {
		for _, sp := range sourcePaths {
			vol := filepath.VolumeName(sp) + `\`
			if _, already := snaps[vol]; already {
				continue
			}
			e.log.Info("creating VSS snapshot", zap.String("volume", vol))
			snap, err := vss.Create(vol)
			if err != nil {
				return nil, fmt.Errorf("engine: VSS create %q: %w", vol, err)
			}
			snaps[vol] = snap
		}
		defer func() {
			for _, snap := range snaps {
				if delErr := vss.Delete(snap.ShadowID); delErr != nil {
					e.log.Warn("engine: VSS delete", zap.Error(delErr))
				}
			}
		}()
	}

	backupType := "full"
	if backupCfg.Incremental {
		backupType = "incremental"
	}

	mf := manifest.New(jobID, e.nodeID, sourcePaths[0], backupType)
	ts := startedAt.Format("20060102T150405Z")
	var bytesDone int64
	var filesDone int64

	// ── 2-4. Scan + backup each source path ───────────────────────────────────
	for _, origSourcePath := range sourcePaths {
		sourcePath := origSourcePath
		if backupCfg.UseVSS {
			vol := filepath.VolumeName(origSourcePath) + `\`
			if snap, ok := snaps[vol]; ok {
				sourcePath = snap.TranslatePath(origSourcePath)
				e.log.Info("VSS shadow path",
					zap.String("original", origSourcePath),
					zap.String("shadow", sourcePath))
			}
		}

		// Persistent cache (survives reboots, %TEMP% clears).
		cachePath := config.CachePath(origSourcePath)
		cache, err := scanner.LoadCache(cachePath)
		if err != nil {
			e.log.Warn("engine: load cache (starting fresh)",
				zap.String("source", origSourcePath), zap.Error(err))
			cache = scanner.NewCache()
		}

		scanOpts := scanner.Options{
			Incremental:     backupCfg.Incremental,
			ComputeHash:     false,
			ExcludePatterns: backupCfg.ExcludePatterns,
		}
		sc := scanner.New(sourcePath, cache, scanOpts)
		scanResult, newCache, err := sc.Scan()
		if err != nil {
			return nil, fmt.Errorf("engine: scan %q: %w", origSourcePath, err)
		}

		for _, fi := range scanResult.Changed {
			if !fi.IsDir {
				result.TotalFiles++
				result.TotalBytes += fi.Size
			}
		}

		e.log.Info("scan complete",
			zap.String("source", origSourcePath),
			zap.Int64("changed_files", result.TotalFiles),
			zap.Int64("total_bytes", result.TotalBytes),
			zap.Int("deleted", len(scanResult.Deleted)),
		)

		for _, fi := range scanResult.Changed {
			if fi.IsDir {
				continue
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			absPath := filepath.Join(sourcePath, filepath.FromSlash(fi.Path))
			srcTag := sanitizeName(origSourcePath)
			objectName := fmt.Sprintf("jobs/%s/%s/data/%s/%s.bsmc", jobID, ts, srcTag, fi.Path)

			e.rep.Update(reporter.Progress{
				JobID:       jobID,
				NodeID:      e.nodeID,
				Status:      "running",
				FilesTotal:  result.TotalFiles,
				FilesDone:   filesDone,
				BytesTotal:  result.TotalBytes,
				BytesDone:   bytesDone,
				CurrentFile: fi.Path,
				StartedAt:   startedAt,
			})

			entry := manifest.FileEntry{
				RelPath:    srcTag + "/" + fi.Path,
				Size:       fi.Size,
				ModTime:    fi.ModTime,
				ObjectName: objectName,
			}

			if sddl, aclErr := acl.GetSDDL(absPath); aclErr == nil {
				entry.SDDL = sddl
			}

			ext := filepath.Ext(absPath)
			doCompress := !backupCfg.SkipCompressExts[ext]

			if fi.Size > chunkThreshold {
				chunks, backupErr := e.backupLargeFile(ctx, absPath, jobID, ts, srcTag+"/"+fi.Path, passphrase)
				if backupErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", fi.Path, backupErr))
					entry.Skipped = true
					mf.Files = append(mf.Files, entry)
					continue
				}
				entry.Chunks = chunks
				entry.ObjectName = ""
			} else {
				var hash string
				backupErr := retry.Do(ctx, e.cfg.Retry, e.log, func() error {
					h, err := e.backupFile(ctx, absPath, objectName, passphrase, doCompress, backupCfg.ThrottleMbps)
					if err != nil {
						// Source gone or no permission — no point retrying.
						if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
							return retry.Permanent{Err: err}
						}
						return err
					}
					hash = h
					return nil
				})
				if backupErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", fi.Path, backupErr))
					entry.Skipped = true
					mf.Files = append(mf.Files, entry)
					continue
				}
				entry.SHA256 = hash

				// ── 5. Optional post-write integrity verification ──────────────────
				if backupCfg.VerifyAfterBackup && hash != "" {
					if vErr := e.verifyFile(objectName, passphrase, doCompress, hash); vErr != nil {
						e.log.Warn("engine: verify failed",
							zap.String("file", fi.Path), zap.Error(vErr))
						result.Errors = append(result.Errors,
							fmt.Sprintf("%s: verify: %v", fi.Path, vErr))
					}
				}
			}

			bytesDone += fi.Size
			filesDone++
			result.ChangedBytes += fi.Size
			mf.Files = append(mf.Files, entry)
		}

		if saveErr := newCache.Save(cachePath); saveErr != nil {
			e.log.Warn("engine: save cache", zap.Error(saveErr))
		}
	}

	result.ChangedFiles = filesDone

	// ── 6. Write manifest ─────────────────────────────────────────────────────
	mf.TotalFiles = result.TotalFiles
	mf.TotalBytes = result.TotalBytes
	mf.ChangedFiles = result.ChangedFiles
	mf.ChangedBytes = result.ChangedBytes
	mf.SkippedFiles = int64(len(result.Errors))

	sealed, err := mf.Seal(passphrase)
	if err != nil {
		return nil, fmt.Errorf("engine: seal manifest: %w", err)
	}

	manifestPath := manifest.ObjectName(jobID, startedAt)
	manifestErr := retry.Do(ctx, e.cfg.Retry, e.log, func() error {
		w, err := e.dest.Write(manifestPath)
		if err != nil {
			return fmt.Errorf("open manifest dest: %w", err)
		}
		if _, err := w.Write(sealed); err != nil {
			_ = w.Close()
			return fmt.Errorf("write manifest: %w", err)
		}
		return w.Close()
	})
	if manifestErr != nil {
		return nil, fmt.Errorf("engine: manifest: %w", manifestErr)
	}

	result.ManifestPath = manifestPath
	result.FinishedAt = time.Now().UTC()

	status := "completed"
	if len(result.Errors) > 0 {
		status = "warning"
	}
	e.rep.Update(reporter.Progress{
		JobID:      jobID,
		NodeID:     e.nodeID,
		Status:     status,
		FilesTotal: result.TotalFiles,
		FilesDone:  filesDone,
		BytesTotal: result.TotalBytes,
		BytesDone:  bytesDone,
		StartedAt:  startedAt,
	})
	e.rep.Flush(ctx)

	e.log.Info("backup completed",
		zap.String("job_id", jobID),
		zap.Int64("changed_files", result.ChangedFiles),
		zap.Int64("changed_bytes", result.ChangedBytes),
		zap.Int("errors", len(result.Errors)),
		zap.Duration("duration", result.FinishedAt.Sub(startedAt)),
	)

	return result, nil
}

// backupFile compresses + encrypts a single file and writes it to the destination.
// Returns the SHA-256 hex digest of the plaintext (uncompressed) content.
func (e *Engine) backupFile(ctx context.Context, srcPath, objectName, passphrase string, doCompress bool, throttleMbps float64) (sha256hex string, err error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	dst, err := e.dest.Write(objectName)
	if err != nil {
		return "", fmt.Errorf("dest write: %w", err)
	}

	// Apply bandwidth throttle if configured.
	var rawSrc io.Reader = f
	if throttleMbps > 0 {
		rawSrc = throttle.NewReader(f, throttleMbps)
	}

	// Hash tee: computes SHA256 of the plaintext as it flows through.
	h := sha256.New()
	teeRaw := io.TeeReader(rawSrc, h)

	var src io.Reader = teeRaw

	if doCompress {
		pr, pw := io.Pipe()
		go func() {
			zw, zerr := compress.NewWriter(pw)
			if zerr != nil {
				pw.CloseWithError(zerr)
				return
			}
			_, copyErr := io.Copy(zw, teeRaw)
			closeErr := zw.Close()
			if copyErr != nil {
				pw.CloseWithError(copyErr)
			} else {
				pw.CloseWithError(closeErr)
			}
		}()
		src = pr
	}

	if encErr := crypto.Encrypt(dst, src, passphrase); encErr != nil {
		_ = dst.Close()
		return "", fmt.Errorf("encrypt: %w", encErr)
	}
	if closeErr := dst.Close(); closeErr != nil {
		return "", fmt.Errorf("close: %w", closeErr)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifyFile reads back the written object, decrypts+decompresses it, and
// checks that its SHA-256 matches the expected hash.
func (e *Engine) verifyFile(objectName, passphrase string, wasCompressed bool, expectedHash string) error {
	r, err := e.dest.Read(objectName)
	if err != nil {
		return fmt.Errorf("open for verify: %w", err)
	}
	defer r.Close()

	// Decrypt into an in-memory buffer.
	var plain bytes.Buffer
	if err := crypto.Decrypt(&plain, r, passphrase); err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}

	var dataReader io.Reader = &plain
	if wasCompressed {
		zr, err := compress.NewReader(&plain)
		if err != nil {
			return fmt.Errorf("decompress open: %w", err)
		}
		defer zr.Close()
		dataReader = zr
	}

	h := sha256.New()
	if _, err := io.Copy(h, dataReader); err != nil {
		return fmt.Errorf("hash read: %w", err)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedHash {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, got)
	}
	return nil
}

// backupLargeFile splits a file into 512 MB chunks, each compressed+encrypted separately.
func (e *Engine) backupLargeFile(ctx context.Context, srcPath, jobID, ts, relPath, passphrase string) ([]manifest.Chunk, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("open large file: %w", err)
	}
	defer f.Close()

	// Apply throttle on the raw file reader.
	var src io.Reader = f
	if e.cfg.Backup.ThrottleMbps > 0 {
		src = throttle.NewReader(f, e.cfg.Backup.ThrottleMbps)
	}

	var chunks []manifest.Chunk
	buf := make([]byte, chunkSize)
	idx := 0

	for {
		n, readErr := io.ReadFull(src, buf)
		if n == 0 {
			break
		}

		objectName := fmt.Sprintf("jobs/%s/%s/data/%s.part%04d.bsmc", jobID, ts, relPath, idx)
		dst, err := e.dest.Write(objectName)
		if err != nil {
			return nil, fmt.Errorf("chunk %d dest: %w", idx, err)
		}

		var compBuf bytes.Buffer
		zw, _ := compress.NewWriter(&compBuf)
		_, _ = zw.Write(buf[:n])
		_ = zw.Close()

		if encErr := crypto.Encrypt(dst, &compBuf, passphrase); encErr != nil {
			_ = dst.Close()
			return nil, fmt.Errorf("chunk %d encrypt: %w", idx, encErr)
		}
		if closeErr := dst.Close(); closeErr != nil {
			return nil, fmt.Errorf("chunk %d close: %w", idx, closeErr)
		}

		chunks = append(chunks, manifest.Chunk{
			Index:      idx,
			ObjectName: objectName,
			Size:       int64(n),
		})
		idx++

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("chunk read: %w", readErr)
		}
	}

	return chunks, nil
}

// runScript executes a shell command using the platform's native interpreter.
// On Windows: cmd.exe /C <script>. On all others: sh -c <script>.
func runScript(ctx context.Context, script string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", script)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", script)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exit error: %w\noutput:\n%s", err, out)
	}
	return nil
}

func sanitizeName(s string) string {
	safe := make([]byte, len(s))
	for i, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			safe[i] = c
		default:
			safe[i] = '_'
		}
	}
	return string(safe)
}
