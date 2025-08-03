package platform

import (
	"bitbit-app/local_file"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Worker 인터페이스 - 모든 플랫폼 워커가 구현해야 하는 인터페이스
type Worker interface {
	// Start 워커를 시작합니다
	Start(ctx context.Context) error

	// Stop 워커를 중지합니다
	Stop() error

	// GetStatus 워커의 현재 상태를 반환합니다
	GetStatus() WorkerStatus

	// GetOrderInfo 주문 정보를 반환합니다
	GetOrderInfo() local_file.SellOrder
}

// WorkerStatus 워커 상태 정보
type WorkerStatus struct {
	IsRunning  bool      `json:"isRunning"`
	LastCheck  time.Time `json:"lastCheck"`
	LastPrice  float64   `json:"lastPrice"`
	CheckCount int       `json:"checkCount"`
	ErrorCount int       `json:"errorCount"`
	LastError  string    `json:"lastError"`
}

// WorkerInfo 워커 정보
type WorkerInfo struct {
	OrderName    string               `json:"orderName"`
	UserID       string               `json:"userId"`
	Order        local_file.SellOrder `json:"order"`
	Status       WorkerStatus         `json:"status"`
	CreatedAt    time.Time            `json:"createdAt"`
	LastActivity time.Time            `json:"lastActivity"`
	Logs         []WorkerLog          `json:"logs"`
	MaxLogs      int                  `json:"maxLogs"` // 최대 로그 개수
}

// WorkerManager 워커들을 관리하는 매니저
type WorkerManager struct {
	workers        map[string]Worker      // key: orderName
	workerInfo     map[string]*WorkerInfo // key: orderName
	mu             sync.RWMutex
	logChan        chan WorkerLog
	unifiedLogChan chan UnifiedLog // 통합된 로그 채널
	ctx            context.Context
	cancel         context.CancelFunc
	localHandler   *local_file.Handler        // 로컬 파일 핸들러
	clients        map[string]chan UnifiedLog // 소켓 클라이언트들 (userID -> channel)
	clientsMu      sync.RWMutex
	wsClients      map[string]*Client // 웹소켓 클라이언트들 (userID -> client)
	wsClientsMu    sync.RWMutex
}

// WorkerLog 워커에서 발생하는 로그
type WorkerLog struct {
	OrderName   string    `json:"orderName"`
	Platform    string    `json:"platform"`
	Symbol      string    `json:"symbol"`
	Message     string    `json:"message"`
	LogType     string    `json:"logType"` // "info", "error", "price", "order", "status"
	Timestamp   time.Time `json:"timestamp"`
	Price       float64   `json:"price,omitempty"`
	Quantity    float64   `json:"quantity,omitempty"`
	CheckCount  int       `json:"checkCount,omitempty"`
	ErrorCount  int       `json:"errorCount,omitempty"`
	LastPrice   float64   `json:"lastPrice,omitempty"`
	TargetPrice float64   `json:"targetPrice,omitempty"`
	UserID      string    `json:"userId,omitempty"`
	OrderStatus string    `json:"orderStatus,omitempty"` // "running", "stopped", "completed"
}

// UnifiedLog 모든 워커의 통합된 로그 모델
type UnifiedLog struct {
	Platform    string    `json:"platform"`    // 플랫폼명
	Nickname    string    `json:"nickname"`    // 별칭
	OrderName   string    `json:"orderName"`   // 주문명
	Symbol      string    `json:"symbol"`      // 심볼
	Message     string    `json:"message"`     // 메시지
	LogType     string    `json:"logType"`     // 로그 타입
	Timestamp   time.Time `json:"timestamp"`   // 타임스탬프
	UserID      string    `json:"userId"`      // 사용자 ID
	CheckCount  int       `json:"checkCount"`  // 체크 횟수
	ErrorCount  int       `json:"errorCount"`  // 에러 횟수
	LastPrice   float64   `json:"lastPrice"`   // 마지막 가격
	TargetPrice float64   `json:"targetPrice"` // 목표 가격
	OrderStatus string    `json:"orderStatus"` // 주문 상태
}

// NewWorkerManager 새로운 워커 매니저를 생성합니다
func NewWorkerManager() *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerManager{
		workers:        make(map[string]Worker),
		workerInfo:     make(map[string]*WorkerInfo),
		logChan:        make(chan WorkerLog, 1000),  // 버퍼 크기 1000
		unifiedLogChan: make(chan UnifiedLog, 1000), // 통합된 로그 채널
		ctx:            ctx,
		cancel:         cancel,
		localHandler:   local_file.NewHandler(),
		clients:        make(map[string]chan UnifiedLog),
		wsClients:      make(map[string]*Client),
	}
}

