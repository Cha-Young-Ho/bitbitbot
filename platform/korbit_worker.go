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

// KorbitWorker 코빗 거래소 워커
type KorbitWorker struct {
	mu        sync.RWMutex
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewKorbitWorker 새로운 코빗 워커를 생성합니다
func NewKorbitWorker(config *WorkerConfig, storage *MemoryStorage) *KorbitWorker {
	return &KorbitWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.korbit.co.kr/v2/orders",
	}
}

// Start 워커를 시작합니다
func (kw *KorbitWorker) Start(ctx context.Context) {
	kw.mu.Lock()
	kw.running = true
	kw.mu.Unlock()
	
	kw.storage.AddLog("info", "코빗 워커가 시작되었습니다.", kw.config.Exchange, kw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(kw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		kw.mu.RLock()
		if !kw.running {
			kw.mu.RUnlock()
			kw.storage.AddLog("info", "코빗 워커가 중지되었습니다.", kw.config.Exchange, kw.config.Symbol)
			return
		}
		kw.mu.RUnlock()

		select {
		case <-ctx.Done():
			kw.mu.Lock()
			kw.running = false
			kw.mu.Unlock()
			kw.storage.AddLog("info", "코빗 워커가 중지되었습니다.", kw.config.Exchange, kw.config.Symbol)
			return
		case <-kw.stopCh:
			kw.mu.Lock()
			kw.running = false
			kw.mu.Unlock()
			kw.storage.AddLog("info", "코빗 워커가 중지되었습니다.", kw.config.Exchange, kw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			kw.mu.RLock()
			if kw.running {
				kw.mu.RUnlock()
				kw.executeSellOrder()
			} else {
				kw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (kw *KorbitWorker) Stop() {
	kw.mu.Lock()
	defer kw.mu.Unlock()
	
	if kw.running {
		kw.running = false
		close(kw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (kw *KorbitWorker) IsRunning() bool {
	kw.mu.RLock()
	defer kw.mu.RUnlock()
	return kw.running
}

// executeSellOrder 코빗에서 매도 주문 실행
func (kw *KorbitWorker) executeSellOrder() {
	// 실행 상태 재확인
	kw.mu.RLock()
	if !kw.running {
		kw.mu.RUnlock()
		return
	}
	kw.mu.RUnlock()

	// 심볼 변환 (BTC/KRW -> btc_krw)
	korbitSymbol := kw.convertToKorbitSymbol(kw.config.Symbol)

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	params := url.Values{}
	params.Set("symbol", korbitSymbol) // btc_krw
	params.Set("side", "sell")         // 매도
	params.Set("price", fmt.Sprintf("%.0f", kw.config.SellPrice))
	params.Set("qty", fmt.Sprintf("%.8f", kw.config.SellAmount))
	params.Set("orderType", "limit") // 지정가
	params.Set("timeInForce", "gtc") // Good Till Cancel
	params.Set("timestamp", timestamp)

	// HMAC-SHA256 서명 생성
	signature := kw.createKorbitSignature(params.Encode())
	params.Set("signature", signature)

	req, err := http.NewRequest("POST", kw.url, strings.NewReader(params.Encode()))
	if err != nil {
		kw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), kw.config.Exchange, kw.config.Symbol)
		return
	}

	req.Header.Set("X-KAPI-KEY", kw.accessKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		kw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), kw.config.Exchange, kw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		kw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), kw.config.Exchange, kw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["id"].(string)
		if ok && orderID != "" {
			kw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, kw.config.SellPrice, kw.config.SellAmount), kw.config.Exchange, kw.config.Symbol)
		} else {
			kw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				kw.config.SellPrice, kw.config.SellAmount), kw.config.Exchange, kw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["error"] != nil {
			errorMsg = fmt.Sprintf("%v", result["error"])
		}
		kw.storage.AddLog("error", fmt.Sprintf("코빗 API 오류: %s", errorMsg), kw.config.Exchange, kw.config.Symbol)
	}
}

// convertToKorbitSymbol 심볼을 코빗 형식으로 변환
func (kw *KorbitWorker) convertToKorbitSymbol(symbol string) string {
	// BTC/KRW -> btc_krw
	// USDT/KRW -> usdt_krw
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return strings.ToLower(parts[0]) + "_" + strings.ToLower(parts[1])
	}
	return strings.ToLower(symbol)
}

// createKorbitSignature 코빗 HMAC-SHA256 서명 생성
func (kw *KorbitWorker) createKorbitSignature(queryString string) string {
	h := hmac.New(sha256.New, []byte(kw.secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (kw *KorbitWorker) GetPlatformName() string {
	return "Korbit"
}
