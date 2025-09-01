package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HuobiWorker 후오비 거래소 워커
type HuobiWorker struct {
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewHuobiWorker 새로운 후오비 워커를 생성합니다
func NewHuobiWorker(config *WorkerConfig, storage *MemoryStorage) *HuobiWorker {
	return &HuobiWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.huobi.pro/v1/order/orders/place",
	}
}

// Start 워커를 시작합니다
func (hw *HuobiWorker) Start(ctx context.Context) {
	hw.running = true
	hw.storage.AddLog("info", "후오비 워커가 시작되었습니다.", hw.config.Exchange, hw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(hw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hw.running = false
			hw.storage.AddLog("info", "후오비 워커가 중지되었습니다.", hw.config.Exchange, hw.config.Symbol)
			return
		case <-hw.stopCh:
			hw.running = false
			hw.storage.AddLog("info", "후오비 워커가 중지되었습니다.", hw.config.Exchange, hw.config.Symbol)
			return
		case <-ticker.C:
			hw.executeSellOrder()
		}
	}
}

// Stop 워커를 중지합니다
func (hw *HuobiWorker) Stop() {
	if hw.running {
		close(hw.stopCh)
		hw.running = false
	}
}

// IsRunning 워커 실행 상태 확인
func (hw *HuobiWorker) IsRunning() bool {
	return hw.running
}

// executeSellOrder 후오비에서 매도 주문 실행
func (hw *HuobiWorker) executeSellOrder() {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")

	requestBody := map[string]interface{}{
		"account-id": "12345678", // 계정 ID (실제로는 조회 필요)
		"symbol":     strings.ToLower(strings.ReplaceAll(hw.config.Symbol, "/", "")),
		"type":       "sell-limit",
		"amount":     fmt.Sprintf("%.8f", hw.config.SellAmount),
		"price":      fmt.Sprintf("%.8f", hw.config.SellPrice),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("JSON 변환 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	// 서명 생성
	signature := hw.createHuobiSignature(string(jsonBody), timestamp)

	req, err := http.NewRequest("POST", hw.url, bytes.NewReader(jsonBody))
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("AccessKeyId", hw.accessKey)
	req.Header.Set("SignatureMethod", "HmacSHA256")
	req.Header.Set("SignatureVersion", "2")
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("Signature", signature)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["data"].(string)
		if ok && orderID != "" {
			hw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, hw.config.SellPrice, hw.config.SellAmount), hw.config.Exchange, hw.config.Symbol)
		} else {
			hw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				hw.config.SellPrice, hw.config.SellAmount), hw.config.Exchange, hw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["err-msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["err-msg"])
		}
		hw.storage.AddLog("error", fmt.Sprintf("후오비 API 오류: %s", errorMsg), hw.config.Exchange, hw.config.Symbol)
	}
}

// createHuobiSignature 후오비 HMAC-SHA256 서명 생성
func (hw *HuobiWorker) createHuobiSignature(body string, timestamp string) string {
	message := "POST\napi.huobi.pro\n/v1/order/orders/place\n" + body
	h := hmac.New(sha256.New, []byte(hw.secretKey))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (hw *HuobiWorker) GetPlatformName() string {
	return "Huobi"
}
