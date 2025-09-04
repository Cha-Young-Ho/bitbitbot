package platform

import (
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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
		url:       "https://api.bithumb.com/v1/orders",
	}
}

// Start 워커를 시작합니다
func (bw *BithumbWorker) Start(ctx context.Context) {
	// 설정 검증
	if err := bw.validateBithumbConfig(); err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("빗썸 설정 검증 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}
	
	bw.mu.Lock()
	bw.running = true
	bw.mu.Unlock()
	
	// 워커 시작 로그 제거

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

	// 심볼 변환 (BTC/KRW -> KRW-BTC)
	bithumbSymbol := bw.convertToBithumbSymbol(bw.config.Symbol)

	// 빗썸 API 2.0 파라미터 구성 (Upbit과 같은 방식)
	params := url.Values{}
	params.Set("market", bithumbSymbol)
	params.Set("side", "ask") // ask = 매도, bid = 매수
	params.Set("ord_type", "limit") // limit = 지정가 주문
	params.Set("price", fmt.Sprintf("%.0f", bw.config.SellPrice))
	params.Set("volume", fmt.Sprintf("%.8f", bw.config.SellAmount))

	// JWT 토큰 생성 (쿼리 파라미터를 query_hash로 사용)
	authToken, err := bw.createBithumbJWTToken(params)
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("JWT 토큰 생성 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	// JSON 바디 구성 (요청에 사용)
	requestBody := map[string]string{
		"market":   params.Get("market"),
		"side":     params.Get("side"),
		"ord_type": params.Get("ord_type"),
		"price":    params.Get("price"),
		"volume":   params.Get("volume"),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("JSON 바디 생성 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}
	
	// 디버깅 로그 제거

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", bw.url, strings.NewReader(string(jsonBody)))
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	// 빗썸 API 2.0 헤더 설정
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// HTTP 클라이언트 설정
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
		},
	}

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	// 응답 파싱
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	// 빗썸 API 2.0 응답 처리
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		// 주문 성공 (200 또는 201)
		orderID := ""
		if uuid, ok := result["uuid"].(string); ok && uuid != "" {
			orderID = uuid
		}
		
		// 전체 응답 로그 제거
		
		if orderID != "" {
			bw.storage.AddLog("success", fmt.Sprintf("빗썸 매도 주문 성공: 주문번호=%s, 가격=%.0f원, 수량=%.8f%s",
				orderID, bw.config.SellPrice, bw.config.SellAmount, bithumbSymbol), bw.config.Exchange, bw.config.Symbol)
		} else {
			bw.storage.AddLog("success", fmt.Sprintf("빗썸 매도 주문 성공: 가격=%.0f원, 수량=%.8f%s, 응답=%+v",
				bw.config.SellPrice, bw.config.SellAmount, bithumbSymbol, result), bw.config.Exchange, bw.config.Symbol)
		}
	} else {
		// HTTP 오류 - 상세한 오류 정보 추출
		errorMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		
		// 빗썸 API 2.0 오류 응답 구조 확인
		if errorInfo, ok := result["error"].(map[string]interface{}); ok {
			if name, ok := errorInfo["name"].(string); ok {
				errorMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, name)
			}
			if message, ok := errorInfo["message"].(string); ok {
				errorMsg = fmt.Sprintf("HTTP %d: %s - %s", resp.StatusCode, errorMsg, message)
			}
		} else if message, ok := result["message"].(string); ok {
			errorMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, message)
		}
		
		// 전체 응답 로그 제거
		bw.storage.AddLog("error", fmt.Sprintf("빗썸 요청 실패: %s", errorMsg), bw.config.Exchange, bw.config.Symbol)
	}
}

// convertToBithumbSymbol 심볼을 빗썸 형식으로 변환
func (bw *BithumbWorker) convertToBithumbSymbol(symbol string) string {
	// BTC/KRW -> KRW-BTC (빗썸 API 2.0 형식)
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return fmt.Sprintf("KRW-%s", parts[0])
	}
	return symbol
}

// createBithumbJWTToken 빗썸 API 2.0 JWT 토큰 생성 (Upbit과 같은 방식)
func (bw *BithumbWorker) createBithumbJWTToken(params url.Values) (string, error) {
	// UUID 생성
	nonce := uuid.New().String()
	
	// 현재 시간을 밀리초로 변환
	timestamp := time.Now().UnixMilli()
	
	// JWT 페이로드 구성
	payload := jwt.MapClaims{
		"access_key": bw.accessKey,
		"nonce":      nonce,
		"timestamp":  timestamp,
	}
	
	// 쿼리 파라미터가 있는 경우 query_hash 추가 (Upbit과 같은 방식)
	if len(params) > 0 {
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
		
		// 3) SHA512 해시 생성
		sum := sha512.Sum512([]byte(rawQuery))
		queryHash := hex.EncodeToString(sum[:])
		
		payload["query_hash"] = queryHash
		payload["query_hash_alg"] = "SHA512"
		
		// 디버깅 로그 제거
	}
	
	// JWT 토큰 생성
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenString, err := token.SignedString([]byte(bw.secretKey))
	if err != nil {
		return "", err
	}
	
	// 디버깅 로그 제거
	
	return "Bearer " + tokenString, nil
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BithumbWorker) GetPlatformName() string {
	return "Bithumb"
}

// validateBithumbConfig 빗썸 설정 검증
func (bw *BithumbWorker) validateBithumbConfig() error {
	if bw.accessKey == "" || bw.secretKey == "" {
		return fmt.Errorf("빗썸 API 키가 설정되지 않았습니다")
	}
	
	if bw.config.SellAmount <= 0 {
		return fmt.Errorf("매도 수량은 0보다 커야 합니다")
	}
	
	if bw.config.SellPrice <= 0 {
		return fmt.Errorf("매도 가격은 0보다 커야 합니다")
	}
	
	// 빗썸 최소 주문 금액 체크 (1000원 이상)
	if bw.config.SellPrice*bw.config.SellAmount < 1000 {
		return fmt.Errorf("빗썸 최소 주문 금액은 1,000원 이상이어야 합니다")
	}
	
	return nil
}

// getBithumbBalance 빗썸 잔고 조회 (선택적 구현)
func (bw *BithumbWorker) getBithumbBalance() (map[string]interface{}, error) {
	// 빗썸 API 2.0 잔고 조회 API 호출
	balanceURL := "https://api.bithumb.com/v1/accounts"
	
	// JWT 토큰 생성 (파라미터 없음)
	authToken, err := bw.createBithumbJWTToken(url.Values{})
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("GET", balanceURL, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}
