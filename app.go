package main

import (
	"bitbit-app/local_file"
	"bitbit-app/platform"
	"bitbit-app/user"
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// App struct
type App struct {
	ctx              context.Context
	localFileHandler *local_file.Handler
	userHandler      *user.Handler
	platformHandler  *platform.Handler
	wsServer         *platform.WebSocketServer
}

// NewApp creates a new App application struct
func NewApp() *App {
	// 로컬 파일 핸들러 생성
	localFileHandler := local_file.NewHandler()

	// 플랫폼 핸들러 생성
	platformHandler := platform.NewHandler(localFileHandler)

	// 사용자 핸들러 생성
	userHandler := user.NewHandler(localFileHandler, platformHandler)

	// 웹소켓 서버 생성
	wsServer := platform.NewWebSocketServer(platformHandler.GetWorkerManager(), "8080")

	return &App{
		localFileHandler: localFileHandler,
		userHandler:      userHandler,
		platformHandler:  platformHandler,
		wsServer:         wsServer,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 웹소켓 서버 시작
	go func() {
		if err := a.wsServer.Start(); err != nil {
			log.Printf("웹소켓 서버 시작 실패: %v", err)
		}
	}()
}

// LocalFileService 객체
func (a *App) LocalFileService() *local_file.Handler {
	return a.localFileHandler
}

// UserService 객체
func (a *App) UserService() *user.Handler {
	return a.userHandler
}

// PlatformService 객체
func (a *App) PlatformService() *platform.Handler {
	return a.platformHandler
}

// User 관련 메서드들
func (a *App) Login(userID, password string) map[string]interface{} {
	return a.userHandler.Login(userID, password)
}

func (a *App) Register(userID, password string) map[string]interface{} {
	return a.userHandler.Register(userID, password)
}

func (a *App) GetUserInfo(userID string) map[string]interface{} {
	return a.userHandler.GetUserInfo(userID)
}

func (a *App) GetAccountInfo(userID string) map[string]interface{} {
	return a.userHandler.GetAccountInfo(userID)
}

func (a *App) Logout(userID string) map[string]interface{} {
	return a.userHandler.Logout(userID)
}

// Platform 관련 메서드들
func (a *App) GetPlatformInfo(userID string) map[string]interface{} {
	return a.platformHandler.GetPlatformInfo(userID)
}

func (a *App) AddPlatform(userID string, platformName string, name string, accessKey, secretKey, passwordPhrase string) map[string]interface{} {
	return a.platformHandler.AddPlatform(userID, platformName, name, accessKey, secretKey, passwordPhrase)
}

// 파일 관리 API 메서드들
func (a *App) GetLocalFileData(userID string) map[string]interface{} {
	userData := a.localFileHandler.GetUserData(userID)
	if userData == nil {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 데이터를 찾을 수 없습니다.",
		}
	}

	// JSON으로 변환
	jsonData, err := json.MarshalIndent(userData, "", "  ")
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "JSON 변환 중 오류가 발생했습니다.",
		}
	}

	return map[string]interface{}{
		"success": true,
		"data":    string(jsonData),
	}
}

func (a *App) SaveLocalFileData(userID string, jsonData string) map[string]interface{} {
	// JSON 파싱
	var userData local_file.UserData
	if err := json.Unmarshal([]byte(jsonData), &userData); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "잘못된 JSON 형식입니다.",
		}
	}

	// 사용자 ID 검증
	if userData.ID != userID {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 일치하지 않습니다.",
		}
	}

	// 데이터 저장
	if err := a.localFileHandler.SaveUserDataFromJSON(userID, jsonData); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "파일 저장 중 오류가 발생했습니다.",
		}
	}

	// 기존 워커 삭제
	a.platformHandler.RemoveAllWorkers()

	// 새로운 워커 생성
	if err := a.createWorkersForUser(userID, &userData); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "워커 재생성 중 오류가 발생했습니다: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "파일이 성공적으로 저장되었습니다.",
	}
}

func (a *App) RemovePlatform(userID string, platformName string, name string) map[string]interface{} {
	return a.platformHandler.RemovePlatform(userID, platformName, name)
}

func (a *App) UpdatePlatform(userID string, oldPlatformName string, oldName string, newPlatformName string, newName string, accessKey, secretKey, passwordPhrase string) map[string]interface{} {
	return a.platformHandler.UpdatePlatform(userID, oldPlatformName, oldName, newPlatformName, newName, accessKey, secretKey, passwordPhrase)
}

func (a *App) GetAllPlatforms() map[string]interface{} {
	return a.platformHandler.GetAllPlatforms()
}

func (a *App) GetPlatforms(userID string) map[string]interface{} {
	return a.platformHandler.GetPlatforms(userID)
}

func (a *App) AddSellOrder(userID string, orderName string, symbol string, price float64, quantity float64, term float64, platformName string, platformNickName string) map[string]interface{} {
	return a.platformHandler.AddSellOrder(userID, orderName, symbol, price, quantity, term, platformName, platformNickName)
}

