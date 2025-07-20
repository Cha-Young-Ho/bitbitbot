package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"gui-app/models"
	"gui-app/utils"
)

// DataService 데이터 서비스 (간소화)
type DataService struct {
	userKey string
	data    *models.AppData
	mutex   sync.RWMutex
}

// DashboardStats 대시보드 통계
type DashboardStats struct {
	ExchangeCount    int
	ActiveOrderCount int
	TotalOrderValue  float64
}

// NewDataService 새로운 데이터 서비스를 생성합니다
func NewDataService() *DataService {
	return &DataService{
		data: &models.AppData{},
	}
}

// SetUserKey 사용자 키를 설정합니다
func (ds *DataService) SetUserKey(key string) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	ds.userKey = key
	return nil
}

// LoadData 데이터를 로드합니다
func (ds *DataService) LoadData() error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// 데이터 파일 로드 시도
	data, err := utils.LoadEncryptedData(ds.userKey)
	if err != nil {
		// 새로운 데이터 초기화
		ds.data = &models.AppData{
			Exchanges:  []models.Exchange{},
			SellOrders: []models.SellOrder{},
		}
		log.Println("새로운 데이터 파일 생성")
		return nil
	}

	ds.data = data
	log.Printf("데이터 로드 완료: 거래소 %d개, 주문 %d개", len(ds.data.Exchanges), len(ds.data.SellOrders))
	return nil
}

// SaveData 데이터를 저장합니다 (외부 호출용)
func (ds *DataService) SaveData() error {
	fmt.Println("=== SaveData 시작 ===")
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	fmt.Println("SaveData 뮤텍스 락 획득")

	return ds.saveDataInternal()
}

// saveDataInternal 내부용 데이터 저장 (락 없음)
func (ds *DataService) saveDataInternal() error {
	fmt.Printf("저장할 데이터: 거래소 %d개, 주문 %d개\n", len(ds.data.Exchanges), len(ds.data.SellOrders))
	err := utils.SaveEncryptedData(ds.data, ds.userKey)
	if err != nil {
		fmt.Printf("SaveEncryptedData 실패: %v\n", err)
		return err
	}
	fmt.Println("SaveEncryptedData 성공")
	fmt.Println("=== SaveData 완료 ===")
	return err
}

// GetSupportedExchanges 지원하는 거래소 목록을 반환합니다
func (ds *DataService) GetSupportedExchanges() []models.ExchangeInfo {
	return models.GetSupportedExchanges()
}

