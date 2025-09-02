package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Config S3 설정 정보
type Config struct {
	Running      string   `json:"running"`      // all, target, off
	WhiteList    []string `json:"whiteList"`    // 화이트리스트 사용자
	MainVer      string   `json:"mainVer"`      // 선택적 업데이트 버전
	MinVer       string   `json:"minVer"`       // 필수 업데이트 버전
	ForceUpdate  bool     `json:"forceUpdate"`  // 강제 업데이트 여부
}

// 빌드 시 주입되는 변수들
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

// CheckRunningStatus running 상태를 확인합니다
func CheckRunningStatus() error {
	if config == nil {
		return fmt.Errorf("설정이 로드되지 않았습니다")
	}

	switch config.Running {
	case "all":
		log.Printf("프로그램 실행 허용: running=%s", config.Running)
		return nil
	case "target":
		log.Printf("타겟 사용자만 실행 허용: running=%s", config.Running)
		// TODO: 화이트리스트 체크 로직 추가
		return nil
	case "off":
		return fmt.Errorf("프로그램 실행이 차단되었습니다: running=%s", config.Running)
	default:
		return fmt.Errorf("알 수 없는 running 상태: %s", config.Running)
	}
}

// CompareVersions 버전을 비교합니다
func CompareVersions() (bool, bool, error) {
	if config == nil {
		return false, false, fmt.Errorf("설정이 로드되지 않았습니다")
	}

	// 현재 버전이 mainVer보다 낮은지 확인
	isMainUpdateNeeded := compareVersion(Version, config.MainVer) < 0
	
	// 현재 버전이 minVer보다 낮은지 확인
	isMinUpdateNeeded := compareVersion(Version, config.MinVer) < 0

	log.Printf("버전 비교 결과: 현재=%s, mainVer=%s, minVer=%s, mainUpdate=%v, minUpdate=%v", 
		Version, config.MainVer, config.MinVer, isMainUpdateNeeded, isMinUpdateNeeded)

	return isMainUpdateNeeded, isMinUpdateNeeded, nil
}

// compareVersion 버전 문자열을 비교합니다 (1.0.0 형식)
func compareVersion(v1, v2 string) int {
	// 간단한 버전 비교 로직
	// 실제로는 더 정교한 버전 비교가 필요할 수 있습니다
	if v1 == v2 {
		return 0
	}
	
	// 버전이 같지 않으면 문자열 비교로 처리
	if v1 < v2 {
		return -1
	}
	return 1
}

// GetConfig 설정을 반환합니다
func GetConfig() *Config {
	return config
}

