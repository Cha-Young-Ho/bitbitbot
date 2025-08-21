package local_file

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Handler struct {
	filePath string
	data     []UserData
}

type UserData struct {
	ID              string        `json:"id"`
	Password        string        `json:"password"`
	PlatformKeyList []PlatformKey `json:"platformKeyList"`
	SellOrderList   []SellOrder   `json:"sellOrderList"`
}

type SellOrder struct {
	Name             string     `json:"name"`
	Symbol           string     `json:"symbol"`
	Price            float64    `json:"price"`
	Quantity         float64    `json:"quantity"`
	Term             float64    `json:"term"`
	Platform         string     `json:"platform"`
	PlatformNickName string     `json:"platformNickName"`
	Logs             []OrderLog `json:"logs,omitempty"`
	LastUpdated      time.Time  `json:"lastUpdated,omitempty"`
}

// UpdateSellOrder 사용자 예약 매도 주문을 수정합니다
func (h *Handler) UpdateSellOrder(userID string, oldName string, updated SellOrder) error {
	if userID == "" || oldName == "" {
		return fmt.Errorf("필수 인자가 비어있습니다")
	}
	for ui, user := range h.data {
		if user.ID != userID {
			continue
		}
		// 이름 중복 방지 (이름이 변경되는 경우)
		if updated.Name != oldName && updated.Name != "" {
			for _, o := range user.SellOrderList {
				if o.Name == updated.Name {
					return fmt.Errorf("이미 존재하는 매도 주문 별칭입니다: %s", updated.Name)
				}
			}
		}
		for oi, o := range user.SellOrderList {
			if o.Name == oldName {
				// 로그는 유지
				updated.Logs = o.Logs
				updated.LastUpdated = time.Now()
				h.data[ui].SellOrderList[oi] = updated
				return h.saveData()
			}
		}
		return fmt.Errorf("주문을 찾을 수 없습니다: %s", oldName)
	}
	return fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
}

// RemoveSellOrder 사용자 예약 매도 주문을 삭제합니다
func (h *Handler) RemoveSellOrder(userID string, name string) error {
	if userID == "" || name == "" {
		return fmt.Errorf("필수 인자가 비어있습니다")
	}
	for ui, user := range h.data {
		if user.ID != userID {
			continue
		}
		for oi, o := range user.SellOrderList {
			if o.Name == name {
				// 삭제
				h.data[ui].SellOrderList = append(user.SellOrderList[:oi], user.SellOrderList[oi+1:]...)
				return h.saveData()
			}
		}
		return fmt.Errorf("주문을 찾을 수 없습니다: %s", name)
	}
	return fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
}

type OrderLog struct {
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
	LogType     string    `json:"logType"`
	CheckCount  int       `json:"checkCount,omitempty"`
	ErrorCount  int       `json:"errorCount,omitempty"`
	LastPrice   float64   `json:"lastPrice,omitempty"`
	TargetPrice float64   `json:"targetPrice,omitempty"`
	OrderStatus string    `json:"orderStatus,omitempty"`
}

type PlatformKey struct {
	PlatformName      string `json:"platformName"`
	Name              string `json:"name"`
	PlatformAccessKey string `json:"platformAccessKey"`
	PlatformSecretKey string `json:"platformSecretKey"`
	PasswordPhrase    string `json:"passwordPhrase"`
}

func NewHandler() *Handler {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("홈 디렉터리를 찾을 수 없습니다: %v", err)
		return nil
	}

	directoryPath := filepath.Join(homeDir, "bit_info")
	filePath := filepath.Join(directoryPath, "bitbit_data.json")

	// 디렉터리 생성 시도
	if err := os.MkdirAll(directoryPath, 0755); err != nil {
		log.Printf("디렉터리 생성 실패: %v", err)
	}

	handler := &Handler{
		filePath: filePath,
		data:     []UserData{},
	}

	// 초기 데이터 로드
	if err := handler.loadData(); err != nil {
		log.Printf("데이터 로드 실패: %v", err)
		// 실패 시 빈 데이터로 시작
		handler.data = []UserData{}
	}

	return handler
}

