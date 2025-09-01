package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GateWorker Gate.io 거래소 워커
type GateWorker struct {
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewGateWorker 새로운 Gate.io 워커를 생성합니다
func NewGateWorker(config *WorkerConfig, storage *MemoryStorage) *GateWorker {
	return &GateWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.gateio.ws/api/v4/spot/orders",
	}
}

// Start 워커를 시작합니다
func (gw *GateWorker) Start(ctx context.Context) {
	gw.running = true
	gw.storage.AddLog("info", "Gate.io 워커가 시작되었습니다.", gw.config.Exchange, gw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(gw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			gw.running = false
			gw.storage.AddLog("info", "Gate.io 워커가 중지되었습니다.", gw.config.Exchange, gw.config.Symbol)
			return
		case <-gw.stopCh:
			gw.running = false
			gw.storage.AddLog("info", "Gate.io 워커가 중지되었습니다.", gw.config.Exchange, gw.config.Symbol)
			return
		case <-ticker.C:
			gw.executeSellOrder()
		}
	}
}

// Stop 워커를 중지합니다
func (gw *GateWorker) Stop() {
	if gw.running {
		close(gw.stopCh)
		gw.running = false
	}
}

// IsRunning 워커 실행 상태 확인
func (gw *GateWorker) IsRunning() bool {
	return gw.running
}

// executeSellOrder Gate.io에서 매도 주문 실행
func (gw *GateWorker) executeSellOrder() {
	timestamp := time.Now().Unix()

	requestBody := map[string]interface{}{
		"currency_pair": strings.ReplaceAll(gw.config.Symbol, "/", "_"),
		"side":          "sell",
		"type":          "limit",
		"amount":        fmt.Sprintf("%.8f", gw.config.SellAmount),
		"price":         fmt.Sprintf("%.8f", gw.config.SellPrice),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		gw.storage.AddLog("error", fmt.Sprintf("JSON 변환 실패: %v", err), gw.config.Exchange, gw.config.Symbol)
		return
	}

	// 서명 생성
	signature := gw.createGateSignature(string(jsonBody), timestamp)

	req, err := http.NewRequest("POST", gw.url, bytes.NewReader(jsonBody))
	if err != nil {
		gw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), gw.config.Exchange, gw.config.Symbol)
		return
	}

	req.Header.Set("KEY", gw.accessKey)
	req.Header.Set("SIGN", signature)
	req.Header.Set("Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		gw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), gw.config.Exchange, gw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		gw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), gw.config.Exchange, gw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["id"].(string)
		if ok && orderID != "" {
			gw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, gw.config.SellPrice, gw.config.SellAmount), gw.config.Exchange, gw.config.Symbol)
		} else {
			gw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				gw.config.SellPrice, gw.config.SellAmount), gw.config.Exchange, gw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["message"] != nil {
			errorMsg = fmt.Sprintf("%v", result["message"])
		}
		gw.storage.AddLog("error", fmt.Sprintf("Gate.io API 오류: %s", errorMsg), gw.config.Exchange, gw.config.Symbol)
	}
}

// createGateSignature Gate.io HMAC-SHA512 서명 생성
func (gw *GateWorker) createGateSignature(body string, timestamp int64) string {
	message := "POST\n/api/v4/spot/orders\n" + body + "\n" + strconv.FormatInt(timestamp, 10)
	h := hmac.New(sha512.New, []byte(gw.secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (gw *GateWorker) GetPlatformName() string {
	return "Gate.io"
}
