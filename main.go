package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 설정 초기화
	if err := initConfigSettings(); err != nil {
		return
	}

	// 1. 앱이 처음 시작되면 설정 파일에 접근
	if err := performConfigValidation(); err != nil {
		showInvalidVersionAndExit()
		return
	}

	// Create an instance of the app structure
	app := NewApp()

	// 주기적 설정 검증 시작 (30분마다)
	startPeriodicConfigCheck()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "bitbit",
		Width:            1024,
		Height:           768,
		Assets:           assets,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
