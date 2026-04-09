package main

import (
	_ "embed"
	"os"
	"time"

	"github.com/getlantern/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIconBytes []byte

// startTray launches the system tray icon. Must be called after app.ctx is set.
func (a *App) startTray() {
	go systray.Run(a.onTrayReady, func() {})
}

func (a *App) onTrayReady() {
	systray.SetIcon(trayIconBytes)
	systray.SetTitle("BackupSMC")
	systray.SetTooltip("BackupSMC Agent")

	mShow   := systray.AddMenuItem("Open BackupSMC", "Show the main window")
	mBackup := systray.AddMenuItem("Backup Now", "Run a backup immediately")
	systray.AddSeparator()
	mStatus := systray.AddMenuItem("Service: checking…", "Current service state")
	mStatus.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit BackupSMC GUI")

	// Poll service status every 15 seconds
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for ; true; <-ticker.C {
			s := a.GetServiceStatus()
			mStatus.SetTitle("Service: " + s.Status)
		}
	}()

	// Handle tray menu clicks
	for {
		select {
		case <-mShow.ClickedCh:
			if a.ctx != nil {
				wailsruntime.WindowShow(a.ctx)
			}
		case <-mBackup.ClickedCh:
			a.RunBackupNow()
		case <-mQuit.ClickedCh:
			systray.Quit()
			os.Exit(0)
		}
	}
}
