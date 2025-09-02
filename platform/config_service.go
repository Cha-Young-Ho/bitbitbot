package platform

import (
	"fmt"
	"log"
)

// ConfigService 설정 관리 로직을 담당하는 서비스
type ConfigService struct {
	s3Service *S3Service
	config    *Config
}

// NewConfigService 새로운 설정 서비스 생성
func NewConfigService(s3Service *S3Service) *ConfigService {
	return &ConfigService{
		s3Service: s3Service,
		config:    nil,
	}
}

// LoadConfig S3에서 설정을 로드합니다
func (cs *ConfigService) LoadConfig() error {
	config, err := cs.s3Service.LoadConfigWithRetry()
	if err != nil {
		return fmt.Errorf("S3 설정 로드 실패: %v", err)
	}

	cs.config = config
	log.Printf("설정 로드 완료: running=%s, mainVer=%s, minVer=%s", 
		config.Running, config.MainVer, config.MinVer)
	return nil
}

// GetConfig 현재 로드된 설정을 반환합니다
func (cs *ConfigService) GetConfig() *Config {
	return cs.config
}

// CheckRunningStatus running 상태를 확인합니다
func (cs *ConfigService) CheckRunningStatus() error {
	if cs.config == nil {
		return fmt.Errorf("설정이 로드되지 않았습니다")
	}

	switch cs.config.Running {
	case "all":
		log.Printf("프로그램 실행 허용: running=%s", cs.config.Running)
		return nil
	case "target":
		log.Printf("타겟 사용자만 실행 허용: running=%s", cs.config.Running)
		// TODO: 화이트리스트 체크 로직 추가
		return nil
	case "off":
		return fmt.Errorf("프로그램 실행이 차단되었습니다: running=%s", cs.config.Running)
	default:
		return fmt.Errorf("알 수 없는 running 상태: %s", cs.config.Running)
	}
}

// IsRunningAllowed running 상태가 허용되는지 확인합니다
func (cs *ConfigService) IsRunningAllowed() bool {
	if cs.config == nil {
		return false
	}
	return cs.config.Running != "off"
}

// GetRunningStatus running 상태를 반환합니다
func (cs *ConfigService) GetRunningStatus() string {
	if cs.config == nil {
		return ""
	}
	return cs.config.Running
}

// ValidateConfig 설정의 유효성을 검증합니다
func (cs *ConfigService) ValidateConfig() error {
	if cs.config == nil {
		return fmt.Errorf("설정이 nil입니다")
	}

	// 필수 필드 검증
	if cs.config.Running == "" {
		return fmt.Errorf("running 필드가 비어있습니다")
	}
	
	if cs.config.MainVer == "" {
		return fmt.Errorf("mainVer 필드가 비어있습니다")
	}
	
	if cs.config.MinVer == "" {
		return fmt.Errorf("minVer 필드가 비어있습니다")
	}

	// running 값 검증
	switch cs.config.Running {
	case "all", "target", "off":
		// 유효한 값
	default:
		return fmt.Errorf("잘못된 running 값: %s", cs.config.Running)
	}

	return nil
}

// RefreshConfig 설정을 새로고침합니다
func (cs *ConfigService) RefreshConfig() error {
	return cs.LoadConfig()
}
