package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Config 설정 정보
type Config struct {
	Running   string   `json:"running"`   // "on", "off", "all"
	WhiteList []string `json:"whiteList"` // 허용된 사용자 ID 목록
	MainVer   string   `json:"mainVer"`   // 최소 요구 버전
}

var (
	configUrl string // 빌드 시 주입되는 설정 파일 URL
	config    *Config
	Version   string // 빌드 시 주입되는 버전 정보
)

// initConfigSettings 빌드 시 주입된 설정을 확인합니다
func initConfigSettings() error {
	if configUrl == "" {
		return fmt.Errorf("설정 URL이 빌드 시 주입되지 않았습니다: -ldflags로 configUrl를 설정해주세요")
	}

	log.Printf("설정 초기화 완료: URL=%s", configUrl)
	return nil
}

// loadConfig 설정 파일을 읽어옵니다 (타임아웃 포함)
func loadConfig() error {
	// 30초 타임아웃으로 설정 파일 읽기
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configUrl, nil)
	if err != nil {
		return fmt.Errorf("http 요청 생성 실패: %v", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http 요청 실패: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http 상태 코드 오류: %d", resp.StatusCode)
	}
	var cfg Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return fmt.Errorf("json 파싱 실패: %v", err)
	}
	config = &cfg
	log.Printf("설정 로드 완료: Running=%s, MainVer=%s", config.Running, config.MainVer)
	return nil
}

// checkProgramStatus 프로그램 상태를 체크합니다
func checkProgramStatus() error {
	if config == nil {
		return fmt.Errorf("설정이 로드되지 않았습니다")
	}

	// 버전 체크 먼저 수행
	if err := checkVersion(); err != nil {
		return err
	}

	// Running 상태 체크
	if config.Running == "off" {
		return fmt.Errorf("invalid request")
	}

	return nil
}

// checkVersion 버전을 체크합니다
func checkVersion() error {
	if config.MainVer == "" {
		return nil // 버전 체크가 설정되지 않은 경우
	}

	// 현재 버전 (빌드 시 설정됨)
	currentVersion := getVersion()

	// 버전 비교 로직 (간단한 문자열 비교)
	if currentVersion < config.MainVer {
		return fmt.Errorf("invalid version")
	}

	return nil
}

// checkUserAccess 사용자 접근 권한을 체크합니다
func checkUserAccess(userID string) error {
	if config == nil {
		return fmt.Errorf("설정이 로드되지 않았습니다")
	}

	// Running이 "all"이면 모든 사용자 허용 (whiteList 검증 하지 않음)
	if config.Running == "all" {
		return nil
	}

	// Running이 "on"일 때만 whiteList 체크
	if config.Running == "on" {
		// WhiteList 체크
		for _, allowedID := range config.WhiteList {
			if allowedID == userID {
				return nil
			}
		}
		return fmt.Errorf("Invalid Account")
	}

	// Running이 "off"이면 접근 거부
	return fmt.Errorf("Invalid Account")
}

// getVersion 현재 버전을 반환합니다 (빌드 시 주입됨)
func getVersion() string {
	// 빌드 시 -ldflags로 주입되는 버전 정보
	if Version != "" {
		return Version
	}
	return "1.0.0" // 기본값
}

// showInvalidVersionAndExit Invalid Version 메시지를 표시하고 종료합니다
func showInvalidVersionAndExit() {
	log.Printf("Invalid Version")
	fmt.Println("Invalid Version")
	time.Sleep(30 * time.Second)
	// 프로그램 종료
	// 실제로는 Wails 앱에서 처리해야 하므로 로그만 남김
}

// startPeriodicConfigCheck 주기적으로 설정을 체크하는 고루틴을 시작합니다
func startPeriodicConfigCheck() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute) // 30분마다
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := performConfigValidation(); err != nil {
					log.Printf("주기적 설정 검증 실패: %v", err)
					showInvalidVersionAndExit()
					return
				}
				log.Printf("주기적 설정 검증 성공")
			}
		}
	}()
}

// performConfigValidation 설정을 검증합니다 (초기 로드 + 주기적 체크용)
func performConfigValidation() error {
	// 설정 다시 로드
	if err := loadConfig(); err != nil {
		return fmt.Errorf("설정 로드 실패: %v", err)
	}

	// 프로그램 상태 체크
	if err := checkProgramStatus(); err != nil {
		return fmt.Errorf("프로그램 상태 체크 실패: %v", err)
	}

	return nil
}
