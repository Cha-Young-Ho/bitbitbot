package platform

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BinanceWorker 바이낸스 거래소 워커
type BinanceWorker struct {
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
	mu        sync.RWMutex
}

// NewBinanceWorker 새로운 바이낸스 워커를 생성합니다
func NewBinanceWorker(config *WorkerConfig, storage *MemoryStorage) *BinanceWorker {
	return &BinanceWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.binance.com/api/v3/order",
	}
}

// Start 워커를 시작합니다
func (bw *BinanceWorker) Start(ctx context.Context) {
	bw.mu.Lock()
	bw.running = true
	bw.mu.Unlock()
	
	bw.storage.AddLog("info", "바이낸스 워커가 시작되었습니다.", bw.config.Exchange, bw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(bw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		bw.mu.RLock()
		if !bw.running {
			bw.mu.RUnlock()
			bw.storage.AddLog("info", "바이낸스 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		}
		bw.mu.RUnlock()

		select {
		case <-ctx.Done():
			bw.mu.Lock()
			bw.running = false
			bw.mu.Unlock()
			bw.storage.AddLog("info", "바이낸스 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		case <-bw.stopCh:
			bw.mu.Lock()
			bw.running = false
			bw.mu.Unlock()
			bw.storage.AddLog("info", "바이낸스 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			bw.mu.RLock()
			if bw.running {
				bw.mu.RUnlock()
				bw.executeSellOrder()
			} else {
				bw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (bw *BinanceWorker) Stop() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	
	if bw.running {
		bw.running = false
		close(bw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (bw *BinanceWorker) IsRunning() bool {
	bw.mu.RLock()
	defer bw.mu.RUnlock()
	return bw.running
}

// executeSellOrder 바이낸스에서 매도 주문 실행
func (bw *BinanceWorker) executeSellOrder() {
	// 실행 상태 재확인
	bw.mu.RLock()
	if !bw.running {
		bw.mu.RUnlock()
		return
	}
	bw.mu.RUnlock()

	timestamp := time.Now().UnixMilli()

	params := url.Values{}
	params.Set("symbol", strings.ReplaceAll(bw.config.Symbol, "/", ""))
	params.Set("side", "SELL")
	params.Set("type", "LIMIT")
	params.Set("timeInForce", "GTC")
	params.Set("quantity", fmt.Sprintf("%.8f", bw.config.SellAmount))
	params.Set("price", fmt.Sprintf("%.8f", bw.config.SellPrice))
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))

	// 서명 생성
	signature := bw.generateBinanceSignature(params.Encode())
	params.Set("signature", signature)

	req, err := http.NewRequest("POST", bw.url, strings.NewReader(params.Encode()))
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	req.Header.Set("X-MBX-APIKEY", bw.accessKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["orderId"].(float64)
		if ok {
			bw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%.0f, 가격=%.2f, 수량=%.8f",
				orderID, bw.config.SellPrice, bw.config.SellAmount), bw.config.Exchange, bw.config.Symbol)
		} else {
			bw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				bw.config.SellPrice, bw.config.SellAmount), bw.config.Exchange, bw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		bw.storage.AddLog("error", fmt.Sprintf("바이낸스 API 오류: %s", errorMsg), bw.config.Exchange, bw.config.Symbol)
	}
}

// generateBinanceSignature 바이낸스 HMAC-SHA256 서명 생성
func (bw *BinanceWorker) generateBinanceSignature(queryString string) string {
	h := hmac.New(sha256.New, []byte(bw.secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BinanceWorker) GetPlatformName() string {
	return "Binance"
}
