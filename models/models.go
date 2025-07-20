package models

import "time"

// SellOrder 매도 주문 정보
type SellOrder struct {
	ID           string    `json:"id"`
	ExchangeID   string    `json:"exchange_id"`
	ExchangeName string    `json:"exchange_name"`
	Amount       float64   `json:"amount"`
	Price        float64   `json:"price"`
	Status       string    `json:"status"` // "active", "completed", "cancelled"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AppData 애플리케이션 데이터
type AppData struct {
	Exchanges  []Exchange  `json:"exchanges"`
	SellOrders []SellOrder `json:"sell_orders"`
	Version    string      `json:"version"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// CreateOrder 주문 생성 요청
type CreateOrderRequest struct {
	ExchangeID string  `json:"exchange_id"`
	Amount     float64 `json:"amount"`
	Price      float64 `json:"price"`
}

// UpdateOrder 주문 수정 요청
type UpdateOrderRequest struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Price  float64 `json:"price"`
}
