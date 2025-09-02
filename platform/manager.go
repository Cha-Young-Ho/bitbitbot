package platform

import (
	"context"
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
	storage       *MemoryStorage
	worker        WorkerInterface
	ctx           context.Context
	cancelFunc    context.CancelFunc
	factory       *WorkerFactory
	versionChecker VersionChecker
}

// NewHandler 새로운 핸들러 생성
func NewHandler() *Handler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Handler{
		storage:       NewMemoryStorage(),
		worker:        nil,
		ctx:           ctx,
		cancelFunc:    cancel,
		factory:       NewWorkerFactory(),
		versionChecker: nil, // main 패키지에서 주입받아야 함
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

	h.storage.SetWorkerConfig("main", config)

	return map[string]interface{}{
		"success": true,
		"message": "워커 설정이 저장되었습니다.",
	}
}

// GetWorkerConfig 워커 설정 조회
func (h *Handler) GetWorkerConfig() map[string]interface{} {
	config := h.storage.GetWorkerConfig("main")
	if config == nil {
		return map[string]interface{}{
			"success": false,
			"message": "저장된 워커 설정이 없습니다.",
		}
	}

	return map[string]interface{}{
		"success": true,
		"config":  config,
	}
}

// StartWorker 워커 시작
func (h *Handler) StartWorker() map[string]interface{} {
	config := h.storage.GetWorkerConfig("main")
	if config == nil {
		return map[string]interface{}{
			"success": false,
			"message": "워커 설정이 없습니다. 먼저 설정해주세요.",
		}
	}

	// 이미 실행 중인 경우
	if h.worker != nil && h.worker.IsRunning() {
		return map[string]interface{}{
			"success": false,
			"message": "워커가 이미 실행 중입니다.",
		}
	}

	// 워커 생성
	worker, err := h.factory.CreateWorker(config, h.storage)
	if err != nil {
		h.storage.AddLog("error", fmt.Sprintf("워커 생성 실패: %v", err), config.Exchange, config.Symbol)
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 생성 실패: %v", err),
		}
	}

	// 워커 시작
	worker.Start(h.ctx)
	h.worker = worker

	// 상태 업데이트
	h.storage.SetWorkerStatus("main", "running")

	return map[string]interface{}{
		"success": true,
		"message": "워커가 시작되었습니다.",
	}
}

// StopWorker 워커 중지
func (h *Handler) StopWorker() map[string]interface{} {
	if h.worker == nil || !h.worker.IsRunning() {
		return map[string]interface{}{
			"success": false,
			"message": "실행 중인 워커가 없습니다.",
		}
	}

	h.worker.Stop()
	h.worker = nil

	// 상태 업데이트
	h.storage.SetWorkerStatus("main", "stopped")

	return map[string]interface{}{
		"success": true,
		"message": "워커가 중지되었습니다.",
	}
}

// GetWorkerStatus 워커 상태 조회
func (h *Handler) GetWorkerStatus() map[string]interface{} {
	status := h.storage.GetWorkerStatus("main")
	return map[string]interface{}{
		"success": true,
		"status":  status,
	}
}

// GetLogs 로그 조회
func (h *Handler) GetLogs(limit int) map[string]interface{} {
	if limit <= 0 {
		limit = 100 // 기본값
	}

	logs := h.storage.GetLogs(limit)
	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"count":   len(logs),
	}
}

// ClearLogs 로그 초기화
func (h *Handler) ClearLogs() map[string]interface{} {
	h.storage.ClearLogs()
	return map[string]interface{}{
		"success": true,
		"message": "로그가 초기화되었습니다.",
	}
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
		h.storage.AddLog("error", fmt.Sprintf("S3 설정 로드 실패: %v", err), "", "")
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("설정 로드 실패: %v", err),
		}
	}

	// running 상태 체크
	if err := h.versionChecker.CheckRunningStatus(); err != nil {
		h.storage.AddLog("error", fmt.Sprintf("실행 상태 체크 실패: %v", err), "", "")
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("실행 상태 체크 실패: %v", err),
		}
	}

	// 버전 비교
	isMainUpdateNeeded, isMinUpdateNeeded, err := h.versionChecker.CompareVersions()
	if err != nil {
		h.storage.AddLog("error", fmt.Sprintf("버전 비교 실패: %v", err), "", "")
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

	h.storage.AddLog("info", fmt.Sprintf("버전 체크: 현재 %s, 최신 %s, 업데이트 타입: %s", 
		currentVersion, latestVersion, updateType), "", "")

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
	if h.worker != nil {
		h.worker.Stop()
	}
	h.cancelFunc()
}
