package platform

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// Handler 워커 관리 핸들러
type Handler struct {
	storage    *MemoryStorage
	worker     WorkerInterface
	ctx        context.Context
	cancelFunc context.CancelFunc
	factory    *WorkerFactory
}

// NewHandler 새로운 핸들러 생성
func NewHandler() *Handler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Handler{
		storage:    NewMemoryStorage(),
		worker:     nil,
		ctx:        ctx,
		cancelFunc: cancel,
		factory:    NewWorkerFactory(),
	}
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

	h.storage.SetWorkerConfig(config)

	// 로그 추가
	h.storage.AddLog("info", "워커 설정이 저장되었습니다.", exchange, symbol)

	return map[string]interface{}{
		"success": true,
		"message": "워커 설정이 저장되었습니다.",
		"config":  config,
	}
}

// GetWorkerConfig 워커 설정 조회
func (h *Handler) GetWorkerConfig() map[string]interface{} {
	config := h.storage.GetWorkerConfig()
	return map[string]interface{}{
		"success": true,
		"config":  config,
	}
}

// StartWorker 워커 시작
func (h *Handler) StartWorker() map[string]interface{} {
	config := h.storage.GetWorkerConfig()
	if config.Exchange == "" {
		return map[string]interface{}{
			"success": false,
			"message": "먼저 워커 설정을 해주세요.",
		}
	}

	// 이미 실행 중이면 중지
	if h.worker != nil && h.worker.IsRunning() {
		h.StopWorker()
	}

	// 팩토리를 통해 거래소별 워커 생성
	worker, err := h.factory.CreateWorker(config, h.storage)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 생성 실패: %v", err),
		}
	}
	h.worker = worker

	// 워커 시작
	go worker.Start(h.ctx)

	// 상태 저장
	status := &WorkerStatus{
		IsRunning:    true,
		StartedAt:    time.Now(),
		LastRequest:  time.Time{},
		RequestCount: 0,
		ErrorCount:   0,
	}
	h.storage.SetWorkerStatus("main", status)

	// 로그 추가
	h.storage.AddLog("success", "워커가 시작되었습니다.", config.Exchange, config.Symbol)

	return map[string]interface{}{
		"success": true,
		"message": "워커가 시작되었습니다.",
	}
}

// StopWorker 워커 중지
func (h *Handler) StopWorker() map[string]interface{} {
	if h.worker == nil {
		return map[string]interface{}{
			"success": false,
			"message": "실행 중인 워커가 없습니다.",
		}
	}

	// 워커 중지
	h.worker.Stop()

	// 상태 업데이트
	status := h.storage.GetWorkerStatus("main")
	if status != nil {
		status.IsRunning = false
		h.storage.SetWorkerStatus("main", status)
	}

	// 로그 추가
	config := h.storage.GetWorkerConfig()
	h.storage.AddLog("info", "워커가 중지되었습니다.", config.Exchange, config.Symbol)

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
	// 간단한 버전 체크 (실제로는 S3에서 버전 정보를 가져와야 함)
	currentVersion := "1.0.0"
	latestVersion := "1.0.0"

	isLatest := currentVersion == latestVersion

	h.storage.AddLog("info", fmt.Sprintf("버전 체크: 현재 %s, 최신 %s", currentVersion, latestVersion), "", "")

	return map[string]interface{}{
		"success":        true,
		"currentVersion": currentVersion,
		"latestVersion":  latestVersion,
		"isLatest":       isLatest,
		"message":        "버전 체크가 완료되었습니다.",
	}
}

// Cleanup 정리
func (h *Handler) Cleanup() {
	if h.worker != nil {
		h.worker.Stop()
	}
	h.cancelFunc()
}
