package restore_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/smcsoluciones/backup-system/agent/internal/backup/manifest"
	"github.com/smcsoluciones/backup-system/agent/internal/backup/restore"
	"github.com/smcsoluciones/backup-system/agent/internal/compress"
	"github.com/smcsoluciones/backup-system/agent/internal/crypto"
	"github.com/smcsoluciones/backup-system/agent/internal/destination/local"
)

const testPass = "test-passphrase-for-restore-2024"

// buildBackupDest creates a fake backup destination with one encrypted file and a manifest.
func buildBackupDest(t *testing.T, jobID string, files map[string][]byte) (string, *manifest.Manifest) {
	t.Helper()
	destDir := t.TempDir()
	dest, err := local.New(destDir)
	if err != nil {
		t.Fatal(err)
	}
	defer dest.Close()

	startedAt := time.Now().UTC()
	mf := manifest.New(jobID, "test-node", `C:\Source`, "full")

	for relPath, content := range files {
		objectName := "jobs/" + jobID + "/20260101T000000Z/data/" + relPath + ".bsmc"

		// Compress + encrypt the content into the destination
		var compBuf bytes.Buffer
		zw, _ := compress.NewWriter(&compBuf)
		_, _ = zw.Write(content)
		_ = zw.Close()

		w, _ := dest.Write(objectName)
		_ = crypto.Encrypt(w, &compBuf, testPass)
		_ = w.Close()

		mf.Files = append(mf.Files, manifest.FileEntry{
			RelPath:    relPath,
			Size:       int64(len(content)),
			ModTime:    startedAt,
			ObjectName: objectName,
		})
	}

	// Seal and write manifest
	sealed, _ := mf.Seal(testPass)
	manifestPath := manifest.ObjectName(jobID, startedAt)
	w, _ := dest.Write(manifestPath)
	_, _ = w.Write(sealed)
	_ = w.Close()

	return destDir, mf
}

