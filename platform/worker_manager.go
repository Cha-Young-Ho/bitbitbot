package platform

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// WorkerManager 워커 생명주기를 관리하는 매니저
type WorkerManager struct {
	mu            sync.RWMutex
	workers       map[string]WorkerInterface
	workerConfigs map[string]*WorkerConfig
	storage       *MemoryStorage
	factory       *WorkerFactory
	ctx           context.Context
	cancelFunc    context.CancelFunc
}

// NewWorkerManager 새로운 워커 매니저 생성
func NewWorkerManager() *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerManager{
		workers:       make(map[string]WorkerInterface),
		workerConfigs: make(map[string]*WorkerConfig),
		storage:       NewMemoryStorage(),
		factory:       NewWorkerFactory(),
		ctx:           ctx,
		cancelFunc:    cancel,
	}
}

// StartWorker 워커 시작
func (wm *WorkerManager) StartWorker(workerID string, config *WorkerConfig) map[string]interface{} {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// 이미 실행 중인 워커가 있는지 확인
	if existingWorker, exists := wm.workers[workerID]; exists && existingWorker.IsRunning() {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 %s가 이미 실행 중입니다.", workerID),
		}
	}

	// 기존 워커가 있다면 정리
	if existingWorker, exists := wm.workers[workerID]; exists {
		wm.stopWorkerInternal(existingWorker, workerID)
	}

	// 워커 생성
	worker, err := wm.factory.CreateWorker(config, wm.storage)
	if err != nil {
		wm.storage.AddLog("error", fmt.Sprintf("워커 생성 실패: %v", err), config.Exchange, config.Symbol)
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 생성 실패: %v", err),
		}
	}

	// 워커 설정 저장
	wm.workerConfigs[workerID] = config

	// 워커 시작 (별도 고루틴에서 실행)
	go func() {
		worker.Start(wm.ctx)
	}()

	// 워커 맵에 저장
	wm.workers[workerID] = worker

	// 상태 업데이트
	wm.storage.SetWorkerStatus(workerID, "running")
	wm.storage.AddLog("info", fmt.Sprintf("워커 %s가 시작되었습니다.", workerID), config.Exchange, config.Symbol)

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("워커 %s가 시작되었습니다.", workerID),
		"workerID": workerID,
	}
}

// StopWorker 워커 중지
func (wm *WorkerManager) StopWorker(workerID string) map[string]interface{} {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	worker, exists := wm.workers[workerID]
	if !exists {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 %s를 찾을 수 없습니다.", workerID),
		}
	}

	if !worker.IsRunning() {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 %s가 이미 중지되었습니다.", workerID),
		}
	}

	// 워커 중지
	wm.stopWorkerInternal(worker, workerID)

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("워커 %s가 중지되었습니다.", workerID),
	}
}

// stopWorkerInternal 워커 내부 정지 로직
func (wm *WorkerManager) stopWorkerInternal(worker WorkerInterface, workerID string) {
	// 워커 중지
	worker.Stop()
	
	// 워커 맵에서 제거
	delete(wm.workers, workerID)
	
	// 설정 제거
	delete(wm.workerConfigs, workerID)
	
	// 상태 업데이트
	wm.storage.SetWorkerStatus(workerID, "stopped")
	
	// 로그 추가
	config := wm.workerConfigs[workerID]
	if config != nil {
		wm.storage.AddLog("info", fmt.Sprintf("워커 %s가 중지되었습니다.", workerID), config.Exchange, config.Symbol)
	}
}

// StopAllWorkers 모든 워커 중지
func (wm *WorkerManager) StopAllWorkers() map[string]interface{} {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	stoppedCount := 0
	for workerID, worker := range wm.workers {
		if worker.IsRunning() {
			wm.stopWorkerInternal(worker, workerID)
			stoppedCount++
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("%d개의 워커가 중지되었습니다.", stoppedCount),
		"stoppedCount": stoppedCount,
	}
}

// GetWorkerStatus 워커 상태 조회
func (wm *WorkerManager) GetWorkerStatus(workerID string) map[string]interface{} {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	worker, exists := wm.workers[workerID]
	if !exists {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 %s를 찾을 수 없습니다.", workerID),
		}
	}

	status := "stopped"
	if worker.IsRunning() {
		status = "running"
	}

	return map[string]interface{}{
		"success": true,
		"status":  status,
		"workerID": workerID,
	}
}