func (a *App) GetSellOrders(userID string) map[string]interface{} {
	return a.platformHandler.GetSellOrders(userID)
}

// UpdateSellOrder 예약 매도 주문 수정
func (a *App) UpdateSellOrder(userID string, oldName string, orderName string, symbol string, price float64, quantity float64, term float64, platformName string, platformNickName string) map[string]interface{} {
	return a.platformHandler.UpdateSellOrder(userID, oldName, orderName, symbol, price, quantity, term, platformName, platformNickName)
}

// RemoveSellOrder 예약 매도 주문 삭제
func (a *App) RemoveSellOrder(userID string, orderName string) map[string]interface{} {
	return a.platformHandler.RemoveSellOrder(userID, orderName)
}

// Worker 관련 메서드들
func (a *App) StartWorkerForOrder(userID string, orderName string) map[string]interface{} {
	return a.platformHandler.StartWorkerForOrder(userID, orderName)
}

func (a *App) StopWorkerForOrder(userID string, orderName string) map[string]interface{} {
	return a.platformHandler.StopWorkerForOrder(userID, orderName)
}

func (a *App) GetWorkerStatus(userID string) map[string]interface{} {
	return a.platformHandler.GetWorkerStatus(userID)
}

func (a *App) GetWorkerLogs(userID string, limit int) map[string]interface{} {
	return a.platformHandler.GetWorkerLogs(userID, limit)
}

func (a *App) StartAllWorkersForUser(userID string) map[string]interface{} {
	return a.platformHandler.StartAllWorkersForUser(userID)
}

func (a *App) StopAllWorkersForUser(userID string) map[string]interface{} {
	return a.platformHandler.StopAllWorkersForUser(userID)
}

// 새로운 워커 관리 API 메서드들
func (a *App) GetWorkerInfoByUserID(userID string) map[string]interface{} {
	return a.platformHandler.GetWorkerInfoByUserID(userID)
}

func (a *App) GetWorkerLogsByOrderName(userID string, orderName string, limit int) map[string]interface{} {
	return a.platformHandler.GetWorkerLogsByOrderName(userID, orderName, limit)
}

func (a *App) ClearWorkerLogsByOrderName(userID string, orderName string) map[string]interface{} {
	return a.platformHandler.ClearWorkerLogsByOrderName(userID, orderName)
}

// GetWorkerLogsStream 사용자의 워커 로그를 실시간으로 스트리밍합니다
// (삭제) GetWorkerLogsStream: 웹소켓 통합으로 불필요

// GetOrderLogs 특정 주문의 로그를 반환합니다
// (삭제) GetOrderLogs: 웹소켓 통합으로 불필요

// SubscribeToLogs 로그 구독을 시작합니다
// (삭제) SubscribeToLogs: 웹소켓 통합으로 불필요

// UnsubscribeFromLogs 로그 구독을 해제합니다
// (삭제) UnsubscribeFromLogs: 웹소켓 통합으로 불필요

// GetUnifiedLogs 통합된 로그를 반환합니다
// (삭제) GetUnifiedLogs: 웹소켓 통합으로 불필요

// initializeWorkersForExistingUsers 앱 시작 시 기존 사용자들의 워커를 생성합니다
func (a *App) initializeWorkersForExistingUsers() {
	// 모든 사용자 조회
	users := a.localFileHandler.GetAllUsers()

	failedCount := 0

	for _, user := range users {
		// 각 사용자의 매도 예약 주문에 대한 워커 생성
		if err := a.createWorkersForUser(user.ID, &user); err != nil {
			log.Printf("사용자 워커 초기화 실패: userID=%s, error=%v", user.ID, err)
			failedCount++
		}
	}
}

// createWorkersForUser 사용자의 모든 매도 예약 주문에 대한 워커를 생성합니다
func (a *App) createWorkersForUser(userID string, userData *local_file.UserData) error {
	failedCount := 0

	// 각 매도 예약 주문에 대해 워커 생성
	for _, order := range userData.SellOrderList {
		// 플랫폼 키 찾기
		var platformKey local_file.PlatformKey
		found := false

		for _, key := range userData.PlatformKeyList {
			if key.PlatformName == order.Platform && key.Name == order.PlatformNickName {
				platformKey = key
				found = true
				break
			}
		}

		if !found {
			failedCount++
			continue
		}

		if err := a.platformHandler.CreateWorkerForOrder(order, userID, platformKey.PlatformAccessKey, platformKey.PlatformSecretKey, platformKey.PasswordPhrase); err != nil {
			log.Printf("워커 생성 실패: order=%s, error=%v", order.Name, err)
			failedCount++
			continue
		}
	}

	if failedCount > 0 {
		return fmt.Errorf("일부 워커 생성 실패: %d개", failedCount)
	}

	return nil
}
