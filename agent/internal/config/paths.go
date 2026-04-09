// Package config — runtime paths for agent state files.
package config

import (
	"os"
	"path/filepath"
)

// DataDir returns the persistent data directory for agent state
// (%ProgramData%\BackupSMC on Windows, /var/lib/backupsmc on Linux).
// The directory is created if it does not exist.
func DataDir() string {
	var base string
	if pd := os.Getenv("PROGRAMDATA"); pd != "" {
		base = filepath.Join(pd, "BackupSMC")
	} else {
		base = "/var/lib/backupsmc"
	}
	dir := filepath.Join(base, "state")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// CachePath returns the incremental-scan cache file path for a given source.
// Uses a sanitized form of the source path as the filename to avoid collisions.
func CachePath(sourcePath string) string {
	safe := make([]byte, len(sourcePath))
	for i, c := range []byte(sourcePath) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			safe[i] = c
		default:
			safe[i] = '_'
		}
	}
	return filepath.Join(DataDir(), "cache_"+string(safe)+".json")
}
