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
	mu                 sync.RWMutex
	config             *WorkerConfig
	storage            *MemoryStorage
	running            bool
	stopCh             chan struct{}
	accessKey          string
	secretKey          string
	url                string
	lastSuccessOrderID string
	marketInfoCache    map[string]*UpbitMarketInfo
	marketInfoMu       sync.RWMutex
}

// UpbitMarketInfo 업비트 마켓 정밀도 정보
type UpbitMarketInfo struct {
	Market         string  `json:"market"`
	MinPriceUnit   float64 `json:"min_price_unit,omitempty"` // (주의) 실제 필드는 시기별로 다를 수 있음
	PricePrecision int     // 우리가 계산한 가격 소수 자릿수
}

// NewUpbitWorker 새로운 업비트 워커를 생성합니다
func NewUpbitWorker(config *WorkerConfig, storage *MemoryStorage) *UpbitWorker {
	return &UpbitWorker{
		config:          config,
		storage:         storage,
		running:         false,
		stopCh:          make(chan struct{}),
		accessKey:       config.AccessKey,
		secretKey:       config.SecretKey,
		url:             "https://api.upbit.com/v1/orders",
		marketInfoCache: make(map[string]*UpbitMarketInfo),
	}
}

// Start 워커를 시작합니다
func (uw *UpbitWorker) Start(ctx context.Context) {
	uw.mu.Lock()
	uw.running = true
	uw.mu.Unlock()

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

	// 수량은 정수만, 가격은 기본 8자리 사용 (업비트는 호가 단위가 다양하므로)
	formattedVolume := truncateToPrecision(uw.config.SellAmount, 0)
	formattedPrice := fmt.Sprintf("%.8f", uw.config.SellPrice)

	params := url.Values{}
	params.Set("market", market)
	params.Set("side", "ask")
	params.Set("volume", formattedVolume)
	params.Set("price", formattedPrice)
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
			uw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%s, 수량=%s",
				orderID, formattedPrice, formattedVolume), uw.config.Exchange, uw.config.Symbol)
		} else {
			uw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%s, 수량=%s",
				formattedPrice, formattedVolume), uw.config.Exchange, uw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["error"] != nil {
			errorMap := result["error"].(map[string]interface{})
			if errorMap["message"] != nil {
				errorMsg = fmt.Sprintf("%v", errorMap["message"])
			}
		}
		uw.storage.AddLog("error", fmt.Sprintf("업비트 API 오류: %s (요청 가격=%s, 수량=%s)", errorMsg, formattedPrice, formattedVolume), uw.config.Exchange, uw.config.Symbol)
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

// getMarketInfo 업비트 마켓 정밀도 정보를 조회하고 캐시합니다.
// 참고: 업비트 공식 시세/마켓 정보 API는 고정된 price unit을 직접 주지 않기 때문에,
// 여기서는 마켓 캔들/호가를 참고해 대략적인 소수 자릿수를 계산하는 방식 대신,
// 향후 확장용으로 캐시 구조만 두고 현재는 가격을 입력값 기준으로만 truncate 합니다.
func (uw *UpbitWorker) getMarketInfo(market string) (*UpbitMarketInfo, error) {
	// 캐시 확인
	uw.marketInfoMu.RLock()
	if info, ok := uw.marketInfoCache[market]; ok {
		uw.marketInfoMu.RUnlock()
		return info, nil
	}
	uw.marketInfoMu.RUnlock()

	// 간단히: 기본 구조만 채우고, 향후 필요 시 /v1/market/all, /v1/ticker 등으로 확장 가능
	info := &UpbitMarketInfo{
		Market:         market,
		PricePrecision: 0, // 0이면 원본 값 사용
	}

	uw.marketInfoMu.Lock()
	uw.marketInfoCache[market] = info
	uw.marketInfoMu.Unlock()

	return info, nil
}
