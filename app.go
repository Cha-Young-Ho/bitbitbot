package main

import (
	"bitbit-app/platform"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 간단한 애플리케이션
type App struct {
	handler *platform.Handler
	ctx     context.Context
}

// NewApp 새로운 애플리케이션 생성
func NewApp() *App {
	return &App{
		handler: platform.NewHandler(),
	}
}

// SetWorkerConfig 워커 설정
func (a *App) SetWorkerConfig(exchange, accessKey, secretKey, passwordPhrase, requestInterval, symbol, sellAmount, sellPrice string) map[string]interface{} {
	return a.handler.SetWorkerConfig(exchange, accessKey, secretKey, passwordPhrase, requestInterval, symbol, sellAmount, sellPrice)
}

// GetWorkerConfig 워커 설정 조회
func (a *App) GetWorkerConfig() map[string]interface{} {
	return a.handler.GetWorkerConfig()
}

// StartWorker 워커 시작
func (a *App) StartWorker() map[string]interface{} {
	return a.handler.StartWorker()
}

// StopWorker 워커 중지
func (a *App) StopWorker() map[string]interface{} {
	return a.handler.StopWorker()
}

// GetWorkerStatus 워커 상태 조회
func (a *App) GetWorkerStatus() map[string]interface{} {
	return a.handler.GetWorkerStatus()
}

// GetLogs 로그 조회
func (a *App) GetLogs(limit int) map[string]interface{} {
	return a.handler.GetLogs(limit)
}

// ClearLogs 로그 초기화
func (a *App) ClearLogs() map[string]interface{} {
	return a.handler.ClearLogs()
}

// CheckVersion 버전 체크
func (a *App) CheckVersion() map[string]interface{} {
	return a.handler.CheckVersion()
}

// OnStartup 애플리케이션 시작 시 호출
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	log.Println("애플리케이션이 시작되었습니다.")

	// 주기적 버전 체크 시작
	go a.startVersionCheck()
}

// OnShutdown 애플리케이션 종료 시 호출
func (a *App) OnShutdown(ctx context.Context) {
	log.Println("애플리케이션이 종료됩니다.")
	a.handler.Cleanup()
}

// startVersionCheck 주기적 버전 체크
func (a *App) startVersionCheck() {
	ticker := time.NewTicker(30 * time.Second) // 30초마다 체크
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.performVersionCheck(); err != nil {
				log.Printf("버전 체크 실패: %v", err)
			}
		}
	}
}

// performVersionCheck 버전 체크 수행
func (a *App) performVersionCheck() error {
	result := a.CheckVersion()
	if result == nil {
		return fmt.Errorf("버전 체크 결과가 nil입니다")
	}

	isLatest, ok := result["isLatest"].(bool)
	if !ok {
		return fmt.Errorf("버전 체크 결과 형식이 잘못되었습니다")
	}

	if !isLatest {
		latestVersion, _ := result["latestVersion"].(string)
		if latestVersion == "" {
			latestVersion = "알 수 없음"
		}

		// 새 버전이 있으면 프론트엔드에 알림
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "업데이트 알림",
			Message: fmt.Sprintf("새 버전이 있습니다: %s", latestVersion),
		})
	}

	return nil
}
