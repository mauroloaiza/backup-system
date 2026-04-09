//go:build windows

package notify

import (
	"fmt"

	"golang.org/x/sys/windows/svc/eventlog"
)

const sourceName = "BackupSMC"

// WriteEventLog writes a message to the Windows Application Event Log.
// eventType: "info" | "warning" | "error"
func WriteEventLog(eventType, message string) {
	// Try to open existing source; create if not registered
	log, err := eventlog.Open(sourceName)
	if err != nil {
		// Register the event source (only works if we have admin rights)
		_ = eventlog.InstallAsEventCreate(sourceName, eventlog.Info|eventlog.Warning|eventlog.Error)
		log, err = eventlog.Open(sourceName)
		if err != nil {
			return // silently ignore if event log is unavailable
		}
	}
	defer log.Close()

	msg := fmt.Sprintf("BackupSMC: %s", message)
	switch eventType {
	case "warning":
		_ = log.Warning(1, msg)
	case "error":
		_ = log.Error(1, msg)
	default:
		_ = log.Info(1, msg)
	}
}
