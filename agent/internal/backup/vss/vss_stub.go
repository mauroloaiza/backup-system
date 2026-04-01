//go:build !windows

package vss

import "errors"

var errNotSupported = errors.New("vss: VSS is only supported on Windows")

// Snapshot is a stub for non-Windows platforms.
type Snapshot struct {
	ShadowID   string
	ShadowPath string
	Volume     string
}

// Create always returns an error on non-Windows platforms.
func Create(volume string) (*Snapshot, error) {
	return nil, errNotSupported
}

// Delete is a no-op on non-Windows platforms.
func Delete(shadowID string) error {
	return errNotSupported
}

// TranslatePath returns the original path unchanged (stub).
func (s *Snapshot) TranslatePath(origPath string) string {
	return origPath
}
