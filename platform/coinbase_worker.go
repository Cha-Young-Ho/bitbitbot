package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CoinbaseWorker 코인베이스 거래소 워커
type CoinbaseWorker struct {
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewCoinbaseWorker 새로운 코인베이스 워커를 생성합니다
func NewCoinbaseWorker(config *WorkerConfig, storage *MemoryStorage) *CoinbaseWorker {
	return &CoinbaseWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.exchange.coinbase.com/orders",
	}
}

// Start 워커를 시작합니다
func (cbw *CoinbaseWorker) Start(ctx context.Context) {
	cbw.running = true
	cbw.storage.AddLog("info", "코인베이스 워커가 시작되었습니다.", cbw.config.Exchange, cbw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(cbw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cbw.running = false
			cbw.storage.AddLog("info", "코인베이스 워커가 중지되었습니다.", cbw.config.Exchange, cbw.config.Symbol)
			return
		case <-cbw.stopCh:
			cbw.running = false
			cbw.storage.AddLog("info", "코인베이스 워커가 중지되었습니다.", cbw.config.Exchange, cbw.config.Symbol)
			return
		case <-ticker.C:
			cbw.executeSellOrder()
		}
	}
}

// Stop 워커를 중지합니다
func (cbw *CoinbaseWorker) Stop() {
	if cbw.running {
		close(cbw.stopCh)
		cbw.running = false
	}
}

// IsRunning 워커 실행 상태 확인
func (cbw *CoinbaseWorker) IsRunning() bool {
	return cbw.running
}

// executeSellOrder 코인베이스에서 매도 주문 실행
func (cbw *CoinbaseWorker) executeSellOrder() {
	timestamp := time.Now().Unix()

	requestBody := map[string]interface{}{
		"product_id":    strings.ReplaceAll(cbw.config.Symbol, "/", "-"),
		"side":          "sell",
		"type":          "limit",
		"size":          fmt.Sprintf("%.8f", cbw.config.SellAmount),
		"price":         fmt.Sprintf("%.8f", cbw.config.SellPrice),
		"time_in_force": "GTC",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		cbw.storage.AddLog("error", fmt.Sprintf("JSON 변환 실패: %v", err), cbw.config.Exchange, cbw.config.Symbol)
		return
	}

	// 서명 생성
	signature := cbw.createCoinbaseSignature(string(jsonBody), timestamp)

	req, err := http.NewRequest("POST", cbw.url, bytes.NewReader(jsonBody))
	if err != nil {
		cbw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), cbw.config.Exchange, cbw.config.Symbol)
		return
	}

	req.Header.Set("CB-ACCESS-KEY", cbw.accessKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		cbw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), cbw.config.Exchange, cbw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		cbw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), cbw.config.Exchange, cbw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["id"].(string)
		if ok && orderID != "" {
			cbw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, cbw.config.SellPrice, cbw.config.SellAmount), cbw.config.Exchange, cbw.config.Symbol)
		} else {
			cbw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				cbw.config.SellPrice, cbw.config.SellAmount), cbw.config.Exchange, cbw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["message"] != nil {
			errorMsg = fmt.Sprintf("%v", result["message"])
		}
		cbw.storage.AddLog("error", fmt.Sprintf("코인베이스 API 오류: %s", errorMsg), cbw.config.Exchange, cbw.config.Symbol)
	}
}

// createCoinbaseSignature 코인베이스 HMAC-SHA256 서명 생성
func (cbw *CoinbaseWorker) createCoinbaseSignature(body string, timestamp int64) string {
	message := strconv.FormatInt(timestamp, 10) + "POST" + "/orders" + body
	h := hmac.New(sha256.New, []byte(cbw.secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (cbw *CoinbaseWorker) GetPlatformName() string {
	return "Coinbase"
}