// loadData 로컬 파일에서 데이터를 로드합니다
func (h *Handler) loadData() error {
	log.Printf("데이터 로드 시작: %s", h.filePath)

	// 디렉터리 생성
	directoryPath := filepath.Dir(h.filePath)
	if err := os.MkdirAll(directoryPath, 0755); err != nil {
		log.Printf("디렉터리 생성 실패: %v", err)
		return fmt.Errorf("디렉터리 생성 실패: %w", err)
	}
	log.Printf("디렉터리 생성 완료: %s", directoryPath)

	// 파일 존재 여부 확인
	fileInfo, err := os.Stat(h.filePath)
	if os.IsNotExist(err) {
		// 파일이 없으면 빈 데이터로 시작
		log.Printf("파일이 존재하지 않음: %s", h.filePath)
		h.data = []UserData{}
		return nil
	}

	// 파일 크기 확인 (빈 파일 처리)
	if fileInfo.Size() == 0 {
		h.data = []UserData{}
		return nil
	}

	// 파일 읽기
	log.Printf("파일 읽기 시도: %s", h.filePath)
	data, err := os.ReadFile(h.filePath)
	if err != nil {
		log.Printf("파일 읽기 실패: %v", err)
		return fmt.Errorf("파일 읽기 실패: %w", err)
	}
	log.Printf("파일 읽기 성공: %d bytes", len(data))

	// 빈 문자열이나 공백만 있는 경우 처리
	trimmedData := strings.TrimSpace(string(data))
	if trimmedData == "" {
		log.Printf("파일 내용이 비어있음")
		h.data = []UserData{}
		return nil
	}

	// JSON 파싱
	log.Printf("JSON 파싱 시도")
	if err := json.Unmarshal([]byte(trimmedData), &h.data); err != nil {
		// JSON 파싱 실패 시 빈 데이터로 초기화
		log.Printf("JSON 파싱 실패: %v", err)
		h.data = []UserData{}
		return nil
	}
	log.Printf("JSON 파싱 성공: %d명의 사용자 로드됨", len(h.data))
	return nil
}

