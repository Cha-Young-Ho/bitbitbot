package platform

import (
	"fmt"
	"log"
	"time"
)

// VersionCheckerService 모든 버전 체크 로직을 통합하는 메인 서비스
type VersionCheckerService struct {
	configService   *ConfigService
	versionService  *VersionService
	checkInterval   time.Duration
	isInitialized   bool
}

// NewVersionCheckerService 새로운 버전 체커 서비스 생성
func NewVersionCheckerService(configService *ConfigService, versionService *VersionService) *VersionCheckerService {
	return &VersionCheckerService{
		configService:  configService,
		versionService: versionService,
		checkInterval:  30 * time.Minute,
		isInitialized:  false,
	}
}

// SetCheckInterval 체크 간격을 설정합니다
func (vcs *VersionCheckerService) SetCheckInterval(interval time.Duration) {
	vcs.checkInterval = interval
}

// Initialize 초기 버전 체크를 수행합니다
func (vcs *VersionCheckerService) Initialize() error {
	log.Println("버전 체커 서비스 초기화 시작")
	
	if err := vcs.performFullCheck(); err != nil {
		return fmt.Errorf("초기 버전 체크 실패: %v", err)
	}
	
	vcs.isInitialized = true
	log.Println("버전 체커 서비스 초기화 완료")
	return nil
}

// PerformPeriodicCheck 주기적 버전 체크를 수행합니다
func (vcs *VersionCheckerService) PerformPeriodicCheck() error {
	if !vcs.isInitialized {
		return fmt.Errorf("서비스가 초기화되지 않았습니다")
	}
	
	log.Printf("주기적 버전 체크 시작 (간격: %v)", vcs.checkInterval)
	return vcs.performFullCheck()
}

// performFullCheck 전체 버전 체크를 수행합니다
func (vcs *VersionCheckerService) performFullCheck() error {
	// 1. S3에서 설정 로드
	if err := vcs.configService.LoadConfig(); err != nil {
		return fmt.Errorf("설정 로드 실패: %v", err)
	}

	// 2. 설정 유효성 검증
	if err := vcs.configService.ValidateConfig(); err != nil {
		return fmt.Errorf("설정 유효성 검증 실패: %v", err)
	}

	// 3. running 상태 체크
	if err := vcs.configService.CheckRunningStatus(); err != nil {
		return fmt.Errorf("running 상태 체크 실패: %v", err)
	}

	// 4. 버전 비교
	config := vcs.configService.GetConfig()
	comparison, err := vcs.versionService.CompareVersions(config)
	if err != nil {
		return fmt.Errorf("버전 비교 실패: %v", err)
	}

	// 5. 결과 로깅
	vcs.logCheckResult(comparison)

	return nil
}

// logCheckResult 체크 결과를 로깅합니다
func (vcs *VersionCheckerService) logCheckResult(comparison *VersionComparison) {
	log.Printf("=== 버전 체크 결과 ===")
	log.Printf("현재 버전: %s", comparison.CurrentVersion)
	log.Printf("메인 버전: %s", comparison.MainVersion)
	log.Printf("최소 버전: %s", comparison.MinVersion)
	log.Printf("업데이트 필요: %v", comparison.IsUpdateNeeded)
	log.Printf("강제 업데이트: %v", comparison.IsForceUpdate)
	log.Printf("업데이트 타입: %s", comparison.UpdateType)
	log.Printf("=====================")
}

// GetCheckResult 현재 체크 결과를 반환합니다
func (vcs *VersionCheckerService) GetCheckResult() (*VersionCheckResult, error) {
	if !vcs.isInitialized {
		return nil, fmt.Errorf("서비스가 초기화되지 않았습니다")
	}

	config := vcs.configService.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("설정이 로드되지 않았습니다")
	}

	comparison, err := vcs.versionService.CompareVersions(config)
	if err != nil {
		return nil, fmt.Errorf("버전 비교 실패: %v", err)
	}

	result := &VersionCheckResult{
		Success:         true,
		CurrentVersion:  comparison.CurrentVersion,
		LatestVersion:   comparison.MainVersion,
		IsLatest:        !comparison.IsUpdateNeeded,
		IsUpdateNeeded:  comparison.IsUpdateNeeded,
		UpdateType:      comparison.UpdateType,
		IsForceUpdate:   comparison.IsForceUpdate,
		Message:         "버전 체크가 완료되었습니다.",
		Config:          vcs.configToMap(config),
		Comparison:      comparison,
	}

	return result, nil
}

// configToMap Config 구조체를 map으로 변환합니다
func (vcs *VersionCheckerService) configToMap(config *Config) map[string]interface{} {
	return map[string]interface{}{
		"running":     config.Running,
		"whiteList":   config.WhiteList,
		"mainVer":     config.MainVer,
		"minVer":      config.MinVer,
		"forceUpdate": config.ForceUpdate,
	}
}

// IsInitialized 서비스가 초기화되었는지 확인합니다
func (vcs *VersionCheckerService) IsInitialized() bool {
	return vcs.isInitialized
}

// GetCheckInterval 현재 체크 간격을 반환합니다
func (vcs *VersionCheckerService) GetCheckInterval() time.Duration {
	return vcs.checkInterval
}

// VersionCheckResult 버전 체크 결과
type VersionCheckResult struct {
	Success        bool                 `json:"success"`
	CurrentVersion string               `json:"currentVersion"`
	LatestVersion  string               `json:"latestVersion"`
	IsLatest       bool                 `json:"isLatest"`
	IsUpdateNeeded bool                 `json:"isUpdateNeeded"`
	UpdateType     string               `json:"updateType"`
	IsForceUpdate  bool                 `json:"isForceUpdate"`
	Message        string               `json:"message"`
	Config         map[string]interface{} `json:"config"`
	Comparison     *VersionComparison   `json:"comparison"`
}
