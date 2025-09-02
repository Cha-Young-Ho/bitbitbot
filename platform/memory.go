package platform

import (
	"sync"
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

// MemoryStorage 메모리 기반 저장소
type MemoryStorage struct {
	mu            sync.RWMutex
	workerConfigs map[string]*WorkerConfig
	workerStatus  map[string]*WorkerStatus
	logs          []LogEntry
	lastUpdate    time.Time
}

// WorkerConfig 워커 설정
type WorkerConfig struct {
	Exchange        string  `json:"exchange"`
	AccessKey       string  `json:"accessKey"`
	SecretKey       string  `json:"secretKey"`
	PasswordPhrase  string  `json:"passwordPhrase"`  // Password Phrase (필요한 거래소용)
	RequestInterval float64 `json:"requestInterval"` // 초 단위
	Symbol          string  `json:"symbol"`
	SellAmount      float64 `json:"sellAmount"` // 매도할 수량
	SellPrice       float64 `json:"sellPrice"`  // 매도 가격
}

// WorkerStatus 워커 상태
type WorkerStatus struct {
	IsRunning    bool      `json:"isRunning"`
	StartedAt    time.Time `json:"startedAt"`
	LastRequest  time.Time `json:"lastRequest"`
	RequestCount int64     `json:"requestCount"`
	ErrorCount   int64     `json:"errorCount"`
}

// LogEntry 로그 엔트리
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, error, success
	Message   string    `json:"message"`
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
}

// NewMemoryStorage 새로운 메모리 저장소 생성
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		workerConfigs: make(map[string]*WorkerConfig),
		workerStatus:  make(map[string]*WorkerStatus),
		logs:          make([]LogEntry, 0),
		lastUpdate:    time.Now(),
	}
}

// SetWorkerConfig 워커 설정 저장
func (s *MemoryStorage) SetWorkerConfig(workerKey string, config *WorkerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.workerConfigs[workerKey] = config
	s.lastUpdate = time.Now()
}

// GetWorkerConfig 워커 설정 조회
func (s *MemoryStorage) GetWorkerConfig(workerKey string) *WorkerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.workerConfigs[workerKey]
}

// SetWorkerStatus 워커 상태 설정
func (s *MemoryStorage) SetWorkerStatus(workerKey string, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workerStatus := &WorkerStatus{
		IsRunning:    status == "running",
		StartedAt:    time.Now(),
		LastRequest:  time.Now(),
		RequestCount: 0,
		ErrorCount:   0,
	}

	s.workerStatus[workerKey] = workerStatus
	s.lastUpdate = time.Now()
}

// GetWorkerStatus 워커 상태 조회
func (s *MemoryStorage) GetWorkerStatus(workerKey string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := s.workerStatus[workerKey]
	if status == nil {
		return "stopped"
	}

	if status.IsRunning {
		return "running"
	}
	return "stopped"
}

// GetAllWorkerStatus 모든 워커 상태 조회
func (s *MemoryStorage) GetAllWorkerStatus() map[string]*WorkerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*WorkerStatus)
	for key, status := range s.workerStatus {
		result[key] = status
	}
	return result
}

// AddLog 로그 추가
func (s *MemoryStorage) AddLog(level, message, exchange, symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	logEntry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Exchange:  exchange,
		Symbol:    symbol,
	}

	s.logs = append(s.logs, logEntry)
	s.lastUpdate = time.Now()

	// 로그 개수 제한 (최대 1000개)
	if len(s.logs) > 1000 {
		s.logs = s.logs[len(s.logs)-1000:]
	}
}

// GetLogs 로그 조회
func (s *MemoryStorage) GetLogs(limit int) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.logs) {
		limit = len(s.logs)
	}

	// 최신 로그부터 반환
	start := len(s.logs) - limit
	if start < 0 {
		start = 0
	}

	result := make([]LogEntry, limit)
	copy(result, s.logs[start:])
	return result
}

// ClearLogs 로그 초기화
func (s *MemoryStorage) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = make([]LogEntry, 0)
	s.lastUpdate = time.Now()
}

// GetLastUpdate 마지막 업데이트 시간 조회
func (s *MemoryStorage) GetLastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastUpdate
}
