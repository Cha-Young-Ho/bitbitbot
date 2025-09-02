package platform

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// S3Service S3 관련 작업을 담당하는 서비스
type S3Service struct {
	configURL    string
	httpClient   *http.Client
	retryConfig  *RetryConfig
}

// RetryConfig 재시도 설정
type RetryConfig struct {
	MaxRetries    int           `json:"maxRetries"`
	RetryInterval time.Duration `json:"retryInterval"`
	Timeout       time.Duration `json:"timeout"`
}

// NewS3Service 새로운 S3 서비스 생성
func NewS3Service(configURL string) *S3Service {
	return &S3Service{
		configURL: configURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		retryConfig: &RetryConfig{
			MaxRetries:    3,
			RetryInterval: 30 * time.Minute,
			Timeout:       10 * time.Second,
		},
	}
}

// SetRetryConfig 재시도 설정 변경
func (s *S3Service) SetRetryConfig(config *RetryConfig) {
	s.retryConfig = config
}

// LoadConfigWithRetry 재시도 로직을 포함한 설정 로드
func (s *S3Service) LoadConfigWithRetry() (*Config, error) {
	var lastErr error
	
	for attempt := 1; attempt <= s.retryConfig.MaxRetries; attempt++ {
		log.Printf("S3 설정 로드 시도 %d/%d", attempt, s.retryConfig.MaxRetries)
		
		config, err := s.loadConfig()
		if err == nil {
			log.Printf("S3 설정 로드 성공 (시도 %d)", attempt)
			return config, nil
		}
		
		lastErr = err
		log.Printf("S3 설정 로드 실패 (시도 %d/%d): %v", attempt, s.retryConfig.MaxRetries, err)
		
		if attempt < s.retryConfig.MaxRetries {
			log.Printf("%v 후 재시도합니다...", s.retryConfig.RetryInterval)
			time.Sleep(s.retryConfig.RetryInterval)
		}
	}
	
	return nil, fmt.Errorf("최대 재시도 횟수 초과. 마지막 오류: %v", lastErr)
}

// loadConfig S3에서 설정 파일을 읽어옵니다
func (s *S3Service) loadConfig() (*Config, error) {
	resp, err := s.httpClient.Get(s.configURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 상태 코드 오류: %d", resp.StatusCode)
	}

	var config Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	// 설정 유효성 검증
	if err := s.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("설정 유효성 검증 실패: %v", err)
	}

	return &config, nil
}

// validateConfig 설정 유효성을 검증합니다
func (s *S3Service) validateConfig(config *Config) error {
	if config.Running == "" {
		return fmt.Errorf("running 필드가 비어있습니다")
	}
	
	if config.MainVer == "" {
		return fmt.Errorf("mainVer 필드가 비어있습니다")
	}
	
	if config.MinVer == "" {
		return fmt.Errorf("minVer 필드가 비어있습니다")
	}
	
	// running 값 검증
	switch config.Running {
	case "all", "target", "off":
		// 유효한 값
	default:
		return fmt.Errorf("잘못된 running 값: %s", config.Running)
	}
	
	return nil
}

// GetConfigURL 설정 URL을 반환합니다
func (s *S3Service) GetConfigURL() string {
	return s.configURL
}
