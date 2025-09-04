package platform

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExchangeKey 거래소 키 정보
type ExchangeKey struct {
	ID             string    `json:"id"`             // 고유 ID
	Exchange       string    `json:"exchange"`       // 거래소 이름
	AccessKey      string    `json:"accessKey"`      // Access Key
	SecretKey      string    `json:"secretKey"`      // Secret Key
	PasswordPhrase string    `json:"passwordPhrase"` // Password Phrase (필요한 경우)
	CreatedAt      time.Time `json:"createdAt"`      // 생성 시간
	UpdatedAt      time.Time `json:"updatedAt"`      // 수정 시간
	IsActive       bool      `json:"isActive"`       // 활성 상태
}

// KeyStorage 키 저장소
type KeyStorage struct {
	mu        sync.RWMutex
	keys      map[string]*ExchangeKey // ID -> ExchangeKey
	filePath  string                  // 저장 파일 경로
	lastSave  time.Time               // 마지막 저장 시간
}

// NewKeyStorage 새로운 키 저장소 생성
func NewKeyStorage() *KeyStorage {
	// 크로스 플랫폼 호환 가능한 설정 디렉토리 생성
	configDir, err := getConfigDirectory()
	if err != nil {
		// 모든 방법이 실패하면 현재 디렉토리 사용
		configDir = "."
	}
	
	filePath := filepath.Join(configDir, "exchange_keys.json")
	
	storage := &KeyStorage{
		keys:     make(map[string]*ExchangeKey),
		filePath: filePath,
		lastSave: time.Now(),
	}
	
	// 기존 키 로드
	storage.loadKeys()
	
	return storage
}

// getConfigDirectory 크로스 플랫폼 호환 가능한 설정 디렉토리 반환
func getConfigDirectory() (string, error) {
	// 1. 사용자 홈 디렉토리 시도
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(homeDir, ".bitbitbot")
		if err := os.MkdirAll(configDir, 0755); err == nil {
			return configDir, nil
		}
	}
	
	// 2. 사용자 문서 디렉토리 시도 (Windows에서 더 안전)
	userDir, err := os.UserConfigDir()
	if err == nil {
		configDir := filepath.Join(userDir, "BitBit")
		if err := os.MkdirAll(configDir, 0755); err == nil {
			return configDir, nil
		}
	}
	
	// 3. 임시 디렉토리 시도
	tempDir := os.TempDir()
	if tempDir != "" {
		configDir := filepath.Join(tempDir, "bitbitbot")
		if err := os.MkdirAll(configDir, 0755); err == nil {
			return configDir, nil
		}
	}
	
	// 4. 실행 파일과 같은 디렉토리 시도
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		configDir := filepath.Join(execDir, "config")
		if err := os.MkdirAll(configDir, 0755); err == nil {
			return configDir, nil
		}
	}
	
	// 5. 현재 작업 디렉토리 시도
	currentDir, err := os.Getwd()
	if err == nil {
		configDir := filepath.Join(currentDir, "config")
		if err := os.MkdirAll(configDir, 0755); err == nil {
			return configDir, nil
		}
	}
	
	return "", fmt.Errorf("설정 디렉토리를 생성할 수 없습니다")
}

// AddKey 키 추가
func (ks *KeyStorage) AddKey(exchange, accessKey, secretKey, passwordPhrase string) (*ExchangeKey, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	
	// 입력값 검증
	if exchange == "" {
		return nil, fmt.Errorf("거래소를 선택해주세요")
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("Access Key와 Secret Key를 입력해주세요")
	}
	
	// 중복 키 확인 (같은 거래소의 같은 Access Key)
	for _, key := range ks.keys {
		if key.Exchange == exchange && key.AccessKey == accessKey {
			return nil, fmt.Errorf("이미 등록된 키입니다")
		}
	}
	
	// 새 키 생성
	keyID := fmt.Sprintf("%s_%d", exchange, time.Now().Unix())
	key := &ExchangeKey{
		ID:             keyID,
		Exchange:       exchange,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		PasswordPhrase: passwordPhrase,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		IsActive:       true,
	}
	
	ks.keys[keyID] = key
	
	// 파일에 저장
	if err := ks.saveKeys(); err != nil {
		// 저장 실패 시 키 제거
		delete(ks.keys, keyID)
		return nil, fmt.Errorf("키 저장 실패: %v", err)
	}
	
	return key, nil
}

// UpdateKey 키 수정
func (ks *KeyStorage) UpdateKey(keyID, exchange, accessKey, secretKey, passwordPhrase string) (*ExchangeKey, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	
	key, exists := ks.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("키를 찾을 수 없습니다")
	}
	
	// 입력값 검증
	if exchange == "" {
		return nil, fmt.Errorf("거래소를 선택해주세요")
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("Access Key와 Secret Key를 입력해주세요")
	}
	
	// 다른 키와 중복 확인 (자신 제외)
	for id, existingKey := range ks.keys {
		if id != keyID && existingKey.Exchange == exchange && existingKey.AccessKey == accessKey {
			return nil, fmt.Errorf("이미 등록된 키입니다")
		}
	}
	
	// 키 정보 업데이트
	key.Exchange = exchange
	key.AccessKey = accessKey
	key.SecretKey = secretKey
	key.PasswordPhrase = passwordPhrase
	key.UpdatedAt = time.Now()
	
	// 파일에 저장
	if err := ks.saveKeys(); err != nil {
		return nil, fmt.Errorf("키 저장 실패: %v", err)
	}
	
	return key, nil
}

