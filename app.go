package main

import (
	"bitbit-app/platform"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppVersionChecker VersionChecker 인터페이스 구현체
type AppVersionChecker struct{}

// CheckVersionUpdate S3에서 설정을 로드합니다
func (a *AppVersionChecker) CheckVersionUpdate() error {
	return CheckVersionUpdate()
}

// CheckRunningStatus running 상태를 확인합니다
func (a *AppVersionChecker) CheckRunningStatus() error {
	return CheckRunningStatus()
}

// CompareVersions 버전을 비교합니다
func (a *AppVersionChecker) CompareVersions() (bool, bool, error) {
	return CompareVersions()
}

// GetConfig 설정을 반환합니다
func (a *AppVersionChecker) GetConfig() interface{} {
	config := GetConfig()
	if config == nil {
		return nil
	}
	
	// Config 구조체를 map으로 변환
	return map[string]interface{}{
		"running":     config.Running,
		"whiteList":   config.WhiteList,
		"mainVer":     config.MainVer,
		"minVer":      config.MinVer,
		"forceUpdate": config.ForceUpdate,
	}
}

// GetCurrentVersion 현재 버전을 반환합니다
func (a *AppVersionChecker) GetCurrentVersion() string {
	return Version
}

// App 간단한 애플리케이션
type App struct {
	handler *platform.Handler
	ctx     context.Context
}

// NewApp 새로운 애플리케이션 생성
func NewApp() *App {
	app := &App{
		handler: platform.NewHandler(),
	}
	
	// VersionChecker 설정
	app.handler.SetVersionChecker(&AppVersionChecker{})
	
	return app
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

	// 초기 버전 체크 및 상태 확인
	go a.initialVersionCheck()

	// 지속적인 버전 체크 시작 (30분마다)
	go a.continuousVersionCheck()

	// 지속적인 상태 체크 시작 (30분마다)
	go a.continuousStatusCheck()
}

// OnShutdown 애플리케이션 종료 시 호출
func (a *App) OnShutdown(ctx context.Context) {
	log.Println("애플리케이션이 종료됩니다.")
	a.handler.Cleanup()
}

// initialVersionCheck 초기 버전 체크 및 상태 확인
func (a *App) initialVersionCheck() {
	// 최대 3번 재시도, 30분 간격
	maxRetries := 3
	retryInterval := 30 * time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("초기 버전 체크 시도 %d/%d", attempt, maxRetries)
		
		if err := a.performInitialCheck(); err != nil {
			log.Printf("초기 체크 실패 (시도 %d/%d): %v", attempt, maxRetries, err)
			
			if attempt < maxRetries {
				log.Printf("%v 후 재시도합니다...", retryInterval)
				time.Sleep(retryInterval)
			} else {
				log.Printf("최대 재시도 횟수 초과. 프로그램을 종료합니다.")
				a.showErrorAndExit("비정상 종료입니다.")
				return
			}
		} else {
			log.Printf("초기 버전 체크 완료")
			return
		}
	}
}

// performInitialCheck 초기 체크 수행
func (a *App) performInitialCheck() error {
	result := a.CheckVersion()
	if result == nil {
		return fmt.Errorf("버전 체크 결과가 nil입니다")
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		message, _ := result["message"].(string)
		return fmt.Errorf("버전 체크 실패: %s", message)
	}

	// running 상태 체크
	config, ok := result["config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("설정 정보를 가져올 수 없습니다")
	}

	running, ok := config["running"].(string)
	if !ok {
		return fmt.Errorf("running 상태를 확인할 수 없습니다")
	}

	// running이 "off"인 경우 프로그램 종료
	if running == "off" {
		a.showErrorAndExit("비정상 종료입니다.")
		return fmt.Errorf("프로그램 실행이 차단되었습니다")
	}

	// 버전 업데이트 필요 여부 확인
	isUpdateNeeded, ok := result["isUpdateNeeded"].(bool)
	if !ok {
		return fmt.Errorf("업데이트 필요 여부를 확인할 수 없습니다")
	}

	if isUpdateNeeded {
		isForceUpdate, _ := result["isForceUpdate"].(bool)
		
		if isForceUpdate {
			// 강제 업데이트 (minVer 미달) - 워커 중지 후 5초 후 자동 종료
			log.Printf("강제 업데이트 필요 (minVer 미달): 워커를 중지하고 5초 후 종료합니다.")
			a.stopAllWorkers()
			
			// 강제 업데이트 다이얼로그 표시
			a.showForceUpdateDialog(result)
			
			// 5초 후 자동 종료
			go func() {
				time.Sleep(5 * time.Second)
				log.Printf("5초 후 자동으로 프로그램을 종료합니다.")
				os.Exit(1)
			}()
		} else {
			// 권장 업데이트 (mainVer 미달, minVer 이상)
			log.Printf("권장 업데이트 (mainVer 미달, minVer 이상): 업데이트 다이얼로그를 표시합니다.")
			a.showRecommendedUpdateDialog(result)
		}
	}

	return nil
}

// continuousVersionCheck 지속적인 버전 체크 (30분마다)
func (a *App) continuousVersionCheck() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	log.Println("지속적인 버전 체크가 시작되었습니다 (30분 간격)")

	for {
		select {
		case <-ticker.C:
			if err := a.performVersionCheck(); err != nil {
				log.Printf("지속적 버전 체크 실패: %v", err)
			}
		}
	}
}