// AddWorker 워커를 추가합니다
func (wm *WorkerManager) AddWorker(orderName string, userID string, worker Worker) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.workers[orderName]; exists {
		return fmt.Errorf("워커가 이미 존재합니다: %s", orderName)
	}

	// 워커 정보 생성
	workerInfo := &WorkerInfo{
		OrderName:    orderName,
		UserID:       userID,
		Order:        worker.GetOrderInfo(),
		Status:       worker.GetStatus(),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Logs:         []WorkerLog{},
		MaxLogs:      1000, // 최대 1000개 로그 저장
	}

	wm.workers[orderName] = worker
	wm.workerInfo[orderName] = workerInfo

	// 로그 수집 고루틴 시작
	go wm.collectLogs(orderName)

	return nil
}

// RemoveWorker 워커를 제거합니다
func (wm *WorkerManager) RemoveWorker(orderName string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	worker, exists := wm.workers[orderName]
	if !exists {
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	// 워커 중지
	if err := worker.Stop(); err != nil {
		log.Printf("워커 중지 실패: %v", err)
	}

	delete(wm.workers, orderName)
	return nil
}

// StartWorker 특정 워커를 시작합니다
func (wm *WorkerManager) StartWorker(orderName string) error {
	wm.mu.RLock()
	worker, exists := wm.workers[orderName]
	wm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	return worker.Start(wm.ctx)
}

// StopWorker 특정 워커를 중지합니다
func (wm *WorkerManager) StopWorker(orderName string) error {
	wm.mu.RLock()
	worker, exists := wm.workers[orderName]
	wm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	return worker.Stop()
}

// GetWorkerStatus 모든 워커의 상태를 반환합니다
func (wm *WorkerManager) GetWorkerStatus() map[string]WorkerStatus {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	status := make(map[string]WorkerStatus)
	for orderName, worker := range wm.workers {
		status[orderName] = worker.GetStatus()
	}

	return status
}

// GetLogChannel 로그 채널을 반환합니다
func (wm *WorkerManager) GetLogChannel() <-chan WorkerLog {
	return wm.logChan
}

// GetUnifiedLogChannel 통합된 로그 채널을 반환합니다
func (wm *WorkerManager) GetUnifiedLogChannel() <-chan UnifiedLog {
	return wm.unifiedLogChan
}

// SubscribeClient 클라이언트를 구독합니다
func (wm *WorkerManager) SubscribeClient(userID string) chan UnifiedLog {
	wm.clientsMu.Lock()
	defer wm.clientsMu.Unlock()

	clientChan := make(chan UnifiedLog, 100) // 버퍼 크기 100
	wm.clients[userID] = clientChan

	log.Printf("클라이언트 구독: %s", userID)
	return clientChan
}

// UnsubscribeClient 클라이언트 구독을 해제합니다
func (wm *WorkerManager) UnsubscribeClient(userID string) {
	wm.clientsMu.Lock()
	defer wm.clientsMu.Unlock()

	if clientChan, exists := wm.clients[userID]; exists {
		close(clientChan)
		delete(wm.clients, userID)
		log.Printf("클라이언트 구독 해제: %s", userID)
	}
}

// RegisterWebSocketClient 웹소켓 클라이언트를 등록합니다
func (wm *WorkerManager) RegisterWebSocketClient(userID string, client *Client) {
	wm.wsClientsMu.Lock()
	defer wm.wsClientsMu.Unlock()

	wm.wsClients[userID] = client
	log.Printf("웹소켓 클라이언트 등록: %s", userID)
}

// UnregisterWebSocketClient 웹소켓 클라이언트를 등록 해제합니다
func (wm *WorkerManager) UnregisterWebSocketClient(userID string, client *Client) {
	wm.wsClientsMu.Lock()
	defer wm.wsClientsMu.Unlock()

	if existingClient, exists := wm.wsClients[userID]; exists && existingClient == client {
		delete(wm.wsClients, userID)
		log.Printf("웹소켓 클라이언트 등록 해제: %s", userID)
	}
}

// SendLog 로그를 채널에 전송합니다
func (wm *WorkerManager) SendLog(workerLog WorkerLog) {
	log.Printf("SendLog 호출: %s - %s", workerLog.OrderName, workerLog.Message)

	select {
	case wm.logChan <- workerLog:
		log.Printf("로그 채널 전송 성공: %s", workerLog.Message)
	default:
		// 채널이 가득 찬 경우 로그를 버립니다
		log.Printf("로그 채널이 가득 찼습니다. 로그를 버립니다: %s", workerLog.Message)
	}

	// 통합된 로그로 변환하여 전송
	wm.sendUnifiedLog(workerLog)

	// 로그를 로컬 파일에 저장
	wm.saveLogToLocalFile(workerLog)
}

// sendUnifiedLog 통합된 로그를 전송합니다
func (wm *WorkerManager) sendUnifiedLog(workerLog WorkerLog) {
	// 워커 정보에서 사용자 ID와 별칭 찾기
	wm.mu.RLock()
	workerInfo, exists := wm.workerInfo[workerLog.OrderName]
	wm.mu.RUnlock()

	if !exists {
		log.Printf("워커 정보를 찾을 수 없습니다: %s", workerLog.OrderName)
		return
	}

	// 통합된 로그 생성
	unifiedLog := UnifiedLog{
		Platform:    workerLog.Platform,
		Nickname:    workerInfo.Order.PlatformNickName,
		OrderName:   workerLog.OrderName,
		Symbol:      workerLog.Symbol,
		Message:     workerLog.Message,
		LogType:     workerLog.LogType,
		Timestamp:   workerLog.Timestamp,
		UserID:      workerInfo.UserID,
		CheckCount:  workerLog.CheckCount,
		ErrorCount:  workerLog.ErrorCount,
		LastPrice:   workerLog.LastPrice,
		TargetPrice: workerLog.TargetPrice,
		OrderStatus: workerLog.OrderStatus,
	}

	// 통합된 로그 채널로 전송
	select {
	case wm.unifiedLogChan <- unifiedLog:
		log.Printf("통합 로그 전송 성공: %s - %s", unifiedLog.Platform, unifiedLog.Message)
	default:
		log.Printf("통합 로그 채널이 가득 찼습니다: %s", unifiedLog.Message)
	}

	// 소켓 클라이언트들에게 전송
	wm.broadcastToClients(unifiedLog)
}

// broadcastToClients 모든 클라이언트에게 로그를 브로드캐스트합니다
func (wm *WorkerManager) broadcastToClients(unifiedLog UnifiedLog) {
	// 기존 채널 클라이언트들에게 전송
	wm.clientsMu.RLock()
	for userID, clientChan := range wm.clients {
		if unifiedLog.UserID == userID {
			select {
			case clientChan <- unifiedLog:
				// 성공적으로 전송됨
			default:
				log.Printf("클라이언트 채널이 가득 찼습니다: %s", userID)
			}
		}
	}
	wm.clientsMu.RUnlock()

	// 웹소켓 클라이언트들에게 전송
	wm.wsClientsMu.RLock()
	defer wm.wsClientsMu.RUnlock()

	for userID, client := range wm.wsClients {
		if unifiedLog.UserID == userID {
			// UnifiedLog를 JSON으로 마샬링
			data, err := json.Marshal(unifiedLog)
			if err != nil {
				log.Printf("로그 마샬링 실패: %v", err)
				continue
			}
			client.SendMessage(data)
		}
	}
}

// saveLogToLocalFile 워커 로그를 로컬 파일에 저장합니다
func (wm *WorkerManager) saveLogToLocalFile(workerLog WorkerLog) {
	if wm.localHandler == nil {
		log.Printf("로컬 핸들러가 nil입니다")
		return
	}

	// WorkerLog를 OrderLog로 변환
	orderLog := local_file.OrderLog{
		Timestamp:   workerLog.Timestamp,
		Message:     workerLog.Message,
		LogType:     workerLog.LogType,
		CheckCount:  workerLog.CheckCount,
		ErrorCount:  workerLog.ErrorCount,
		LastPrice:   workerLog.LastPrice,
		TargetPrice: workerLog.TargetPrice,
		OrderStatus: workerLog.OrderStatus,
	}

	// 사용자 ID 찾기 (워커 정보에서)
	wm.mu.RLock()
	workerInfo, exists := wm.workerInfo[workerLog.OrderName]
	wm.mu.RUnlock()

	if !exists {
		log.Printf("워커 정보를 찾을 수 없습니다: %s", workerLog.OrderName)
		return
	}

	log.Printf("로그 저장 시도: userID=%s, orderName=%s, message=%s",
		workerInfo.UserID, workerLog.OrderName, workerLog.Message)

	// 로그 저장
	if err := wm.localHandler.AddOrderLog(workerInfo.UserID, workerLog.OrderName, orderLog); err != nil {
		log.Printf("로그 저장 실패: %v", err)
	} else {
		log.Printf("로그 저장 성공: %s", workerLog.Message)
	}
}

// StartAllWorkers 모든 워커를 시작합니다
func (wm *WorkerManager) StartAllWorkers() error {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	for orderName, worker := range wm.workers {
		if err := worker.Start(wm.ctx); err != nil {
			log.Printf("워커 시작 실패 [%s]: %v", orderName, err)
			return err
		}
	}

	return nil
}

// StopAllWorkers 모든 워커를 중지합니다
func (wm *WorkerManager) StopAllWorkers() {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	for orderName, worker := range wm.workers {
		if err := worker.Stop(); err != nil {
			log.Printf("워커 중지 실패 [%s]: %v", orderName, err)
		}
	}
}

// Shutdown 워커 매니저를 종료합니다
func (wm *WorkerManager) Shutdown() {
	wm.StopAllWorkers()
	wm.cancel()
	close(wm.logChan)
}

// GetWorkerCount 활성 워커 수를 반환합니다
func (wm *WorkerManager) GetWorkerCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return len(wm.workers)
}