// DeleteKey 키 삭제
func (ks *KeyStorage) DeleteKey(keyID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	
	_, exists := ks.keys[keyID]
	if !exists {
		return fmt.Errorf("키를 찾을 수 없습니다")
	}
	
	delete(ks.keys, keyID)
	
	// 파일에 저장
	if err := ks.saveKeys(); err != nil {
		return fmt.Errorf("키 저장 실패: %v", err)
	}
	
	return nil
}

// GetKey 키 조회
func (ks *KeyStorage) GetKey(keyID string) (*ExchangeKey, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	
	key, exists := ks.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("키를 찾을 수 없습니다")
	}
	
	return key, nil
}

// GetAllKeys 모든 키 조회
func (ks *KeyStorage) GetAllKeys() []*ExchangeKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	
	keys := make([]*ExchangeKey, 0, len(ks.keys))
	for _, key := range ks.keys {
		keys = append(keys, key)
	}
	
	return keys
}

// GetKeysByExchange 거래소별 키 조회
func (ks *KeyStorage) GetKeysByExchange(exchange string) []*ExchangeKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	
	keys := make([]*ExchangeKey, 0)
	for _, key := range ks.keys {
		if key.Exchange == exchange && key.IsActive {
			keys = append(keys, key)
		}
	}
	
	return keys
}

// GetActiveKeys 활성 키만 조회
func (ks *KeyStorage) GetActiveKeys() []*ExchangeKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	
	keys := make([]*ExchangeKey, 0)
	for _, key := range ks.keys {
		if key.IsActive {
			keys = append(keys, key)
		}
	}
	
	return keys
}

// SetKeyActive 키 활성/비활성 상태 변경
func (ks *KeyStorage) SetKeyActive(keyID string, isActive bool) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	
	key, exists := ks.keys[keyID]
	if !exists {
		return fmt.Errorf("키를 찾을 수 없습니다")
	}
	
	key.IsActive = isActive
	key.UpdatedAt = time.Now()
	
	// 파일에 저장
	if err := ks.saveKeys(); err != nil {
		return fmt.Errorf("키 저장 실패: %v", err)
	}
	
	return nil
}

// loadKeys 파일에서 키 로드
func (ks *KeyStorage) loadKeys() error {
	// 파일이 존재하지 않으면 빈 맵으로 시작
	if _, err := os.Stat(ks.filePath); os.IsNotExist(err) {
		return nil
	}
	
	data, err := ioutil.ReadFile(ks.filePath)
	if err != nil {
		return fmt.Errorf("키 파일 읽기 실패: %v", err)
	}
	
	var keys []*ExchangeKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("키 데이터 파싱 실패: %v", err)
	}
	
	// 맵으로 변환
	ks.keys = make(map[string]*ExchangeKey)
	for _, key := range keys {
		ks.keys[key.ID] = key
	}
	
	return nil
}

// saveKeys 파일에 키 저장
func (ks *KeyStorage) saveKeys() error {
	// 맵을 슬라이스로 변환
	keys := make([]*ExchangeKey, 0, len(ks.keys))
	for _, key := range ks.keys {
		keys = append(keys, key)
	}
	
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("키 데이터 직렬화 실패: %v", err)
	}
	
	// 디렉토리가 존재하는지 확인하고 생성
	dir := filepath.Dir(ks.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %v", err)
	}
	
	// 임시 파일에 먼저 저장 (크로스 플랫폼 호환)
	tempFile := ks.filePath + ".tmp"
	
	// 파일 권한을 크로스 플랫폼 호환으로 설정
	fileMode := os.FileMode(0600) // 소유자만 읽기/쓰기
	if err := ioutil.WriteFile(tempFile, data, fileMode); err != nil {
		return fmt.Errorf("임시 파일 저장 실패: %v", err)
	}
	
	// 원자적 이동 (크로스 플랫폼 호환)
	if err := os.Rename(tempFile, ks.filePath); err != nil {
		// 실패 시 임시 파일 정리
		os.Remove(tempFile)
		return fmt.Errorf("파일 이동 실패: %v", err)
	}
	
	ks.lastSave = time.Now()
	return nil
}

// GetFilePath 저장 파일 경로 반환
func (ks *KeyStorage) GetFilePath() string {
	return ks.filePath
}

// GetLastSave 마지막 저장 시간 반환
func (ks *KeyStorage) GetLastSave() time.Time {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.lastSave
}

// GetKeyCount 키 개수 반환
func (ks *KeyStorage) GetKeyCount() int {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return len(ks.keys)
}

// GetActiveKeyCount 활성 키 개수 반환
func (ks *KeyStorage) GetActiveKeyCount() int {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	
	count := 0
	for _, key := range ks.keys {
		if key.IsActive {
			count++
		}
	}
	return count
}

// GetConfigInfo 설정 디렉토리 정보 반환
func (ks *KeyStorage) GetConfigInfo() map[string]interface{} {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	
	// 파일 존재 여부 확인
	fileExists := false
	fileSize := int64(0)
	if stat, err := os.Stat(ks.filePath); err == nil {
		fileExists = true
		fileSize = stat.Size()
	}
	
	return map[string]interface{}{
		"configDir":  filepath.Dir(ks.filePath),
		"filePath":   ks.filePath,
		"fileExists": fileExists,
		"fileSize":   fileSize,
		"lastSave":   ks.lastSave,
		"keyCount":   len(ks.keys),
		"activeKeys": ks.GetActiveKeyCount(),
	}
}
