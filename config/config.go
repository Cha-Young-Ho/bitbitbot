package config

import (
	"os"
	"path/filepath"
)

// AppConfig 애플리케이션 설정
type AppConfig struct {
	DataFile     string
	WindowWidth  float32
	WindowHeight float32
	AppName      string
	Version      string
}

// NewAppConfig 새로운 설정을 생성합니다
func NewAppConfig() *AppConfig {
	homeDir, err := os.UserHomeDir()
	dataFile := "bitcoin_trader_data.json"
	if err == nil {
		dataFile = filepath.Join(homeDir, "bitcoin_trader_data.json")
	}

	return &AppConfig{
		DataFile:     dataFile,
		WindowWidth:  1600,
		WindowHeight: 1000,
		AppName:      "Bitcoin Trader Professional",
		Version:      "3.0.0",
	}
}

// GetDataFilePath 데이터 파일 경로를 반환합니다
func (c *AppConfig) GetDataFilePath() string {
	return c.DataFile
}

// GetWindowSize 윈도우 크기를 반환합니다
func (c *AppConfig) GetWindowSize() (float32, float32) {
	return c.WindowWidth, c.WindowHeight
}

// GetAppInfo 앱 정보를 반환합니다
func (c *AppConfig) GetAppInfo() (string, string) {
	return c.AppName, c.Version
}