// AddExchange 거래소를 추가합니다
func (ds *DataService) AddExchange(name, exchangeType, apiKey, secretKey string) error {
	fmt.Printf("=== AddExchange 시작 ===\n")
	fmt.Printf("이름: %s, 타입: %s, API Key: %s, Secret Key 길이: %d\n",
		name, exchangeType, apiKey, len(secretKey))

	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	fmt.Println("뮤텍스 락 획득")

	// 거래소 정보 확인
	fmt.Printf("거래소 정보 확인 중... 타입: %s\n", exchangeType)
	exchangeInfo := models.GetExchangeInfo(exchangeType)
	if exchangeInfo == nil {
		fmt.Printf("지원하지 않는 거래소 타입: %s\n", exchangeType)
		return fmt.Errorf("지원하지 않는 거래소 타입: %s", exchangeType)
	}
	fmt.Printf("거래소 정보 확인 완료: %s\n", exchangeInfo.DisplayName)

	// 거래소 ID 생성
	fmt.Println("ID 생성 시작...")
	id, err := ds.generateID()
	if err != nil {
		fmt.Printf("ID 생성 실패: %v\n", err)
		return fmt.Errorf("ID 생성 실패: %v", err)
	}
	fmt.Printf("ID 생성 완료: %s\n", id)

	// 새로운 거래소 생성
	fmt.Println("거래소 객체 생성 중...")
	exchange := models.Exchange{
		ID:        id,
		Name:      name,
		Type:      exchangeType,
		APIKey:    apiKey,
		SecretKey: secretKey,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	fmt.Printf("거래소 객체 생성 완료: %+v\n", exchange)

	// 거래소 추가
	fmt.Printf("기존 거래소 수: %d\n", len(ds.data.Exchanges))
	ds.data.Exchanges = append(ds.data.Exchanges, exchange)
	fmt.Printf("거래소 추가 후 수: %d\n", len(ds.data.Exchanges))

	// 데이터 저장
	fmt.Println("데이터 저장 시작...")
	if err := ds.saveDataInternal(); err != nil {
		fmt.Printf("데이터 저장 실패: %v\n", err)
		return fmt.Errorf("데이터 저장 실패: %v", err)
	}
	fmt.Println("데이터 저장 완료")

	fmt.Printf("거래소 추가 완료: %s (%s)\n", name, exchangeType)
	fmt.Println("=== AddExchange 완료 ===")
	return nil
}

// GetActiveExchanges 활성 거래소 목록을 반환합니다
func (ds *DataService) GetActiveExchanges() []models.Exchange {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	var activeExchanges []models.Exchange
	for _, exchange := range ds.data.Exchanges {
		if exchange.IsActive {
			activeExchanges = append(activeExchanges, exchange)
		}
	}

	return activeExchanges
}

// GetExchangeByID ID로 거래소를 조회합니다
func (ds *DataService) GetExchangeByID(id string) (*models.Exchange, error) {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	return ds.getExchangeByIDInternal(id)
}

// getExchangeByIDInternal 내부용 거래소 조회 (락 없음)
func (ds *DataService) getExchangeByIDInternal(id string) (*models.Exchange, error) {
	for _, exchange := range ds.data.Exchanges {
		if exchange.ID == id {
			return &exchange, nil
		}
	}
	return nil, fmt.Errorf("거래소를 찾을 수 없습니다: %s", id)
}

// AddSellOrder 매도 주문을 추가합니다
func (ds *DataService) AddSellOrder(exchangeID string, amount, price float64) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// 거래소 확인
	exchange, err := ds.getExchangeByIDInternal(exchangeID)
	if err != nil {
		return err
	}

	// 주문 ID 생성
	id, err := ds.generateID()
	if err != nil {
		return fmt.Errorf("ID 생성 실패: %v", err)
	}

	// 새로운 주문 생성
	order := models.SellOrder{
		ID:           id,
		ExchangeID:   exchangeID,
		ExchangeName: exchange.Name,
		Amount:       amount,
		Price:        price,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 주문 추가
	ds.data.SellOrders = append(ds.data.SellOrders, order)

	// 데이터 저장
	if err := ds.saveDataInternal(); err != nil {
		return fmt.Errorf("데이터 저장 실패: %v", err)
	}

	log.Printf("매도 주문 추가 완료: %.8f BTC @ %.2f KRW", amount, price)
	return nil
}

// GetSellOrders 모든 매도 주문을 반환합니다
func (ds *DataService) GetSellOrders() []models.SellOrder {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	return ds.data.SellOrders
}

// GetSellOrdersByExchange 특정 거래소의 매도 주문을 반환합니다
func (ds *DataService) GetSellOrdersByExchange(exchangeID string) []models.SellOrder {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	var orders []models.SellOrder
	for _, order := range ds.data.SellOrders {
		if order.ExchangeID == exchangeID {
			orders = append(orders, order)
		}
	}

	return orders
}

// UpdateSellOrder 매도 주문을 수정합니다
func (ds *DataService) UpdateSellOrder(id string, amount, price float64) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	for i, order := range ds.data.SellOrders {
		if order.ID == id {
			ds.data.SellOrders[i].Amount = amount
			ds.data.SellOrders[i].Price = price
			ds.data.SellOrders[i].UpdatedAt = time.Now()

			// 데이터 저장
			if err := ds.saveDataInternal(); err != nil {
				return fmt.Errorf("데이터 저장 실패: %v", err)
			}

			log.Printf("매도 주문 수정 완료: %s", id)
			return nil
		}
	}

	return fmt.Errorf("주문을 찾을 수 없습니다: %s", id)
}

// DeleteSellOrder 매도 주문을 삭제합니다
func (ds *DataService) DeleteSellOrder(id string) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	for i, order := range ds.data.SellOrders {
		if order.ID == id {
			// 주문 삭제
			ds.data.SellOrders = append(ds.data.SellOrders[:i], ds.data.SellOrders[i+1:]...)

			// 데이터 저장
			if err := ds.saveDataInternal(); err != nil {
				return fmt.Errorf("데이터 저장 실패: %v", err)
			}

			log.Printf("매도 주문 삭제 완료: %s", id)
			return nil
		}
	}

	return fmt.Errorf("주문을 찾을 수 없습니다: %s", id)
}

// GetDashboardStats 대시보드 통계를 반환합니다
func (ds *DataService) GetDashboardStats() DashboardStats {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	var exchangeCount, activeOrderCount int
	var totalOrderValue float64

	// 활성 거래소 수 계산
	for _, exchange := range ds.data.Exchanges {
		if exchange.IsActive {
			exchangeCount++
		}
	}

	// 활성 주문 수와 총 가치 계산
	for _, order := range ds.data.SellOrders {
		if order.Status == "active" {
			activeOrderCount++
			totalOrderValue += order.Amount
		}
	}

	return DashboardStats{
		ExchangeCount:    exchangeCount,
		ActiveOrderCount: activeOrderCount,
		TotalOrderValue:  totalOrderValue,
	}
}

// generateID 고유 ID를 생성합니다
func (ds *DataService) generateID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
