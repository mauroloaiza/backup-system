//go:build !windows

package notify

// WriteEventLog is a no-op on non-Windows platforms.
func WriteEventLog(eventType, message string) {}
