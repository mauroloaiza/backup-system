package restore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smcsoluciones/backup-system/agent/internal/backup/acl"
	"github.com/smcsoluciones/backup-system/agent/internal/backup/manifest"
	"github.com/smcsoluciones/backup-system/agent/internal/compress"
	"github.com/smcsoluciones/backup-system/agent/internal/crypto"
	"github.com/smcsoluciones/backup-system/agent/internal/destination"
)

// Options controls restore behaviour.
type Options struct {
	// JobID is the backup job to restore from.
	JobID string
	// TargetPath is the directory where files will be written.
	TargetPath string
	// Passphrase must match the one used during backup.
	Passphrase string
	// Filter is an optional glob pattern applied to RelPath (e.g. "docs/**").
	// Empty means restore all files.
	Filter string
	// OverwriteExisting overwrites files that already exist at TargetPath.
	OverwriteExisting bool
	// DryRun lists what would be restored without writing anything.
	DryRun bool
	// RestoreACLs applies saved Windows ACLs after writing each file.
	RestoreACLs bool
}

// FileResult is the outcome for a single restored file.
type FileResult struct {
	RelPath string
	Status  string // "restored" | "skipped" | "exists" | "error"
	Error   error
}

// Result summarises a completed restore operation.
type Result struct {
	JobID         string
	ManifestPath  string
	TotalFiles    int
	RestoredFiles int
	SkippedFiles  int
	ErrorFiles    int
	Files         []FileResult
	StartedAt     time.Time
	FinishedAt    time.Time
}

// Engine executes restore operations.
type Engine struct {
	dest destination.Writer
	log  *zap.Logger
}

// New creates a restore Engine backed by the given destination.
func New(dest destination.Writer, log *zap.Logger) *Engine {
	return &Engine{dest: dest, log: log}
}

// Run restores a backup job. It:
//  1. Discovers the manifest for the given JobID
//  2. Decrypts and parses the manifest
//  3. Restores each matching file to TargetPath
func (e *Engine) Run(ctx context.Context, opts Options) (*Result, error) {
	startedAt := time.Now().UTC()
	result := &Result{JobID: opts.JobID, StartedAt: startedAt}

	// ── 1. Locate manifest ────────────────────────────────────────────────────
	manifestPath, err := e.findManifest(opts.JobID)
	if err != nil {
		return nil, fmt.Errorf("restore: find manifest: %w", err)
	}
	result.ManifestPath = manifestPath

	e.log.Info("restore started",
		zap.String("job_id", opts.JobID),
		zap.String("manifest", manifestPath),
		zap.String("target", opts.TargetPath),
		zap.Bool("dry_run", opts.DryRun),
	)

	// ── 2. Read + decrypt manifest ────────────────────────────────────────────
	mf, err := e.readManifest(manifestPath, opts.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("restore: read manifest: %w", err)
	}

	e.log.Info("manifest loaded",
		zap.String("backup_type", mf.BackupType),
		zap.Int("files", len(mf.Files)),
		zap.Time("backed_up_at", mf.FinishedAt),
	)

	// ── 3. Restore files ──────────────────────────────────────────────────────
	if !opts.DryRun {
		if err := os.MkdirAll(opts.TargetPath, 0o750); err != nil {
			return nil, fmt.Errorf("restore: create target dir: %w", err)
		}
	}

	for _, entry := range mf.Files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result.TotalFiles++

		if entry.Skipped {
			result.SkippedFiles++
			result.Files = append(result.Files, FileResult{RelPath: entry.RelPath, Status: "skipped"})
			continue
		}

		if !matchesFilter(entry.RelPath, opts.Filter) {
			result.SkippedFiles++
			result.Files = append(result.Files, FileResult{RelPath: entry.RelPath, Status: "skipped"})
			continue
		}

		fr := e.restoreFile(ctx, entry, opts)
		result.Files = append(result.Files, fr)

		switch fr.Status {
		case "restored":
			result.RestoredFiles++
		case "exists":
			result.SkippedFiles++
		case "error":
			result.ErrorFiles++
			e.log.Warn("restore: file error",
				zap.String("path", entry.RelPath),
				zap.Error(fr.Error),
			)
		}
	}

	result.FinishedAt = time.Now().UTC()

	e.log.Info("restore completed",
		zap.String("job_id", opts.JobID),
		zap.Int("restored", result.RestoredFiles),
		zap.Int("skipped", result.SkippedFiles),
		zap.Int("errors", result.ErrorFiles),
		zap.Duration("duration", result.FinishedAt.Sub(startedAt)),
	)

	return result, nil
}

