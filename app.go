package main

import (
	"bitbit-app/local_file"
	"bitbit-app/platform"
	"bitbit-app/user"
	"context"
)

// App struct
type App struct {
	ctx              context.Context
	localFileHandler *local_file.Handler
	userHandler      *user.Handler
	platformHandler  *platform.Handler
}

// NewApp creates a new App application struct
func NewApp() *App {
	// 로컬 파일 핸들러 생성
	localFileHandler := local_file.NewHandler()

	// 사용자 핸들러 생성
	userHandler := user.NewHandler(localFileHandler)

	// 플랫폼 핸들러 생성
	platformHandler := platform.NewHandler(localFileHandler)

	return &App{
		localFileHandler: localFileHandler,
		userHandler:      userHandler,
		platformHandler:  platformHandler,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
