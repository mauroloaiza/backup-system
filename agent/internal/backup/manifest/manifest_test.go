package manifest_test

import (
	"testing"
	"time"

	"github.com/smcsoluciones/backup-system/agent/internal/backup/manifest"
)

const testPass = "test-passphrase-minimum-16-chars"

func TestManifestSealOpen(t *testing.T) {
	m := manifest.New("job-123", "node-abc", `C:\Data`, "full")
	m.Files = []manifest.FileEntry{
		{RelPath: "docs/file.txt", Size: 1234, ModTime: time.Now().UTC(), ObjectName: "jobs/job-123/data/docs/file.txt.bsmc"},
		{RelPath: "img/photo.jpg", Size: 5678, ModTime: time.Now().UTC(), ObjectName: "jobs/job-123/data/img/photo.jpg.bsmc"},
	}
	m.TotalFiles = 2
	m.TotalBytes = 6912

	sealed, err := m.Seal(testPass)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if len(sealed) == 0 {
		t.Fatal("Seal returned empty blob")
	}

	reopened, err := manifest.Open(sealed, testPass)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if reopened.JobID != m.JobID {
		t.Errorf("JobID mismatch: got %q, want %q", reopened.JobID, m.JobID)
	}
	if reopened.NodeID != m.NodeID {
		t.Errorf("NodeID mismatch: got %q, want %q", reopened.NodeID, m.NodeID)
	}
	if len(reopened.Files) != len(m.Files) {
		t.Errorf("Files count mismatch: got %d, want %d", len(reopened.Files), len(m.Files))
	}
	if reopened.Files[0].RelPath != "docs/file.txt" {
		t.Errorf("File[0] RelPath mismatch: got %q", reopened.Files[0].RelPath)
	}
	if reopened.TotalBytes != 6912 {
		t.Errorf("TotalBytes mismatch: got %d", reopened.TotalBytes)
	}
	if reopened.EncryptionAlg != "AES-256-GCM" {
		t.Errorf("EncryptionAlg mismatch: got %q", reopened.EncryptionAlg)
	}
}

func TestManifestOpenWrongPassphraseFails(t *testing.T) {
	m := manifest.New("job-456", "node-xyz", `C:\Data`, "incremental")
	sealed, err := m.Seal(testPass)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manifest.Open(sealed, "wrong-passphrase-entirely!!")
	if err == nil {
		t.Fatal("expected error opening with wrong passphrase, got nil")
	}
}

func TestManifestObjectName(t *testing.T) {
	ts := time.Date(2026, 3, 31, 20, 0, 0, 0, time.UTC)
	name := manifest.ObjectName("my-job-id", ts)
	expected := "jobs/my-job-id/20260331T200000Z/manifest.bsmc"
	if name != expected {
		t.Errorf("ObjectName mismatch: got %q, want %q", name, expected)
	}
}

func TestManifestVersion(t *testing.T) {
	m := manifest.New("job-789", "node-1", `C:\`, "full")
	sealed, _ := m.Seal(testPass)
	reopened, _ := manifest.Open(sealed, testPass)
	if reopened.Version != 1 {
		t.Errorf("expected Version=1, got %d", reopened.Version)
	}
}
