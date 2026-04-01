package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
//  1. Creates a VSS snapshot (Windows only)
//  2. Scans the source directory for changed files
//  3. Compresses + encrypts each file, writes to destination
//  4. Saves an encrypted manifest
//  5. Reports progress to the server
func (e *Engine) Run(ctx context.Context, jobID string) (*RunResult, error) {
	if jobID == "" {
		jobID = uuid.New().String()
	}
	startedAt := time.Now().UTC()
	result := &RunResult{JobID: jobID, StartedAt: startedAt}

	backupCfg := e.cfg.Backup
	passphrase := backupCfg.EncryptionPassphrase

	e.log.Info("backup started",
		zap.String("job_id", jobID),
		zap.String("source", backupCfg.SourcePath),
		zap.Bool("incremental", backupCfg.Incremental),
		zap.Bool("use_vss", backupCfg.UseVSS),
	)

	// ── 1. VSS snapshot ──────────────────────────────────────────────────────
	sourcePath := backupCfg.SourcePath
	var snap *vss.Snapshot

	if backupCfg.UseVSS {
		volume := filepath.VolumeName(sourcePath) + `\`
		e.log.Info("creating VSS snapshot", zap.String("volume", volume))
		var err error
		snap, err = vss.Create(volume)
		if err != nil {
			return nil, fmt.Errorf("engine: VSS create: %w", err)
		}
		defer func() {
			if delErr := vss.Delete(snap.ShadowID); delErr != nil {
				e.log.Warn("engine: VSS delete", zap.Error(delErr))
			}
		}()
		// Remap source to shadow path
		sourcePath = snap.TranslatePath(sourcePath)
		e.log.Info("VSS shadow path", zap.String("shadow_path", sourcePath))
	}

	// ── 2. Load incremental cache ─────────────────────────────────────────────
	cachePath := filepath.Join(os.TempDir(), "backupsmc_cache_"+jobID+".json")
	if backupCfg.Incremental {
		cachePath = filepath.Join(os.TempDir(), "backupsmc_cache_"+
			sanitizeName(backupCfg.SourcePath)+".json")
	}

	cache, err := scanner.LoadCache(cachePath)
	if err != nil {
		e.log.Warn("engine: load cache (starting fresh)", zap.Error(err))
		cache = scanner.NewCache()
	}

	// ── 3. Scan for changed files ─────────────────────────────────────────────
	scanOpts := scanner.Options{
		Incremental:     backupCfg.Incremental,
		ComputeHash:     false, // fast mode; use mtime+size
		ExcludePatterns: backupCfg.ExcludePatterns,
	}
	sc := scanner.New(sourcePath, cache, scanOpts)
	scanResult, newCache, err := sc.Scan()
	if err != nil {
		return nil, fmt.Errorf("engine: scan: %w", err)
	}

	backupType := "full"
	if backupCfg.Incremental {
		backupType = "incremental"
	}

	mf := manifest.New(jobID, e.nodeID, backupCfg.SourcePath, backupType)

	// Count totals for progress
	for _, fi := range scanResult.Changed {
		result.TotalFiles++
		result.TotalBytes += fi.Size
		if fi.IsDir {
			result.TotalFiles--
		}
	}
	result.ChangedFiles = result.TotalFiles

	e.log.Info("scan complete",
		zap.Int64("changed_files", result.ChangedFiles),
		zap.Int64("total_bytes", result.TotalBytes),
		zap.Int("deleted", len(scanResult.Deleted)),
	)

	// ── 4. Back up changed files ──────────────────────────────────────────────
	ts := startedAt.Format("20060102T150405Z")
	var bytesDone int64
	var filesDone int64

	for _, fi := range scanResult.Changed {
		if fi.IsDir {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Translate path to VSS shadow path if active
		absPath := filepath.Join(sourcePath, filepath.FromSlash(fi.Path))
		if snap != nil {
			// sourcePath is already the shadow path; absPath is correct
		}

		objectName := fmt.Sprintf("jobs/%s/%s/data/%s.bsmc", jobID, ts, fi.Path)

		// Progress update
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
			RelPath:    fi.Path,
			Size:       fi.Size,
			ModTime:    fi.ModTime,
			ObjectName: objectName,
		}

		// Get Windows ACL (SDDL)
		if sddl, aclErr := acl.GetSDDL(absPath); aclErr == nil {
			entry.SDDL = sddl
		}

		// Back up the file (chunked if large)
		if fi.Size > chunkThreshold {
			chunks, backupErr := e.backupLargeFile(ctx, absPath, jobID, ts, fi.Path, passphrase)
			if backupErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", fi.Path, backupErr))
				entry.Skipped = true
				mf.Files = append(mf.Files, entry)
				continue
			}
			entry.Chunks = chunks
			entry.ObjectName = "" // object split into chunks
		} else {
			if backupErr := e.backupFile(ctx, absPath, objectName, passphrase, backupCfg.SkipCompressExts); backupErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", fi.Path, backupErr))
				entry.Skipped = true
				mf.Files = append(mf.Files, entry)
				continue
			}
		}

		bytesDone += fi.Size
		filesDone++
		result.ChangedBytes += fi.Size
		mf.Files = append(mf.Files, entry)
	}

	// ── 5. Write manifest ─────────────────────────────────────────────────────
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
	w, err := e.dest.Write(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("engine: open manifest dest: %w", err)
	}
	if _, err := w.Write(sealed); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("engine: write manifest: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("engine: close manifest: %w", err)
	}

	// ── 6. Persist cache ──────────────────────────────────────────────────────
	if saveErr := newCache.Save(cachePath); saveErr != nil {
		e.log.Warn("engine: save cache", zap.Error(saveErr))
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
func (e *Engine) backupFile(ctx context.Context, srcPath, objectName, passphrase string, skipCompressExts map[string]bool) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	dst, err := e.dest.Write(objectName)
	if err != nil {
		return fmt.Errorf("dest write: %w", err)
	}

	// Decide whether to compress based on file extension
	ext := filepath.Ext(srcPath)
	doCompress := !skipCompressExts[ext]

	var src io.Reader = f

	if doCompress {
		// Compress → encrypt → destination
		pr, pw := io.Pipe()

		go func() {
			zw, zerr := compress.NewWriter(pw)
			if zerr != nil {
				pw.CloseWithError(zerr)
				return
			}
			_, copyErr := io.Copy(zw, f)
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
		return fmt.Errorf("encrypt: %w", encErr)
	}
	return dst.Close()
}

// backupLargeFile splits a file into 512 MB chunks, each compressed+encrypted separately.
func (e *Engine) backupLargeFile(ctx context.Context, srcPath, jobID, ts, relPath, passphrase string) ([]manifest.Chunk, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("open large file: %w", err)
	}
	defer f.Close()

	var chunks []manifest.Chunk
	buf := make([]byte, chunkSize)
	idx := 0

	for {
		n, readErr := io.ReadFull(f, buf)
		if n == 0 {
			break
		}

		objectName := fmt.Sprintf("jobs/%s/%s/data/%s.part%04d.bsmc", jobID, ts, relPath, idx)
		dst, err := e.dest.Write(objectName)
		if err != nil {
			return nil, fmt.Errorf("chunk %d dest: %w", idx, err)
		}

		// Compress chunk
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