// GetWorkerInfo 특정 워커의 정보를 반환합니다
func (wm *WorkerManager) GetWorkerInfo(orderName string) (*WorkerInfo, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workerInfo, exists := wm.workerInfo[orderName]
	if !exists {
		return nil, fmt.Errorf("워커 정보를 찾을 수 없습니다: %s", orderName)
	}

	return workerInfo, nil
}

// GetAllWorkerInfo 모든 워커의 정보를 반환합니다
func (wm *WorkerManager) GetAllWorkerInfo() map[string]*WorkerInfo {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	result := make(map[string]*WorkerInfo)
	for orderName, workerInfo := range wm.workerInfo {
		result[orderName] = workerInfo
	}

	return result
}

// GetWorkerInfoByUserID 특정 사용자의 워커 정보들을 반환합니다
func (wm *WorkerManager) GetWorkerInfoByUserID(userID string) map[string]*WorkerInfo {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	result := make(map[string]*WorkerInfo)
	for orderName, workerInfo := range wm.workerInfo {
		if workerInfo.UserID == userID {
			result[orderName] = workerInfo
		}
	}

	return result
}

// collectLogs 특정 워커의 로그를 수집합니다
func (wm *WorkerManager) collectLogs(orderName string) {
	logChan := wm.GetLogChannel()

	for {
		select {
		case <-wm.ctx.Done():
			return
		case workerLog := <-logChan:
			if workerLog.OrderName == orderName {
				wm.addLogToWorker(orderName, workerLog)
			}
		}
	}
}