// restoreFile restores a single file entry to the target path.
func (e *Engine) restoreFile(ctx context.Context, entry manifest.FileEntry, opts Options) FileResult {
	fr := FileResult{RelPath: entry.RelPath}

	destPath := filepath.Join(opts.TargetPath, filepath.FromSlash(entry.RelPath))

	// Dry run: just report
	if opts.DryRun {
		e.log.Info("dry-run: would restore", zap.String("path", entry.RelPath))
		fr.Status = "restored"
		return fr
	}

	// Check existing
	if !opts.OverwriteExisting {
		if _, err := os.Stat(destPath); err == nil {
			fr.Status = "exists"
			return fr
		}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
		fr.Status = "error"
		fr.Error = fmt.Errorf("mkdir: %w", err)
		return fr
	}

	// Write to .tmp then rename (atomic)
	tmpPath := destPath + ".restore.tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		fr.Status = "error"
		fr.Error = fmt.Errorf("create tmp: %w", err)
		return fr
	}

	var writeErr error
	if len(entry.Chunks) > 0 {
		writeErr = e.writeChunkedFile(ctx, f, entry.Chunks, opts.Passphrase)
	} else {
		writeErr = e.writeFile(ctx, f, entry.ObjectName, opts.Passphrase)
	}

	f.Close()

	if writeErr != nil {
		_ = os.Remove(tmpPath)
		fr.Status = "error"
		fr.Error = writeErr
		return fr
	}

	// Restore original mtime
	_ = os.Chtimes(tmpPath, entry.ModTime, entry.ModTime)

	// Atomic rename
	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		fr.Status = "error"
		fr.Error = fmt.Errorf("rename: %w", err)
		return fr
	}

	// Restore ACL (Windows only, no-op on other platforms)
	if opts.RestoreACLs && entry.SDDL != "" {
		if aclErr := acl.SetSDDL(destPath, entry.SDDL); aclErr != nil {
			e.log.Warn("restore: set ACL failed",
				zap.String("path", entry.RelPath),
				zap.Error(aclErr),
			)
		}
	}

	fr.Status = "restored"
	return fr
}

// writeFile reads one encrypted object from the destination, decrypts and
// decompresses it, writing the plaintext to dst.
func (e *Engine) writeFile(_ context.Context, dst io.Writer, objectName, passphrase string) error {
	src, err := e.dest.Read(objectName)
	if err != nil {
		return fmt.Errorf("read object %q: %w", objectName, err)
	}
	defer src.Close()

	return decryptDecompress(dst, src, passphrase)
}

// writeChunkedFile reads all chunks in order, decrypting each and writing
// them sequentially to dst.
func (e *Engine) writeChunkedFile(_ context.Context, dst io.Writer, chunks []manifest.Chunk, passphrase string) error {
	// Sort by index to guarantee order
	sorted := make([]manifest.Chunk, len(chunks))
	copy(sorted, chunks)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Index < sorted[j].Index
	})

	for _, chunk := range sorted {
		src, err := e.dest.Read(chunk.ObjectName)
		if err != nil {
			return fmt.Errorf("read chunk %d %q: %w", chunk.Index, chunk.ObjectName, err)
		}
		if err := decryptDecompress(dst, src, passphrase); err != nil {
			src.Close()
			return fmt.Errorf("decrypt chunk %d: %w", chunk.Index, err)
		}
		src.Close()
	}
	return nil
}

// decryptDecompress wires: src → AES-256-GCM decrypt → zstd decompress → dst.
func decryptDecompress(dst io.Writer, src io.Reader, passphrase string) error {
	// Decrypt into a pipe, decompress the other end
	pr, pw := io.Pipe()

	var decryptErr error
	go func() {
		decryptErr = crypto.Decrypt(pw, src, passphrase)
		pw.CloseWithError(decryptErr)
	}()

	zr, err := compress.NewReader(pr)
	if err != nil {
		pr.CloseWithError(err)
		return fmt.Errorf("decompress reader: %w", err)
	}
	defer zr.Close()

	if _, err := io.Copy(dst, zr); err != nil {
		return fmt.Errorf("decompress copy: %w", err)
	}
	return decryptErr
}

// findManifest locates the most recent manifest object for a given job ID.
// Objects are stored as: jobs/{jobID}/{ts}/manifest.bsmc
func (e *Engine) findManifest(jobID string) (string, error) {
	prefix := fmt.Sprintf("jobs/%s/", jobID)
	all, err := e.dest.List(prefix)
	if err != nil {
		return "", fmt.Errorf("list objects under %q: %w", prefix, err)
	}

	var manifests []string
	for _, name := range all {
		if strings.HasSuffix(name, "/manifest.bsmc") {
			manifests = append(manifests, name)
		}
	}

	if len(manifests) == 0 {
		return "", fmt.Errorf("no manifest found for job %q", jobID)
	}

	// Sort lexicographically — timestamps are ISO 8601 so latest = last
	sort.Strings(manifests)
	return manifests[len(manifests)-1], nil
}

// readManifest reads and decrypts the manifest from the destination.
func (e *Engine) readManifest(objectName, passphrase string) (*manifest.Manifest, error) {
	r, err := e.dest.Read(objectName)
	if err != nil {
		return nil, fmt.Errorf("open manifest: %w", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read manifest bytes: %w", err)
	}

	return manifest.Open(data, passphrase)
}

// matchesFilter returns true if relPath matches the glob filter (or filter is empty).
func matchesFilter(relPath, filter string) bool {
	if filter == "" {
		return true
	}
	matched, err := filepath.Match(filter, relPath)
	if err != nil {
		return false
	}
	if matched {
		return true
	}
	// Also match if the path starts with the filter prefix (directory filter)
	return strings.HasPrefix(relPath, strings.TrimSuffix(filter, "*"))
}
