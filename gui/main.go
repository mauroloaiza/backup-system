package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "BackupSMC",
		Width:            1200,
		Height:           750,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 15, G: 22, B: 35, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		// Hide to tray instead of closing
		OnBeforeClose: func(ctx context.Context) bool {
			wailsruntime.WindowHide(ctx)
			return true // true = cancel the close
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
		},
		EnableDefaultContextMenu: false,
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
