package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed res/icon.png
var iconPNG []byte

func main() {
	myApp := NewApp()

	app := application.New(application.Options{
		Name:        "GORO-Patcher",
		Description: "Ragnarok Online GRF Patcher",
		Icon:        iconPNG,
		Services: []application.Service{
			application.NewService(myApp),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	myApp.SetApp(app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "GORO Patcher",
		Width:     800,
		Height:    600,
		Frameless: true,
		URL:       "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
