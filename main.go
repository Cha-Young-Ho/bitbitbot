package main

import (
	"fyne.io/fyne/v2/app"

	"gui-app/config"
	"gui-app/services"
	"gui-app/ui"
)

func main() {
	// 애플리케이션 설정 초기화
	cfg := config.NewAppConfig()

	// Fyne 앱 생성
	fyneApp := app.New()

	// 서비스 초기화
	dataService := services.NewDataService()

	// UI 앱 생성
	uiApp := ui.NewApp(fyneApp, cfg, dataService)

	// 애플리케이션 실행
	uiApp.Run()
}
