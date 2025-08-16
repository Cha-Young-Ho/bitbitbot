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
	workers       map[string]Worker      // key: orderName
	workerInfo    map[string]*WorkerInfo // key: orderName
	mu            sync.RWMutex
	logChan       chan WorkerLog
	systemLogChan chan SystemLog // 시스템 로그 채널
	ctx           context.Context
	cancel        context.CancelFunc
	localHandler  *local_file.Handler // 로컬 파일 핸들러
	wsClients     map[string]*Client  // 웹소켓 클라이언트들 (userID -> client)
	wsClientsMu   sync.RWMutex
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

type SystemLog struct {
	Component string    `json:"component"` // 컴포넌트명 (예: "WorkerManager", "WebSocket", "Handler")
	Function  string    `json:"function"`  // 함수명
	Message   string    `json:"message"`   // 메시지
	LogType   string    `json:"logType"`   // 로그 타입 ("info", "error", "warning")
	Timestamp time.Time `json:"timestamp"` // 타임스탬프
	UserID    string    `json:"userId"`    // 사용자 ID (선택적)
	OrderName string    `json:"orderName"` // 주문명 (선택적)
	Error     string    `json:"error"`     // 에러 메시지 (선택적)
}

// OutboundContent 웹소켓으로 내보낼 페이로드의 콘텐츠
type OutboundContent struct {
	ContentCategory string `json:"content_category"`
	Message         string `json:"message"`
}

