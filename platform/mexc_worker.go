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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MexcWorker MEXC 거래소 워커
type MexcWorker struct {
	mu        sync.RWMutex
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewMexcWorker 새로운 MEXC 워커를 생성합니다
func NewMexcWorker(config *WorkerConfig, storage *MemoryStorage) *MexcWorker {
	return &MexcWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.mexc.com/api/v3/order",
	}
}

// Start 워커를 시작합니다
func (mw *MexcWorker) Start(ctx context.Context) {
	mw.mu.Lock()
	mw.running = true
	mw.mu.Unlock()
	

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(mw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		mw.mu.RLock()
		if !mw.running {
			mw.mu.RUnlock()
			mw.storage.AddLog("info", "MEXC 워커가 중지되었습니다.", mw.config.Exchange, mw.config.Symbol)
			return
		}
		mw.mu.RUnlock()

		select {
		case <-ctx.Done():
			mw.mu.Lock()
			mw.running = false
			mw.mu.Unlock()
			mw.storage.AddLog("info", "MEXC 워커가 중지되었습니다.", mw.config.Exchange, mw.config.Symbol)
			return
		case <-mw.stopCh:
			mw.mu.Lock()
			mw.running = false
			mw.mu.Unlock()
			mw.storage.AddLog("info", "MEXC 워커가 중지되었습니다.", mw.config.Exchange, mw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			mw.mu.RLock()
			if mw.running {
				mw.mu.RUnlock()
				mw.executeSellOrder()
			} else {
				mw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (mw *MexcWorker) Stop() {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	
	if mw.running {
		mw.running = false
		close(mw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (mw *MexcWorker) IsRunning() bool {
	mw.mu.RLock()
	defer mw.mu.RUnlock()
	return mw.running
}

// executeSellOrder MEXC에서 매도 주문 실행
func (mw *MexcWorker) executeSellOrder() {
	// 실행 상태 재확인
	mw.mu.RLock()
	if !mw.running {
		mw.mu.RUnlock()
		return
	}
	mw.mu.RUnlock()

	timestamp := time.Now().UnixMilli()

	params := map[string]string{
		"symbol":   strings.ReplaceAll(mw.config.Symbol, "/", ""),
		"side":     "SELL",
		"type":     "LIMIT",
		"quantity": fmt.Sprintf("%.8f", mw.config.SellAmount),
		"price":    fmt.Sprintf("%.8f", mw.config.SellPrice),
		"timestamp":   strconv.FormatInt(timestamp, 10),
	}

	// MEXC API는 query string 방식으로 요청해야 함
	queryParams := url.Values{}
	queryParams.Set("symbol", params["symbol"])
	queryParams.Set("side", params["side"])
	queryParams.Set("type", params["type"])
	queryParams.Set("quantity", params["quantity"])
	queryParams.Set("price", params["price"])
	queryParams.Set("timestamp", params["timestamp"])

	// 서명 생성 (signature는 query string에 포함)
	signature := mw.createMexcSignature(params)
	queryParams.Set("signature", signature)

	// URL에 query string 추가
	requestURL := mw.url + "?" + queryParams.Encode()

	req, err := http.NewRequest("POST", requestURL, nil)
	if err != nil {
		mw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), mw.config.Exchange, mw.config.Symbol)
		return
	}

	req.Header.Set("X-MEXC-APIKEY", mw.accessKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		mw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), mw.config.Exchange, mw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		mw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), mw.config.Exchange, mw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["orderId"].(float64)
		if ok {
			mw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%.0f, 가격=%.2f, 수량=%.8f",
				orderID, mw.config.SellPrice, mw.config.SellAmount), mw.config.Exchange, mw.config.Symbol)
		} else {
			mw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				mw.config.SellPrice, mw.config.SellAmount), mw.config.Exchange, mw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		mw.storage.AddLog("error", fmt.Sprintf("MEXC API 오류: %s", errorMsg), mw.config.Exchange, mw.config.Symbol)
	}
}

// createMexcSignature MEXC HMAC-SHA256 서명 생성
func (mw *MexcWorker) createMexcSignature(params map[string]string) string {
	// MEXC API 스펙에 따라 키를 정렬하고 query string 생성
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	var queryString strings.Builder
	for i, key := range keys {
		if i > 0 {
			queryString.WriteString("&")
		}
		queryString.WriteString(key)
		queryString.WriteString("=")
		queryString.WriteString(params[key])
	}
	// HMAC-SHA256 서명 생성
	h := hmac.New(sha256.New, []byte(mw.secretKey))
	h.Write([]byte(queryString.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (mw *MexcWorker) GetPlatformName() string {
	return "MEXC"
}
