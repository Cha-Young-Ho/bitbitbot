package main

import (
	"bitbit-app/local_file"
	"bitbit-app/platform"
	"bitbit-app/user"
	"context"
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

	// 앱 시작 시 기존 사용자들의 워커 생성
	go a.initializeWorkersForExistingUsers()
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

func (a *App) AddPlatform(userID string, platformName string, name string, accessKey, secretKey string) map[string]interface{} {
	return a.platformHandler.AddPlatform(userID, platformName, name, accessKey, secretKey)
}

func (a *App) RemovePlatform(userID string, platformName string, name string) map[string]interface{} {
	return a.platformHandler.RemovePlatform(userID, platformName, name)
}

func (a *App) UpdatePlatform(userID string, oldPlatformName string, oldName string, newPlatformName string, newName string, accessKey, secretKey string) map[string]interface{} {
	return a.platformHandler.UpdatePlatform(userID, oldPlatformName, oldName, newPlatformName, newName, accessKey, secretKey)
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
func (a *App) GetWorkerLogsStream(userID string) map[string]interface{} {
	return a.platformHandler.GetWorkerLogsStream(userID)
}

// GetOrderLogs 특정 주문의 로그를 반환합니다
func (a *App) GetOrderLogs(userID string, orderName string) map[string]interface{} {
	return a.platformHandler.GetOrderLogs(userID, orderName)
}

// SubscribeToLogs 로그 구독을 시작합니다
func (a *App) SubscribeToLogs(userID string) map[string]interface{} {
	return a.platformHandler.SubscribeToLogs(userID)
}

// UnsubscribeFromLogs 로그 구독을 해제합니다
func (a *App) UnsubscribeFromLogs(userID string) map[string]interface{} {
	return a.platformHandler.UnsubscribeFromLogs(userID)
}

// GetUnifiedLogs 통합된 로그를 반환합니다
func (a *App) GetUnifiedLogs(userID string) map[string]interface{} {
	return a.platformHandler.GetUnifiedLogs(userID)
}

// initializeWorkersForExistingUsers 앱 시작 시 기존 사용자들의 워커를 생성합니다
func (a *App) initializeWorkersForExistingUsers() {
	log.Printf("기존 사용자 워커 초기화 시작")

	// 모든 사용자 조회
	users := a.localFileHandler.GetAllUsers()

	initializedCount := 0
	failedCount := 0

	for _, user := range users {
		// 각 사용자의 매도 예약 주문에 대한 워커 생성
		if err := a.createWorkersForUser(user.ID, &user); err != nil {
			log.Printf("사용자 워커 초기화 실패: userID=%s, error=%v", user.ID, err)
			failedCount++
		} else {
			log.Printf("사용자 워커 초기화 완료: userID=%s", user.ID)
			initializedCount++
		}
	}

	log.Printf("기존 사용자 워커 초기화 완료: 성공=%d, 실패=%d", initializedCount, failedCount)
}

// createWorkersForUser 사용자의 모든 매도 예약 주문에 대한 워커를 생성합니다
func (a *App) createWorkersForUser(userID string, userData *local_file.UserData) error {
	log.Printf("사용자 워커 생성 시작: userID=%s, 주문 수=%d", userID, len(userData.SellOrderList))

	createdCount := 0
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
			log.Printf("플랫폼 키를 찾을 수 없음: order=%s, platform=%s, nickname=%s",
				order.Name, order.Platform, order.PlatformNickName)
			failedCount++
			continue
		}

		// 워커 생성 및 시작
		if err := a.platformHandler.CreateWorkerForOrder(order, userID, platformKey.PlatformAccessKey, platformKey.PlatformSecretKey); err != nil {
			log.Printf("워커 생성 실패: order=%s, error=%v", order.Name, err)
			failedCount++
			continue
		}

		log.Printf("워커 생성 및 시작 완료: order=%s, platform=%s", order.Name, order.Platform)
		createdCount++
	}

	log.Printf("사용자 워커 생성 완료: userID=%s, 생성=%d, 실패=%d", userID, createdCount, failedCount)

	if failedCount > 0 {
		return fmt.Errorf("일부 워커 생성 실패: %d개", failedCount)
	}

	return nil
}
