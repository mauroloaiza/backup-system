package local_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/smcsoluciones/backup-system/agent/internal/destination/local"
)

func TestWriteRead(t *testing.T) {
	dir := t.TempDir()
	dest, err := local.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer dest.Close()

	payload := []byte("backup data content for testing")

	// Write
	w, err := dest.Write("jobs/test/file.bsmc")
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("w.Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("w.Close: %v", err)
	}

	// Read back
	r, err := dest.Read("jobs/test/file.bsmc")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer r.Close()

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(payload, got) {
		t.Fatalf("content mismatch: got %q, want %q", got, payload)
	}
}

func TestAtomicWrite_FileOnlyAppearsAfterClose(t *testing.T) {
	// The final file must not exist while the write is in progress (before Close).
	// After Close() it must exist. This validates atomic write via .tmp → rename.
	dir := t.TempDir()
	dest, _ := local.New(dir)

	finalPath := filepath.Join(dir, "jobs", "test", "atomic.bsmc")

	w, err := dest.Write("jobs/test/atomic.bsmc")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("atomic payload"))

	// Before Close: final file must NOT exist
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		_ = w.Close()
		t.Fatal("final file exists before Close() — atomic write not working")
	}

	// After Close: final file MUST exist
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		t.Fatal("final file does not exist after Close()")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	dest, _ := local.New(dir)

	w, _ := dest.Write("to-delete.bsmc")
	_, _ = w.Write([]byte("data"))
	_ = w.Close()

	if err := dest.Delete("to-delete.bsmc"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := dest.Read("to-delete.bsmc"); err == nil {
		t.Fatal("expected error reading deleted file, got nil")
	}
}

func TestDeleteNonExistentNoError(t *testing.T) {
	dir := t.TempDir()
	dest, _ := local.New(dir)
	if err := dest.Delete("does-not-exist.bsmc"); err != nil {
		t.Fatalf("Delete nonexistent: %v", err)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	dest, _ := local.New(dir)

	names := []string{"a/file1.bsmc", "a/file2.bsmc", "b/file3.bsmc"}
	for _, name := range names {
		w, _ := dest.Write(name)
		_, _ = w.Write([]byte("x"))
		_ = w.Close()
	}

	listed, err := dest.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(listed) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(listed), listed)
	}
}

func TestCreatesDirIfNotExists(t *testing.T) {
	dir := t.TempDir() + "/new/nested/dir"
	dest, err := local.New(dir)
	if err != nil {
		t.Fatalf("New with new dir: %v", err)
	}
	defer dest.Close()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("directory was not created")
	}
}