// saveData 데이터를 로컬 파일에 저장합니다
func (h *Handler) saveData() error {
	// JSON으로 변환
	jsonData, err := json.MarshalIndent(h.data, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 변환 실패: %w", err)
	}

	// 파일에 저장
	if err := os.WriteFile(h.filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	return nil
}

// GetFilePath 파일 경로를 반환합니다
func (h *Handler) GetFilePath() string {
	return h.filePath
}

// GetUserByID 사용자 ID로 사용자 데이터를 조회합니다
func (h *Handler) GetUserByID(userID string) (*UserData, error) {
	if userID == "" {
		return nil, fmt.Errorf("사용자 ID가 비어있습니다")
	}

	for _, user := range h.data {
		if user.ID == userID {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
}

// AddUser 새로운 사용자를 추가합니다
func (h *Handler) AddUser(userData UserData) error {
	// 입력값 검증
	if userData.ID == "" {
		return fmt.Errorf("사용자 ID가 비어있습니다")
	}
	if userData.Password == "" {
		return fmt.Errorf("비밀번호가 비어있습니다")
	}

	// 기존 사용자 확인
	for _, existingUser := range h.data {
		if existingUser.ID == userData.ID {
			return fmt.Errorf("이미 존재하는 사용자 ID입니다: %s", userData.ID)
		}
	}

	// 새 사용자 추가
	h.data = append(h.data, userData)

	// 파일에 저장
	return h.saveData()
}

// UpdateUser 사용자 데이터를 업데이트합니다
func (h *Handler) UpdateUser(userData UserData) error {
	if userData.ID == "" {
		return fmt.Errorf("사용자 ID가 비어있습니다")
	}

	for i, existingUser := range h.data {
		if existingUser.ID == userData.ID {
			h.data[i] = userData
			return h.saveData()
		}
	}
	return fmt.Errorf("사용자를 찾을 수 없습니다: %s", userData.ID)
}

// GetAllUsers 모든 사용자 데이터를 반환합니다
func (h *Handler) GetAllUsers() []UserData {
	return h.data
}

// GetUserCount 사용자 수를 반환합니다
func (h *Handler) GetUserCount() int {
	return len(h.data)
}

// ReloadData 데이터를 다시 로드합니다
func (h *Handler) ReloadData() error {
	return h.loadData()
}

// GetUserData 사용자 데이터를 반환합니다
func (h *Handler) GetUserData(userID string) *UserData {
	for _, user := range h.data {
		if user.ID == userID {
			return &user
		}
	}
	return nil
}

// SaveUserDataFromJSON JSON 데이터로부터 사용자 데이터를 저장합니다
func (h *Handler) SaveUserDataFromJSON(userID string, jsonData string) error {
	var userData UserData
	if err := json.Unmarshal([]byte(jsonData), &userData); err != nil {
		return fmt.Errorf("JSON 파싱 오류: %v", err)
	}

	// 사용자 ID 검증
	if userData.ID != userID {
		return fmt.Errorf("사용자 ID가 일치하지 않습니다")
	}

	// 기존 사용자 데이터 업데이트 또는 새로 추가
	found := false
	for i, existingUser := range h.data {
		if existingUser.ID == userID {
			h.data[i] = userData
			found = true
			break
		}
	}

	if !found {
		h.data = append(h.data, userData)
	}

	return h.saveData()
}

// AddOrderLog 특정 사용자의 특정 주문에 로그를 추가합니다
func (h *Handler) AddOrderLog(userID, orderName string, log OrderLog) error {
	// 사용자 찾기
	userIndex := -1
	orderIndex := -1

	for i, user := range h.data {
		if user.ID == userID {
			userIndex = i
			// 주문 찾기
			for j, order := range user.SellOrderList {
				if order.Name == orderName {
					orderIndex = j
					break
				}
			}
			break
		}
	}

	if userIndex == -1 {
		return fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
	}

	if orderIndex == -1 {
		return fmt.Errorf("주문을 찾을 수 없습니다: %s", orderName)
	}

	// 로그 추가
	h.data[userIndex].SellOrderList[orderIndex].Logs = append(
		h.data[userIndex].SellOrderList[orderIndex].Logs,
		log,
	)

	// 최대 로그 개수 제한 (최근 100개만 유지)
	if len(h.data[userIndex].SellOrderList[orderIndex].Logs) > 100 {
		h.data[userIndex].SellOrderList[orderIndex].Logs =
			h.data[userIndex].SellOrderList[orderIndex].Logs[len(h.data[userIndex].SellOrderList[orderIndex].Logs)-100:]
	}

	// 마지막 업데이트 시간 설정
	h.data[userIndex].SellOrderList[orderIndex].LastUpdated = time.Now()

	// 파일에 저장
	return h.saveData()
}

// GetOrderLogs 특정 사용자의 특정 주문의 로그를 반환합니다
func (h *Handler) GetOrderLogs(userID, orderName string) ([]OrderLog, error) {
	for _, user := range h.data {
		if user.ID == userID {
			for _, order := range user.SellOrderList {
				if order.Name == orderName {
					return order.Logs, nil
				}
			}
			return nil, fmt.Errorf("주문을 찾을 수 없습니다: %s", orderName)
		}
	}
	return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
}

// ClearOrderLogs 특정 사용자의 특정 주문의 로그를 초기화합니다
func (h *Handler) ClearOrderLogs(userID, orderName string) error {
	for i, user := range h.data {
		if user.ID == userID {
			for j, order := range user.SellOrderList {
				if order.Name == orderName {
					h.data[i].SellOrderList[j].Logs = []OrderLog{}
					h.data[i].SellOrderList[j].LastUpdated = time.Now()
					return h.saveData()
				}
			}
			return fmt.Errorf("주문을 찾을 수 없습니다: %s", orderName)
		}
	}
	return fmt.Errorf("사용자를 찾을 수 없습니다: %s", userID)
}
