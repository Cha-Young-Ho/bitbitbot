package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Config 설정 정보
type Config struct {
	MainVer string `json:"mainVer"` // 선택적 업데이트 버전
	MinVer  string `json:"minVer"`  // 필수 업데이트 버전
}

var (
	configUrl   string // 빌드 시 주입되는 설정 파일 URL
	config      *Config
	Version     string // 빌드 시 주입되는 버전 정보
	Environment string // 빌드 시 주입되는 환경 정보 (prod, dev 등)
)

// initConfigSettings 빌드 시 주입된 설정을 확인합니다
func initConfigSettings() error {
	if configUrl == "" {
		// 개발 모드에서는 기본 설정 URL 사용
		configUrl = "https://test-bucket.s3.ap-northeast-2.amazonaws.com/dev/config.json"
		log.Printf("개발 모드: 기본 설정 URL 사용 - %s", configUrl)
	}

	log.Printf("설정 초기화 완료: URL=%s", configUrl)
	return nil
}

// loadConfigFromS3 S3에서 설정 파일을 읽어옵니다
func loadConfigFromS3() error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(configUrl)
	if err != nil {
		return fmt.Errorf("HTTP 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http 상태 코드 오류: %d", resp.StatusCode)
	}

	var newConfig Config
	if err := json.NewDecoder(resp.Body).Decode(&newConfig); err != nil {
		return fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	config = &newConfig
	log.Printf("S3 설정 로드 완료: 버전=%s, 환경=%s", Version, Environment)
	return nil
}

// CheckVersionUpdate 버전 업데이트를 확인합니다
func CheckVersionUpdate() error {
	if err := loadConfigFromS3(); err != nil {
		return err
	}

	if config == nil {
		return fmt.Errorf("설정이 로드되지 않았습니다")
	}

	log.Printf("버전 체크: 현재=%s, 최소=%s, 메인=%s", Version, config.MinVer, config.MainVer)
	return nil
}
