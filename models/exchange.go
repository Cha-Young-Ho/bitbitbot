package models

import (
	"time"
)

// Exchange 거래소 정보 (간소화)
type Exchange struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"` // 사용자가 설정한 별칭
	Type      string    `json:"type"` // "upbit", "binance", "bithumb"
	APIKey    string    `json:"api_key"`
	SecretKey string    `json:"secret_key"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExchangeInfo 하드코딩된 거래소 정보
type ExchangeInfo struct {
	Type         string
	DisplayName  string
	Logo         string
	BaseURL      string
	WebSocketURL string
	TradingFee   float64
}

// GetSupportedExchanges 지원하는 거래소 목록을 반환합니다
func GetSupportedExchanges() []ExchangeInfo {
	return []ExchangeInfo{
		{
			Type:         "upbit",
			DisplayName:  "업비트 (Upbit)",
			Logo:         "🇰🇷",
			BaseURL:      "https://api.upbit.com",
			WebSocketURL: "wss://api.upbit.com/websocket/v1",
			TradingFee:   0.0005, // 0.05%
		},
		{
			Type:         "binance",
			DisplayName:  "바이낸스 (Binance)",
			Logo:         "🌍",
			BaseURL:      "https://api.binance.com",
			WebSocketURL: "wss://stream.binance.com:9443/ws",
			TradingFee:   0.001, // 0.1%
		},
		{
			Type:         "bithumb",
			DisplayName:  "빗썸 (Bithumb)",
			Logo:         "🇰🇷",
			BaseURL:      "https://api.bithumb.com",
			WebSocketURL: "wss://pubwss.bithumb.com/pub/ws",
			TradingFee:   0.0025, // 0.25%
		},
	}
}

// GetExchangeInfo 특정 타입의 거래소 정보를 반환합니다
func GetExchangeInfo(exchangeType string) *ExchangeInfo {
	for _, info := range GetSupportedExchanges() {
		if info.Type == exchangeType {
			return &info
		}
	}
	return nil
}