// addLogToWorker 워커에 로그를 추가합니다
func (wm *WorkerManager) addLogToWorker(orderName string, workerLog WorkerLog) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workerInfo, exists := wm.workerInfo[orderName]
	if !exists {
		return
	}

	// 로그 추가
	workerInfo.Logs = append(workerInfo.Logs, workerLog)
	workerInfo.LastActivity = time.Now()

	// 최대 로그 개수 제한
	if len(workerInfo.Logs) > workerInfo.MaxLogs {
		workerInfo.Logs = workerInfo.Logs[len(workerInfo.Logs)-workerInfo.MaxLogs:]
	}
}

// GetWorkerLogs 특정 워커의 로그를 반환합니다
func (wm *WorkerManager) GetWorkerLogs(orderName string, limit int) ([]WorkerLog, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workerInfo, exists := wm.workerInfo[orderName]
	if !exists {
		return nil, fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	if limit <= 0 || limit > len(workerInfo.Logs) {
		limit = len(workerInfo.Logs)
	}

	// 최근 로그부터 반환
	start := len(workerInfo.Logs) - limit
	if start < 0 {
		start = 0
	}

	return workerInfo.Logs[start:], nil
}

// ClearWorkerLogs 특정 워커의 로그를 초기화합니다
func (wm *WorkerManager) ClearWorkerLogs(orderName string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workerInfo, exists := wm.workerInfo[orderName]
	if !exists {
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	workerInfo.Logs = []WorkerLog{}
	return nil
}