// continuousStatusCheck 지속적인 상태 체크 (30분마다)
func (a *App) continuousStatusCheck() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	log.Println("지속적인 상태 체크가 시작되었습니다 (30분 간격)")

	for {
		select {
		case <-ticker.C:
			if err := a.performStatusCheck(); err != nil {
				log.Printf("지속적 상태 체크 실패: %v", err)
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

	// 버전 업데이트 필요 여부 확인
	isUpdateNeeded, ok := result["isUpdateNeeded"].(bool)
	if !ok {
		return fmt.Errorf("업데이트 필요 여부를 확인할 수 없습니다")
	}

	if isUpdateNeeded {
		isForceUpdate, _ := result["isForceUpdate"].(bool)
		
		if isForceUpdate {
			// 강제 업데이트 (minVer 미달) - 워커 중지 후 5초 후 자동 종료
			log.Printf("강제 업데이트 필요 (minVer 미달): 워커를 중지하고 5초 후 종료합니다.")
			a.stopAllWorkers()
			
			// 강제 업데이트 다이얼로그 표시
			a.showForceUpdateDialog(result)
			
			// 5초 후 자동 종료
			go func() {
				time.Sleep(5 * time.Second)
				log.Printf("5초 후 자동으로 프로그램을 종료합니다.")
				os.Exit(1)
			}()
		} else {
			// 권장 업데이트 (mainVer 미달, minVer 이상)
			log.Printf("권장 업데이트 (mainVer 미달, minVer 이상): 업데이트 다이얼로그를 표시합니다.")
			a.showRecommendedUpdateDialog(result)
		}
	} else {
		log.Printf("버전 체크 완료: 업데이트가 필요하지 않습니다.")
	}

	return nil
}

// performStatusCheck 상태 체크 수행
func (a *App) performStatusCheck() error {
	result := a.CheckVersion()
	if result == nil {
		return fmt.Errorf("상태 체크 결과가 nil입니다")
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		return fmt.Errorf("상태 체크 실패")
	}

	// running 상태 체크
	config, ok := result["config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("설정 정보를 가져올 수 없습니다")
	}

	running, ok := config["running"].(string)
	if !ok {
		return fmt.Errorf("running 상태를 확인할 수 없습니다")
	}

	// running이 "off"인 경우 프로그램 종료
	if running == "off" {
		log.Printf("상태 체크에서 running이 'off'로 변경됨. 프로그램을 종료합니다.")
		a.showErrorAndExit("비정상 종료입니다.")
		return fmt.Errorf("프로그램 실행이 차단되었습니다")
	}

	log.Printf("상태 체크 완료: running=%s", running)
	return nil
}

// stopAllWorkers 모든 워커를 중지합니다
func (a *App) stopAllWorkers() {
	log.Println("모든 워커를 중지합니다...")
	result := a.StopWorker()
	if result != nil {
		success, ok := result["success"].(bool)
		if ok && success {
			log.Println("모든 워커가 성공적으로 중지되었습니다.")
		} else {
			log.Println("워커 중지 중 오류가 발생했습니다.")
		}
	}
}

// showErrorAndExit 에러 메시지를 보여주고 프로그램 종료
func (a *App) showErrorAndExit(message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.ErrorDialog,
		Title:   "오류",
		Message: message,
	})

	// 즉시 프로그램 종료 (대기 없음)
	log.Printf("프로그램을 종료합니다.")
	os.Exit(1)
}

// showForceUpdateDialog 강제 업데이트 다이얼로그
func (a *App) showForceUpdateDialog(result map[string]interface{}) {
	latestVersion, _ := result["latestVersion"].(string)
	currentVersion, _ := result["currentVersion"].(string)
	
	// 강제 업데이트 다이얼로그 (업데이트 또는 종료)
	button, _ := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.WarningDialog,
		Title:   "업데이트 필요",
		Message: fmt.Sprintf("현재 버전: %s\n최신 버전: %s\n\n보안 및 안정성을 위해 업데이트가 필수입니다.\n\n업데이트하시겠습니까?\n\n올바른 버전으로 실행해주세요.", currentVersion, latestVersion),
		Buttons: []string{"업데이트", "종료"},
	})

	if button == "종료" {
		// 즉시 프로그램 종료
		log.Printf("사용자가 종료를 선택했습니다. 프로그램을 종료합니다.")
		os.Exit(1)
	} else {
		// TODO: 실제 업데이트 로직 구현
		log.Printf("업데이트 선택됨: %s", latestVersion)
	}
}

// showRecommendedUpdateDialog 권장 업데이트 다이얼로그
func (a *App) showRecommendedUpdateDialog(result map[string]interface{}) {
	latestVersion, _ := result["latestVersion"].(string)
	currentVersion, _ := result["currentVersion"].(string)
	
	// 권장 업데이트 다이얼로그 (업데이트 또는 다음에 하기)
	button, _ := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "업데이트 권장",
		Message: fmt.Sprintf("현재 버전: %s\n최신 버전: %s\n\n새로운 기능과 개선사항이 포함된 버전이 있습니다.\n\n업데이트하시겠습니까?", currentVersion, latestVersion),
		Buttons: []string{"업데이트", "다음에 하기"},
	})

	if button == "업데이트" {
		// TODO: 실제 업데이트 로직 구현
		log.Printf("업데이트 선택됨: %s", latestVersion)
	} else {
		log.Printf("업데이트를 다음에 하기로 선택")
	}
}
