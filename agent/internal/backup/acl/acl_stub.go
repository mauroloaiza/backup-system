//go:build !windows

package acl

// GetSDDL is a no-op on non-Windows platforms.
func GetSDDL(path string) (string, error) {
	return "", nil
}

// SetSDDL is a no-op on non-Windows platforms.
func SetSDDL(path, sddl string) error {
	return nil
}
