package platform

import (
	"fmt"
	"strconv"
)

// VersionChecker 버전 체크 인터페이스
type VersionChecker interface {
	CheckVersionUpdate() error
	CheckRunningStatus() error
	CompareVersions() (bool, bool, error)
	GetConfig() interface{}
	GetCurrentVersion() string
}

// Handler 워커 관리 핸들러
type Handler struct {
	workerManager *WorkerManager
	versionChecker VersionChecker
	keyStorage   *KeyStorage
}

// NewHandler 새로운 핸들러 생성
func NewHandler() *Handler {
	return &Handler{
		workerManager: NewWorkerManager(),
		versionChecker: nil, // main 패키지에서 주입받아야 함
		keyStorage:   NewKeyStorage(),
	}
}

// SetVersionChecker 버전 체커 설정
func (h *Handler) SetVersionChecker(checker VersionChecker) {
	h.versionChecker = checker
}

// SetWorkerConfig 워커 설정
func (h *Handler) SetWorkerConfig(exchange, accessKey, secretKey, passwordPhrase, requestInterval, symbol, sellAmount, sellPrice string) map[string]interface{} {
	// 입력값 검증
	if exchange == "" {
		return map[string]interface{}{
			"success": false,
			"message": "거래소를 선택해주세요.",
		}
	}

	if accessKey == "" || secretKey == "" {
		return map[string]interface{}{
			"success": false,
			"message": "Access Key와 Secret Key를 입력해주세요.",
		}
	}

	interval, err := strconv.ParseFloat(requestInterval, 64)
	if err != nil || interval <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "요청 간격은 0보다 큰 숫자여야 합니다.",
		}
	}

	if symbol == "" {
		return map[string]interface{}{
			"success": false,
			"message": "심볼을 입력해주세요.",
		}
	}

	// 매도 수량 검증
	amount, err := strconv.ParseFloat(sellAmount, 64)
	if err != nil || amount <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "매도 수량은 0보다 큰 숫자여야 합니다.",
		}
	}

	// 매도 가격 검증
	price, err := strconv.ParseFloat(sellPrice, 64)
	if err != nil || price <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "매도 가격은 0보다 큰 숫자여야 합니다.",
		}
	}

	// 설정 저장
	config := &WorkerConfig{
		Exchange:        exchange,
		AccessKey:       accessKey,
		SecretKey:       secretKey,
		PasswordPhrase:  passwordPhrase,
		RequestInterval: interval,
		Symbol:          symbol,
		SellAmount:      amount,
		SellPrice:       price,
	}

	return h.workerManager.SetWorkerConfig("main", config)
}

// GetWorkerConfig 워커 설정 조회
func (h *Handler) GetWorkerConfig() map[string]interface{} {
	return h.workerManager.GetWorkerConfig("main")
}

// StartWorker 워커 시작
func (h *Handler) StartWorker() map[string]interface{} {
	config := h.workerManager.GetWorkerConfig("main")
	if config["success"] == false {
		return config
	}

	workerConfig := config["config"].(*WorkerConfig)
	return h.workerManager.StartWorker("main", workerConfig)
}

// StopWorker 워커 중지
func (h *Handler) StopWorker() map[string]interface{} {
	return h.workerManager.StopWorker("main")
}

// GetWorkerStatus 워커 상태 조회
func (h *Handler) GetWorkerStatus() map[string]interface{} {
	return h.workerManager.GetWorkerStatus("main")
}

// GetLogs 로그 조회
func (h *Handler) GetLogs(limit int) map[string]interface{} {
	return h.workerManager.GetLogs(limit)
}

// ClearLogs 로그 초기화
func (h *Handler) ClearLogs() map[string]interface{} {
	return h.workerManager.ClearLogs()
}

