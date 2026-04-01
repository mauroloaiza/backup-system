package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/smcsoluciones/backup-system/agent/internal/compress"
	"github.com/smcsoluciones/backup-system/agent/internal/crypto"
)

// FileEntry describes a single backed-up file within a job run.
type FileEntry struct {
	RelPath    string    `json:"rel_path"`     // relative to backup root
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	SHA256     string    `json:"sha256,omitempty"`
	SDDL       string    `json:"sddl,omitempty"`  // Windows ACL (SDDL)
	ObjectName string    `json:"object_name"`     // destination object path
	Chunks     []Chunk   `json:"chunks,omitempty"` // non-nil for large files split into chunks
	Skipped    bool      `json:"skipped,omitempty"`
}

// Chunk describes one chunk of a large file.
type Chunk struct {
	Index      int    `json:"index"`
	ObjectName string `json:"object_name"`
	Size       int64  `json:"size"`
}

// Manifest is the top-level structure stored alongside a backup job run.
// It is itself compressed + encrypted before being written to the destination.
type Manifest struct {
	Version    int         `json:"version"`
	JobID      string      `json:"job_id"`
	NodeID     string      `json:"node_id"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt time.Time   `json:"finished_at"`
	BackupRoot string      `json:"backup_root"`
	BackupType string      `json:"backup_type"` // "full" | "incremental"
	Files      []FileEntry `json:"files"`

	// Stats
	TotalFiles    int64 `json:"total_files"`
	TotalBytes    int64 `json:"total_bytes"`
	ChangedFiles  int64 `json:"changed_files"`
	ChangedBytes  int64 `json:"changed_bytes"`
	SkippedFiles  int64 `json:"skipped_files"`
	EncryptionAlg string `json:"encryption_alg"` // "AES-256-GCM"
	CompressionAlg string `json:"compression_alg"` // "zstd"
}

const currentVersion = 1

// New returns an initialised Manifest.
func New(jobID, nodeID, backupRoot, backupType string) *Manifest {
	return &Manifest{
		Version:        currentVersion,
		JobID:          jobID,
		NodeID:         nodeID,
		StartedAt:      time.Now().UTC(),
		BackupRoot:     backupRoot,
		BackupType:     backupType,
		EncryptionAlg:  "AES-256-GCM",
		CompressionAlg: "zstd",
	}
}

// Seal compresses then encrypts the manifest and returns the ciphertext blob.
// The same passphrase used for file encryption should be used here.
func (m *Manifest) Seal(passphrase string) ([]byte, error) {
	m.FinishedAt = time.Now().UTC()

	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("manifest: marshal: %w", err)
	}

	// Compress first
	compressed, err := compress.CompressBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("manifest: compress: %w", err)
	}

	// Encrypt
	var buf bytes.Buffer
	if err := crypto.Encrypt(&buf, bytes.NewReader(compressed), passphrase); err != nil {
		return nil, fmt.Errorf("manifest: encrypt: %w", err)
	}
	return buf.Bytes(), nil
}

// Open decrypts and decompresses a sealed manifest blob.
func Open(data []byte, passphrase string) (*Manifest, error) {
	// Decrypt
	var plainBuf bytes.Buffer
	if err := crypto.Decrypt(&plainBuf, bytes.NewReader(data), passphrase); err != nil {
		return nil, fmt.Errorf("manifest: decrypt: %w", err)
	}

	// Decompress
	decompressed, err := compress.DecompressBytes(plainBuf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("manifest: decompress: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(decompressed, &m); err != nil {
		return nil, fmt.Errorf("manifest: unmarshal: %w", err)
	}
	return &m, nil
}

// ObjectName returns the destination object path for the manifest of a given job run.
func ObjectName(jobID string, startedAt time.Time) string {
	ts := startedAt.UTC().Format("20060102T150405Z")
	return fmt.Sprintf("jobs/%s/%s/manifest.bsmc", jobID, ts)
}