// GetAllWorkerStatuses 모든 워커 상태 조회
func (wm *WorkerManager) GetAllWorkerStatuses() map[string]interface{} {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	statuses := make(map[string]string)
	for workerID, worker := range wm.workers {
		if worker.IsRunning() {
			statuses[workerID] = "running"
		} else {
			statuses[workerID] = "stopped"
		}
	}

	return map[string]interface{}{
		"success": true,
		"statuses": statuses,
		"count":    len(statuses),
	}
}

// GetWorkerConfig 워커 설정 조회
func (wm *WorkerManager) GetWorkerConfig(workerID string) map[string]interface{} {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	config, exists := wm.workerConfigs[workerID]
	if !exists {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 %s의 설정을 찾을 수 없습니다.", workerID),
		}
	}

	return map[string]interface{}{
		"success": true,
		"config":  config,
		"workerID": workerID,
	}
}

// SetWorkerConfig 워커 설정 저장
func (wm *WorkerManager) SetWorkerConfig(workerID string, config *WorkerConfig) map[string]interface{} {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// 입력값 검증
	if config.Exchange == "" {
		return map[string]interface{}{
			"success": false,
			"message": "거래소를 선택해주세요.",
		}
	}

	if config.AccessKey == "" || config.SecretKey == "" {
		return map[string]interface{}{
			"success": false,
			"message": "Access Key와 Secret Key를 입력해주세요.",
		}
	}

	if config.RequestInterval <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "요청 간격은 0보다 큰 숫자여야 합니다.",
		}
	}

	if config.Symbol == "" {
		return map[string]interface{}{
			"success": false,
			"message": "심볼을 입력해주세요.",
		}
	}

	if config.SellAmount <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "매도 수량은 0보다 큰 숫자여야 합니다.",
		}
	}

	if config.SellPrice <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "매도 가격은 0보다 큰 숫자여야 합니다.",
		}
	}

	// 설정 저장
	wm.workerConfigs[workerID] = config
	wm.storage.SetWorkerConfig(workerID, config)

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("워커 %s 설정이 저장되었습니다.", workerID),
		"workerID": workerID,
	}
}

// GetLogs 로그 조회
func (wm *WorkerManager) GetLogs(limit int) map[string]interface{} {
	if limit <= 0 {
		limit = 100 // 기본값
	}

	logs := wm.storage.GetLogs(limit)
	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"count":   len(logs),
	}
}

// ClearLogs 로그 초기화
func (wm *WorkerManager) ClearLogs() map[string]interface{} {
	wm.storage.ClearLogs()
	return map[string]interface{}{
		"success": true,
		"message": "로그가 초기화되었습니다.",
	}
}

// Cleanup 정리
func (wm *WorkerManager) Cleanup() {
	log.Println("WorkerManager: 정리 작업 시작")
	
	// 모든 워커 중지
	wm.StopAllWorkers()
	
	// 컨텍스트 취소
	wm.cancelFunc()
	
	log.Println("WorkerManager: 정리 작업 완료")
}

// IsWorkerRunning 워커 실행 상태 확인
func (wm *WorkerManager) IsWorkerRunning(workerID string) bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	worker, exists := wm.workers[workerID]
	if !exists {
		return false
	}

	return worker.IsRunning()
}

// GetRunningWorkerCount 실행 중인 워커 수 반환
func (wm *WorkerManager) GetRunningWorkerCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	count := 0
	for _, worker := range wm.workers {
		if worker.IsRunning() {
			count++
		}
	}

	return count
}