func noopLogger() *zap.Logger {
	return zap.NewNop()
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestRestoreAllFiles(t *testing.T) {
	jobID := "test-job-001"
	files := map[string][]byte{
		"docs/readme.txt": []byte("hello from backup"),
		"data/report.csv": []byte("col1,col2\n1,2\n3,4"),
	}

	destDir, _ := buildBackupDest(t, jobID, files)
	targetDir := t.TempDir()

	dest, _ := local.New(destDir)
	defer dest.Close()

	engine := restore.New(dest, noopLogger())
	result, err := engine.Run(context.Background(), restore.Options{
		JobID:      jobID,
		TargetPath: targetDir,
		Passphrase: testPass,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.RestoredFiles != 2 {
		t.Fatalf("expected 2 restored, got %d", result.RestoredFiles)
	}
	if result.ErrorFiles != 0 {
		t.Fatalf("expected 0 errors, got %d", result.ErrorFiles)
	}

	// Verify file contents
	for relPath, expected := range files {
		got, err := os.ReadFile(filepath.Join(targetDir, filepath.FromSlash(relPath)))
		if err != nil {
			t.Fatalf("read restored %q: %v", relPath, err)
		}
		if !bytes.Equal(expected, got) {
			t.Fatalf("%q: content mismatch\ngot:  %q\nwant: %q", relPath, got, expected)
		}
	}
}

func TestRestoreDryRunDoesNotWriteFiles(t *testing.T) {
	jobID := "test-job-dryrun"
	files := map[string][]byte{
		"file.txt": []byte("should not appear"),
	}

	destDir, _ := buildBackupDest(t, jobID, files)
	targetDir := t.TempDir()

	dest, _ := local.New(destDir)
	defer dest.Close()

	engine := restore.New(dest, noopLogger())
	result, err := engine.Run(context.Background(), restore.Options{
		JobID:      jobID,
		TargetPath: targetDir,
		Passphrase: testPass,
		DryRun:     true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.RestoredFiles != 1 {
		t.Fatalf("dry-run: expected 1 would-restore, got %d", result.RestoredFiles)
	}

	// File must NOT exist at target
	if _, err := os.Stat(filepath.Join(targetDir, "file.txt")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote file — it should not have")
	}
}

func TestRestoreWrongPassphraseFails(t *testing.T) {
	jobID := "test-job-wrongpass"
	destDir, _ := buildBackupDest(t, jobID, map[string][]byte{
		"f.txt": []byte("data"),
	})

	dest, _ := local.New(destDir)
	defer dest.Close()

	engine := restore.New(dest, noopLogger())
	_, err := engine.Run(context.Background(), restore.Options{
		JobID:      jobID,
		TargetPath: t.TempDir(),
		Passphrase: "completely-wrong-passphrase!!",
	})
	if err == nil {
		t.Fatal("expected error with wrong passphrase, got nil")
	}
}

func TestRestoreFilterSubset(t *testing.T) {
	jobID := "test-job-filter"
	files := map[string][]byte{
		"docs/a.txt":  []byte("in docs"),
		"other/b.txt": []byte("not in docs"),
	}

	destDir, _ := buildBackupDest(t, jobID, files)
	targetDir := t.TempDir()

	dest, _ := local.New(destDir)
	defer dest.Close()

	engine := restore.New(dest, noopLogger())
	result, err := engine.Run(context.Background(), restore.Options{
		JobID:      jobID,
		TargetPath: targetDir,
		Passphrase: testPass,
		Filter:     "docs/*",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.RestoredFiles != 1 {
		t.Fatalf("expected 1 restored with filter, got %d", result.RestoredFiles)
	}

	// docs/a.txt must exist
	if _, err := os.Stat(filepath.Join(targetDir, "docs", "a.txt")); os.IsNotExist(err) {
		t.Fatal("docs/a.txt was not restored")
	}
	// other/b.txt must NOT exist
	if _, err := os.Stat(filepath.Join(targetDir, "other", "b.txt")); !os.IsNotExist(err) {
		t.Fatal("other/b.txt was restored despite filter")
	}
}

func TestRestoreNoOverwriteByDefault(t *testing.T) {
	jobID := "test-job-nooverwrite"
	original := []byte("original")
	updated := []byte("updated content from backup")

	destDir, _ := buildBackupDest(t, jobID, map[string][]byte{
		"file.txt": updated,
	})
	targetDir := t.TempDir()

	// Pre-create the file with different content
	_ = os.WriteFile(filepath.Join(targetDir, "file.txt"), original, 0o644)

	dest, _ := local.New(destDir)
	defer dest.Close()

	engine := restore.New(dest, noopLogger())
	result, err := engine.Run(context.Background(), restore.Options{
		JobID:             jobID,
		TargetPath:        targetDir,
		Passphrase:        testPass,
		OverwriteExisting: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.SkippedFiles != 1 {
		t.Fatalf("expected 1 skipped, got %d", result.SkippedFiles)
	}

	// File should still have original content
	got, _ := os.ReadFile(filepath.Join(targetDir, "file.txt"))
	if !bytes.Equal(original, got) {
		t.Fatalf("file was overwritten despite OverwriteExisting=false")
	}
}

func TestRestoreOverwriteReplaces(t *testing.T) {
	jobID := "test-job-overwrite"
	updated := []byte("updated content from backup")

	destDir, _ := buildBackupDest(t, jobID, map[string][]byte{
		"file.txt": updated,
	})
	targetDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(targetDir, "file.txt"), []byte("old"), 0o644)

	dest, _ := local.New(destDir)
	defer dest.Close()

	engine := restore.New(dest, noopLogger())
	_, err := engine.Run(context.Background(), restore.Options{
		JobID:             jobID,
		TargetPath:        targetDir,
		Passphrase:        testPass,
		OverwriteExisting: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(targetDir, "file.txt"))
	if !bytes.Equal(updated, got) {
		t.Fatalf("file was not updated: got %q", got)
	}
}

func TestRestoreJobNotFound(t *testing.T) {
	destDir := t.TempDir()
	dest, _ := local.New(destDir)
	defer dest.Close()

	engine := restore.New(dest, noopLogger())
	_, err := engine.Run(context.Background(), restore.Options{
		JobID:      "nonexistent-job-id",
		TargetPath: t.TempDir(),
		Passphrase: testPass,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent job, got nil")
	}
}

func TestRestoreContextCancellation(t *testing.T) {
	jobID := "test-job-cancel"
	files := make(map[string][]byte)
	for i := 0; i < 10; i++ {
		files[filepath.Join("dir", filepath.FromSlash("f"+string(rune('0'+i))+".txt"))] = []byte("data")
	}

	destDir, _ := buildBackupDest(t, jobID, files)
	dest, _ := local.New(destDir)
	defer dest.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	engine := restore.New(dest, noopLogger())
	_, err := engine.Run(ctx, restore.Options{
		JobID:      jobID,
		TargetPath: t.TempDir(),
		Passphrase: testPass,
	})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