// OutboundLog 웹소켓으로 내보낼 단일 통합 로그 모델
// category: "orderLog" | "systemLog"
// name: 주문 별칭(예약 매도 별칭)
// platform: 거래소 이름
// timestamp: 이벤트 시간
type OutboundLog struct {
	Category  string          `json:"category"`
	Name      string          `json:"name,omitempty"`
	Platform  string          `json:"platform,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Content   OutboundContent `json:"content"`
}

// OutboundEnvelope 단일 전송 래퍼. 프론트엔드가 자유롭게 파싱하도록 data에 원본 구조를 담습니다.
type OutboundEnvelope struct {
	Category string      `json:"category"`
	Data     interface{} `json:"data"`
}

// NewWorkerManager 새로운 워커 매니저를 생성합니다
func NewWorkerManager() *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerManager{
		workers:       make(map[string]Worker),
		workerInfo:    make(map[string]*WorkerInfo),
		logChan:       make(chan WorkerLog, 1000), // 버퍼 크기 1000
		systemLogChan: make(chan SystemLog, 1000), // 시스템 로그 채널
		ctx:           ctx,
		cancel:        cancel,
		localHandler:  local_file.NewHandler(),
		wsClients:     make(map[string]*Client),
	}
}

// AddWorker 워커를 추가합니다
func (workerManager *WorkerManager) AddWorker(orderName string, userID string, worker Worker) error {
	workerManager.mu.Lock()
	defer workerManager.mu.Unlock()

	log.Printf("AddWorker 호출: orderName=%s, userID=%s", orderName, userID)

	if _, exists := workerManager.workers[orderName]; exists {
		log.Printf("워커가 이미 존재함: %s", orderName)
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

	workerManager.workers[orderName] = worker
	workerManager.workerInfo[orderName] = workerInfo

	log.Printf("워커 추가 완료: orderName=%s, userID=%s", orderName, userID)

	// 로그 수집 고루틴 시작
	go workerManager.collectLogs(orderName)

	return nil
}

// RemoveWorker 워커를 제거합니다
func (workerManager *WorkerManager) RemoveWorker(orderName string) error {
	workerManager.mu.Lock()
	defer workerManager.mu.Unlock()

	worker, exists := workerManager.workers[orderName]
	if !exists {
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	// 워커 중지
	if err := worker.Stop(); err != nil {
		log.Printf("워커 중지 실패: %v", err)
	}

	delete(workerManager.workers, orderName)
	delete(workerManager.workerInfo, orderName)
	return nil
}

// StartWorker 특정 워커를 시작합니다
func (workerManager *WorkerManager) StartWorker(orderName string) error {
	workerManager.mu.RLock()
	worker, exists := workerManager.workers[orderName]
	workerManager.mu.RUnlock()

	log.Printf("StartWorker 호출: orderName=%s, exists=%v", orderName, exists)

	if !exists {
		log.Printf("워커를 찾을 수 없음: %s", orderName)
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	log.Printf("워커 시작 시도: %s", orderName)
	err := worker.Start(workerManager.ctx)
	if err != nil {
		log.Printf("워커 시작 실패: %s, error=%v", orderName, err)
	} else {
		log.Printf("워커 시작 성공: %s", orderName)
	}
	return err
}

// StopWorker 특정 워커를 중지합니다
func (workerManager *WorkerManager) StopWorker(orderName string) error {
	workerManager.mu.RLock()
	worker, exists := workerManager.workers[orderName]
	workerManager.mu.RUnlock()

	if !exists {
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", orderName)
	}

	return worker.Stop()
}

// GetWorkerStatus 모든 워커의 상태를 반환합니다
func (workerManager *WorkerManager) GetWorkerStatus() map[string]WorkerStatus {
	workerManager.mu.RLock()
	defer workerManager.mu.RUnlock()

	status := make(map[string]WorkerStatus)
	for orderName, worker := range workerManager.workers {
		status[orderName] = worker.GetStatus()
	}

	return status
}

// GetLogChannel 로그 채널을 반환합니다
func (workerManager *WorkerManager) GetLogChannel() <-chan WorkerLog {
	return workerManager.logChan
}

// GetUnifiedLogChannel 통합된 로그 채널을 반환합니다
// RegisterWebSocketClient 웹소켓 클라이언트를 등록합니다
func (workerManager *WorkerManager) RegisterWebSocketClient(userID string, client *Client) {
	workerManager.wsClientsMu.Lock()
	defer workerManager.wsClientsMu.Unlock()

	workerManager.wsClients[userID] = client
	log.Printf("웹소켓 클라이언트 등록: %s", userID)
}

// UnregisterWebSocketClient 웹소켓 클라이언트를 등록 해제합니다
func (workerManager *WorkerManager) UnregisterWebSocketClient(userID string, client *Client) {
	workerManager.wsClientsMu.Lock()
	defer workerManager.wsClientsMu.Unlock()

	if existingClient, exists := workerManager.wsClients[userID]; exists && existingClient == client {
		delete(workerManager.wsClients, userID)
		log.Printf("웹소켓 클라이언트 등록 해제: %s", userID)
	}
}

// SendLog 로그를 채널에 전송합니다
func (workerManager *WorkerManager) SendLog(workerLog WorkerLog) {
	log.Printf("SendLog 호출: %s - %s", workerLog.OrderName, workerLog.Message)

	select {
	case workerManager.logChan <- workerLog:
		log.Printf("로그 채널 전송 성공: %s", workerLog.Message)
	default:
		// 채널이 가득 찬 경우 로그를 버립니다
		log.Printf("로그 채널이 가득 찼습니다. 로그를 버립니다: %s", workerLog.Message)
	}

	// 단일 전송 래퍼로 전송 (프론트에서 파싱)
	workerManager.emitOutbound(workerLog.UserID, OutboundEnvelope{Category: "orderLog", Data: workerLog})
	// 시스템 로그에도 워커 로그 전송(통합 포맷)
	workerManager.SendSystemLog("Worker", "SendLog", workerLog.Message, workerLog.LogType, workerLog.UserID, workerLog.OrderName, "")
}

// SendSystemLog 시스템 로그를 전송합니다
func (workerManager *WorkerManager) SendSystemLog(component, function, message, logType string, userID, orderName, errorMsg string) {
	// 단일 전송 래퍼로 전송 (프론트에서 파싱)
	workerManager.emitOutbound("", OutboundEnvelope{Category: "systemLog", Data: SystemLog{
		Component: component,
		Function:  function,
		Message:   message,
		LogType:   logType,
		Timestamp: time.Now(),
		UserID:    userID,
		OrderName: orderName,
		Error:     errorMsg,
	}})
}

// sendOrderOutbound 워커 로그를 단일 포맷으로 전송
// emitOutbound 단일 전송 함수. userID가 빈 값이면 브로드캐스트, 있으면 대상 사용자에게 전송
func (workerManager *WorkerManager) emitOutbound(userID string, envelope OutboundEnvelope) {
	data, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("emitOutbound marshal error: %v", err)
		return
	}
	workerManager.wsClientsMu.RLock()
	defer workerManager.wsClientsMu.RUnlock()
	if userID == "" {
		for _, client := range workerManager.wsClients {
			client.SendMessage(data)
		}
		return
	}
	if client, ok := workerManager.wsClients[userID]; ok {
		client.SendMessage(data)
	}
}

// StartAllWorkers 모든 워커를 시작합니다
func (workerManager *WorkerManager) StartAllWorkers() error {
	workerManager.mu.RLock()
	defer workerManager.mu.RUnlock()

	for orderName, worker := range workerManager.workers {
		if err := worker.Start(workerManager.ctx); err != nil {
			log.Printf("워커 시작 실패 [%s]: %v", orderName, err)
			return err
		}
	}

	return nil
}

// StopAllWorkers 모든 워커를 중지합니다
func (workerManager *WorkerManager) StopAllWorkers() {
	workerManager.mu.RLock()
	defer workerManager.mu.RUnlock()

	for orderName, worker := range workerManager.workers {
		if err := worker.Stop(); err != nil {
			log.Printf("워커 중지 실패 [%s]: %v", orderName, err)
		}
	}
}

// Shutdown 워커 매니저를 종료합니다
func (workerManager *WorkerManager) Shutdown() {
	workerManager.StopAllWorkers()
	workerManager.cancel()
	close(workerManager.logChan)
}

// GetWorkerCount 활성 워커 수를 반환합니다
func (workerManager *WorkerManager) GetWorkerCount() int {
	workerManager.mu.RLock()
	defer workerManager.mu.RUnlock()

	return len(workerManager.workers)
}

// GetWorkerInfo 특정 워커의 정보를 반환합니다
func (workerManager *WorkerManager) GetWorkerInfo(orderName string) (*WorkerInfo, error) {
	workerManager.mu.RLock()
	defer workerManager.mu.RUnlock()

	workerInfo, exists := workerManager.workerInfo[orderName]
	if !exists {
		return nil, fmt.Errorf("워커 정보를 찾을 수 없습니다: %s", orderName)
	}

	return workerInfo, nil
}

// GetAllWorkerInfo 모든 워커의 정보를 반환합니다
func (workerManager *WorkerManager) GetAllWorkerInfo() map[string]*WorkerInfo {
	workerManager.mu.RLock()
	defer workerManager.mu.RUnlock()

	result := make(map[string]*WorkerInfo)
	for orderName, workerInfo := range workerManager.workerInfo {
		result[orderName] = workerInfo
	}

	return result
}

// GetWorkerInfoByUserID 특정 사용자의 워커 정보들을 반환합니다
func (workerManager *WorkerManager) GetWorkerInfoByUserID(userID string) map[string]*WorkerInfo {
	workerManager.mu.RLock()
	defer workerManager.mu.RUnlock()

	result := make(map[string]*WorkerInfo)
	for orderName, workerInfo := range workerManager.workerInfo {
		if workerInfo.UserID == userID {
			result[orderName] = workerInfo
		}
	}

	return result
}

// collectLogs 특정 워커의 로그를 수집합니다
func (workerManager *WorkerManager) collectLogs(orderName string) {
	logChan := workerManager.GetLogChannel()

	for {
		select {
		case <-workerManager.ctx.Done():
			return
		case workerLog := <-logChan:
			if workerLog.OrderName == orderName {
				workerManager.addLogToWorker(orderName, workerLog)
			}
		}
	}
}

// addLogToWorker 워커에 로그를 추가합니다
func (workerManager *WorkerManager) addLogToWorker(orderName string, workerLog WorkerLog) {
	workerManager.mu.Lock()
	defer workerManager.mu.Unlock()

	workerInfo, exists := workerManager.workerInfo[orderName]
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

// RemoveAllWorkers 모든 워커를 제거합니다
func (wm *WorkerManager) RemoveAllWorkers() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// 모든 워커 중지
	for orderName, worker := range wm.workers {
		if worker != nil {
			worker.Stop()
		}
		delete(wm.workers, orderName)
	}

	// 워커 정보 초기화
	wm.workerInfo = make(map[string]*WorkerInfo)

	log.Printf("모든 워커가 제거되었습니다")
}
