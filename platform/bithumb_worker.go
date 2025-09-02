package platform

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BithumbWorker 빗썸 거래소 워커
type BithumbWorker struct {
	mu        sync.RWMutex
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewBithumbWorker 새로운 빗썸 워커를 생성합니다
func NewBithumbWorker(config *WorkerConfig, storage *MemoryStorage) *BithumbWorker {
	return &BithumbWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.bithumb.com/trade/place",
	}
}

// Start 워커를 시작합니다
func (bw *BithumbWorker) Start(ctx context.Context) {
	bw.mu.Lock()
	bw.running = true
	bw.mu.Unlock()
	
	bw.storage.AddLog("info", "빗썸 워커가 시작되었습니다.", bw.config.Exchange, bw.config.Symbol)

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
			bw.storage.AddLog("info", "빗썸 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		}
		bw.mu.RUnlock()

		select {
		case <-ctx.Done():
			bw.mu.Lock()
			bw.running = false
			bw.mu.Unlock()
			bw.storage.AddLog("info", "빗썸 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		case <-bw.stopCh:
			bw.mu.Lock()
			bw.running = false
			bw.mu.Unlock()
			bw.storage.AddLog("info", "빗썸 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
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
func (bw *BithumbWorker) Stop() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	
	if bw.running {
		bw.running = false
		close(bw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (bw *BithumbWorker) IsRunning() bool {
	bw.mu.RLock()
	defer bw.mu.RUnlock()
	return bw.running
}

// executeSellOrder 빗썸에서 매도 주문 실행
func (bw *BithumbWorker) executeSellOrder() {
	// 실행 상태 재확인
	bw.mu.RLock()
	if !bw.running {
		bw.mu.RUnlock()
		return
	}
	bw.mu.RUnlock()

	// 심볼 변환 (BTC/KRW -> BTC)
	bithumbSymbol := bw.convertToBithumbSymbol(bw.config.Symbol)

	params := url.Values{}
	params.Set("order_currency", bithumbSymbol)
	params.Set("payment_currency", "KRW")
	params.Set("units", fmt.Sprintf("%.8f", bw.config.SellAmount))
	params.Set("price", fmt.Sprintf("%.0f", bw.config.SellPrice))
	params.Set("type", "ask")

	// 서명 생성
	signature := bw.createBithumbSignature(params.Encode())

	req, err := http.NewRequest("POST", bw.url, strings.NewReader(params.Encode()))
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	req.Header.Set("Api-Key", bw.accessKey)
	req.Header.Set("Api-Sign", signature)
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
		orderID, ok := result["order_id"].(string)
		if ok && orderID != "" {
			bw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, bw.config.SellPrice, bw.config.SellAmount), bw.config.Exchange, bw.config.Symbol)
		} else {
			bw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				bw.config.SellPrice, bw.config.SellAmount), bw.config.Exchange, bw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["message"] != nil {
			errorMsg = fmt.Sprintf("%v", result["message"])
		}
		bw.storage.AddLog("error", fmt.Sprintf("빗썸 API 오류: %s", errorMsg), bw.config.Exchange, bw.config.Symbol)
	}
}

// convertToBithumbSymbol 심볼을 빗썸 형식으로 변환
func (bw *BithumbWorker) convertToBithumbSymbol(symbol string) string {
	// BTC/KRW -> BTC
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return symbol
}

// createBithumbSignature 빗썸 HMAC-SHA512 서명 생성
func (bw *BithumbWorker) createBithumbSignature(queryString string) string {
	h := hmac.New(sha512.New, []byte(bw.secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BithumbWorker) GetPlatformName() string {
	return "Bithumb"
}