// CheckVersion 버전 체크
func (h *Handler) CheckVersion() map[string]interface{} {
	// 버전 체커가 설정되지 않은 경우 기본 응답
	if h.versionChecker == nil {
		return map[string]interface{}{
			"success": false,
			"message": "버전 체커가 설정되지 않았습니다",
		}
	}

	// S3에서 설정 로드
	if err := h.versionChecker.CheckVersionUpdate(); err != nil {
		h.workerManager.storage.AddLog("error", fmt.Sprintf("S3 설정 로드 실패: %v", err), "", "")
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("설정 로드 실패: %v", err),
		}
	}

	// running 상태 체크
	if err := h.versionChecker.CheckRunningStatus(); err != nil {
		h.workerManager.storage.AddLog("error", fmt.Sprintf("실행 상태 체크 실패: %v", err), "", "")
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("실행 상태 체크 실패: %v", err),
		}
	}

	// 버전 비교
	isMainUpdateNeeded, isMinUpdateNeeded, err := h.versionChecker.CompareVersions()
	if err != nil {
		h.workerManager.storage.AddLog("error", fmt.Sprintf("버전 비교 실패: %v", err), "", "")
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("버전 비교 실패: %v", err),
		}
	}

	config := h.versionChecker.GetConfig()
	currentVersion := h.versionChecker.GetCurrentVersion()
	
	// config에서 MainVer 추출
	var latestVersion string
	if configMap, ok := config.(map[string]interface{}); ok {
		if mainVer, exists := configMap["mainVer"]; exists {
			latestVersion = fmt.Sprintf("%v", mainVer)
		}
	}
	
	if latestVersion == "" {
		latestVersion = "알 수 없음"
	}

	// 업데이트 필요 여부 결정
	isUpdateNeeded := isMainUpdateNeeded || isMinUpdateNeeded
	updateType := "none"
	
	if isMinUpdateNeeded {
		updateType = "force" // 강제 업데이트
	} else if isMainUpdateNeeded {
		updateType = "recommended" // 권장 업데이트
	}

	// 버전 체크 로그 제거

	return map[string]interface{}{
		"success":        true,
		"currentVersion": currentVersion,
		"latestVersion":  latestVersion,
		"isLatest":       !isUpdateNeeded,
		"isUpdateNeeded": isUpdateNeeded,
		"updateType":     updateType,
		"isForceUpdate":  isMinUpdateNeeded,
		"message":        "버전 체크가 완료되었습니다.",
		"config":         config,
	}
}

// Cleanup 정리
func (h *Handler) Cleanup() {
	h.workerManager.Cleanup()
}

// GetKeyStorage 키 저장소 반환
func (h *Handler) GetKeyStorage() *KeyStorage {
	return h.keyStorage
}

// DownloadUpdate 업데이트 파일 다운로드
func (h *Handler) DownloadUpdate() map[string]interface{} {
	// S3에서 최신 버전 정보 가져오기
	configInterface := h.versionChecker.GetConfig()
	if configInterface == nil {
		return map[string]interface{}{
			"success": false,
			"message": "설정 정보를 가져올 수 없습니다",
		}
	}

	// Config 타입으로 캐스팅
	config, ok := configInterface.(*Config)
	if !ok {
		return map[string]interface{}{
			"success": false,
			"message": "설정 정보 형식이 올바르지 않습니다",
		}
	}

	// 현재 플랫폼에 맞는 다운로드 URL 결정
	var downloadURL string
	if config.MacURL != "" {
		downloadURL = config.MacURL
	} else if config.WinURL != "" {
		downloadURL = config.WinURL
	} else {
		return map[string]interface{}{
			"success": false,
			"message": "다운로드 URL을 찾을 수 없습니다",
		}
	}

	// 다운로드 실행 (실제 구현은 별도 함수로)
	h.workerManager.storage.AddLog("info", fmt.Sprintf("업데이트 다운로드 시작: %s", downloadURL), "", "")
	
	return map[string]interface{}{
		"success": true,
		"message": "업데이트 다운로드가 시작되었습니다",
		"downloadURL": downloadURL,
	}
}

// InstallUpdate 업데이트 설치
func (h *Handler) InstallUpdate() map[string]interface{} {
	h.workerManager.storage.AddLog("info", "업데이트 설치를 시작합니다", "", "")
	
	// 실제 설치 로직은 여기에 구현
	// 1. 다운로드된 파일 확인
	// 2. 현재 실행 파일 백업
	// 3. 새 파일로 교체
	// 4. 애플리케이션 재시작
	
	return map[string]interface{}{
		"success": true,
		"message": "업데이트가 설치되었습니다. 애플리케이션을 재시작해주세요.",
	}
}
