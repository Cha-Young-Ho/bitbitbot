package platform

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UpbitWorker 업비트 거래소 워커
type UpbitWorker struct {
	mu        sync.RWMutex
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewUpbitWorker 새로운 업비트 워커를 생성합니다
func NewUpbitWorker(config *WorkerConfig, storage *MemoryStorage) *UpbitWorker {
	return &UpbitWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.upbit.com/v1/orders",
	}
}

// Start 워커를 시작합니다
func (uw *UpbitWorker) Start(ctx context.Context) {
	uw.mu.Lock()
	uw.running = true
	uw.mu.Unlock()
	
	uw.storage.AddLog("info", "업비트 워커가 시작되었습니다.", uw.config.Exchange, uw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(uw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		uw.mu.RLock()
		if !uw.running {
			uw.mu.RUnlock()
			uw.storage.AddLog("info", "업비트 워커가 중지되었습니다.", uw.config.Exchange, uw.config.Symbol)
			return
		}
		uw.mu.RUnlock()

		select {
		case <-ctx.Done():
			uw.mu.Lock()
			uw.running = false
			uw.mu.Unlock()
			uw.storage.AddLog("info", "업비트 워커가 중지되었습니다.", uw.config.Exchange, uw.config.Symbol)
			return
		case <-uw.stopCh:
			uw.mu.Lock()
			uw.running = false
			uw.mu.Unlock()
			uw.storage.AddLog("info", "업비트 워커가 중지되었습니다.", uw.config.Exchange, uw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			uw.mu.RLock()
			if uw.running {
				uw.mu.RUnlock()
				uw.executeSellOrder()
			} else {
				uw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (uw *UpbitWorker) Stop() {
	uw.mu.Lock()
	defer uw.mu.Unlock()
	
	if uw.running {
		uw.running = false
		close(uw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (uw *UpbitWorker) IsRunning() bool {
	uw.mu.RLock()
	defer uw.mu.RUnlock()
	return uw.running
}

// executeSellOrder 업비트에서 매도 주문 실행
func (uw *UpbitWorker) executeSellOrder() {
	// 실행 상태 재확인
	uw.mu.RLock()
	if !uw.running {
		uw.mu.RUnlock()
		return
	}
	uw.mu.RUnlock()

	// 업비트 마켓 형식으로 변환 (BTC/KRW -> KRW-BTC)
	market := uw.toUpbitMarket(uw.config.Symbol)

	// 요청 파라미터
	params := url.Values{}
	params.Set("market", market)
	params.Set("side", "ask")
	params.Set("volume", fmt.Sprintf("%.8f", uw.config.SellAmount))
	params.Set("price", fmt.Sprintf("%.8f", uw.config.SellPrice))
	params.Set("ord_type", "limit")

	// JWT 토큰 생성
	jwtToken, err := uw.createUpbitJWTToken(params)
	if err != nil {
		uw.storage.AddLog("error", fmt.Sprintf("JWT 생성 실패: %v", err), uw.config.Exchange, uw.config.Symbol)
		return
	}

	// JSON 바디 구성
	body := map[string]string{
		"market":   params.Get("market"),
		"side":     params.Get("side"),
		"volume":   params.Get("volume"),
		"price":    params.Get("price"),
		"ord_type": params.Get("ord_type"),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		uw.storage.AddLog("error", fmt.Sprintf("바디 변환 실패: %v", err), uw.config.Exchange, uw.config.Symbol)
		return
	}

	req, err := http.NewRequest("POST", uw.url, bytes.NewReader(jsonBody))
	if err != nil {
		uw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), uw.config.Exchange, uw.config.Symbol)
		return
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		uw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), uw.config.Exchange, uw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		uw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), uw.config.Exchange, uw.config.Symbol)
		return
	}

	if resp.StatusCode == 201 {
		orderID, ok := result["uuid"].(string)
		if ok && orderID != "" {
			uw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, uw.config.SellPrice, uw.config.SellAmount), uw.config.Exchange, uw.config.Symbol)
		} else {
			uw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				uw.config.SellPrice, uw.config.SellAmount), uw.config.Exchange, uw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["error"] != nil {
			errorMap := result["error"].(map[string]interface{})
			if errorMap["message"] != nil {
				errorMsg = fmt.Sprintf("%v", errorMap["message"])
			}
		}
		uw.storage.AddLog("error", fmt.Sprintf("업비트 API 오류: %s", errorMsg), uw.config.Exchange, uw.config.Symbol)
	}
}

// toUpbitMarket 사용자 입력("BTC/KRW")을 업비트 마켓 포맷("KRW-BTC")으로 변환
func (uw *UpbitWorker) toUpbitMarket(symbol string) string {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return symbol // 포맷이 다르면 원본 반환
	}
	base := strings.TrimSpace(strings.ToUpper(parts[0]))
	quote := strings.TrimSpace(strings.ToUpper(parts[1]))
	return quote + "-" + base
}

// createUpbitJWTToken 업비트 JWT 토큰 생성
func (uw *UpbitWorker) createUpbitJWTToken(params url.Values) (string, error) {
	claims := jwt.MapClaims{
		"access_key": uw.accessKey,
		"nonce":      uuid.NewString(),
	}

	if len(params) > 0 {
		// 업비트 요구사항: 인코딩되지 않은 쿼리 문자열로 SHA512 해시 생성
		// 1) 키를 정렬
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// 2) key=value 형식으로 연결 (값은 인코딩하지 않음)
		var b strings.Builder
		first := true
		for _, k := range keys {
			for _, v := range params[k] {
				if !first {
					b.WriteByte('&')
				} else {
					first = false
				}
				b.WriteString(k)
				b.WriteByte('=')
				b.WriteString(v)
			}
		}
		rawQuery := b.String()
		sum := sha512.Sum512([]byte(rawQuery))
		claims["query_hash"] = hex.EncodeToString(sum[:])
		claims["query_hash_alg"] = "SHA512"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uw.secretKey))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (uw *UpbitWorker) GetPlatformName() string {
	return "Upbit"
}
