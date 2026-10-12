package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := &App{}
	err := wails.Run(&options.App{
		Title:  "Git Power",
		Width:  1440,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 15, B: 25, A: 0},
		Mac: &mac.Options{
			// FullSizeContent makes the webview cover the title bar area, and
			// TitlebarAppearsTransparent drops its opaque fill, so the glass
			// surface reaches the very top edge instead of stopping under a
			// white native title bar. The traffic lights stay visible on top.
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				FullSizeContent:            true,
				HideToolbarSeparator:       true,
			},
			WindowIsTranslucent:  true,
			WebviewIsTransparent: true,
		},
		Windows: &windows.Options{
			WindowIsTranslucent: true,
			BackdropType:        windows.Acrylic,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
