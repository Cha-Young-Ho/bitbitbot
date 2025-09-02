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
	"strconv"
	"strings"
	"sync"
	"time"
)

// KuCoinWorker 쿠코인 거래소 워커
type KuCoinWorker struct {
	mu        sync.RWMutex
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewKuCoinWorker 새로운 쿠코인 워커를 생성합니다
func NewKuCoinWorker(config *WorkerConfig, storage *MemoryStorage) *KuCoinWorker {
	return &KuCoinWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.kucoin.com/api/v1/orders",
	}
}

// Start 워커를 시작합니다
func (kcw *KuCoinWorker) Start(ctx context.Context) {
	kcw.mu.Lock()
	kcw.running = true
	kcw.mu.Unlock()
	
	kcw.storage.AddLog("info", "쿠코인 워커가 시작되었습니다.", kcw.config.Exchange, kcw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(kcw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		kcw.mu.RLock()
		if !kcw.running {
			kcw.mu.RUnlock()
			kcw.storage.AddLog("info", "쿠코인 워커가 중지되었습니다.", kcw.config.Exchange, kcw.config.Symbol)
			return
		}
		kcw.mu.RUnlock()

		select {
		case <-ctx.Done():
			kcw.mu.Lock()
			kcw.running = false
			kcw.mu.Unlock()
			kcw.storage.AddLog("info", "쿠코인 워커가 중지되었습니다.", kcw.config.Exchange, kcw.config.Symbol)
			return
		case <-kcw.stopCh:
			kcw.mu.Lock()
			kcw.running = false
			kcw.mu.Unlock()
			kcw.storage.AddLog("info", "쿠코인 워커가 중지되었습니다.", kcw.config.Exchange, kcw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			kcw.mu.RLock()
			if kcw.running {
				kcw.mu.RUnlock()
				kcw.executeSellOrder()
			} else {
				kcw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (kcw *KuCoinWorker) Stop() {
	kcw.mu.Lock()
	defer kcw.mu.Unlock()
	
	if kcw.running {
		kcw.running = false
		close(kcw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (kcw *KuCoinWorker) IsRunning() bool {
	kcw.mu.RLock()
	defer kcw.mu.RUnlock()
	return kcw.running
}

// executeSellOrder 쿠코인에서 매도 주문 실행
func (kcw *KuCoinWorker) executeSellOrder() {
	// 실행 상태 재확인
	kcw.mu.RLock()
	if !kcw.running {
		kcw.mu.RUnlock()
		return
	}
	kcw.mu.RUnlock()

	timestamp := time.Now().UnixMilli()

	requestBody := map[string]interface{}{
		"clientOid":   fmt.Sprintf("sell_%d", timestamp),
		"symbol":      strings.ReplaceAll(kcw.config.Symbol, "/", "-"),
		"side":        "sell",
		"type":        "limit",
		"size":        fmt.Sprintf("%.8f", kcw.config.SellAmount),
		"price":       fmt.Sprintf("%.8f", kcw.config.SellPrice),
		"timeInForce": "GTC",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("JSON 변환 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}

	// 서명 생성
	signature := kcw.createKuCoinSignature(string(jsonBody), timestamp)

	req, err := http.NewRequest("POST", kcw.url, bytes.NewReader(jsonBody))
	if err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}

	req.Header.Set("KC-API-KEY", kcw.accessKey)
	req.Header.Set("KC-API-SIGN", signature)
	req.Header.Set("KC-API-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["orderId"].(string)
		if ok && orderID != "" {
			kcw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, kcw.config.SellPrice, kcw.config.SellAmount), kcw.config.Exchange, kcw.config.Symbol)
		} else {
			kcw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				kcw.config.SellPrice, kcw.config.SellAmount), kcw.config.Exchange, kcw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		kcw.storage.AddLog("error", fmt.Sprintf("쿠코인 API 오류: %s", errorMsg), kcw.config.Exchange, kcw.config.Symbol)
	}
}

// createKuCoinSignature 쿠코인 HMAC-SHA256 서명 생성
func (kcw *KuCoinWorker) createKuCoinSignature(body string, timestamp int64) string {
	message := strconv.FormatInt(timestamp, 10) + "POST" + "/api/v1/orders" + body
	h := hmac.New(sha256.New, []byte(kcw.secretKey))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (kcw *KuCoinWorker) GetPlatformName() string {
	return "KuCoin"
}
