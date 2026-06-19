package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SymbolInfo 심볼 정보 구조체
type SymbolInfo struct {
	BaseCurrency    string
	QuoteCurrency   string
	PricePrecision  int
	AmountPrecision int
	MinOrderAmount  float64
	MinOrderValue   float64
	Symbol          string
}

// HuobiWorker 후오비 거래소 워커
type HuobiWorker struct {
	mu                 sync.RWMutex
	config             *WorkerConfig
	storage            *MemoryStorage
	running            bool
	stopCh             chan struct{}
	accessKey          string
	secretKey          string
	url                string
	lastSuccessOrderID string
	symbolInfoCache    map[string]*SymbolInfo
	symbolInfoCacheMu  sync.RWMutex
}

// NewHuobiWorker 새로운 후오비 워커를 생성합니다
func NewHuobiWorker(config *WorkerConfig, storage *MemoryStorage) *HuobiWorker {
	return &HuobiWorker{
		config:          config,
		storage:         storage,
		running:         false,
		stopCh:          make(chan struct{}),
		accessKey:       config.AccessKey,
		secretKey:       config.SecretKey,
		url:             "https://api.huobi.pro",
		symbolInfoCache: make(map[string]*SymbolInfo),
	}
}

// Start 워커를 시작합니다
func (hw *HuobiWorker) Start(ctx context.Context) {
	hw.mu.Lock()
	hw.running = true
	hw.mu.Unlock()

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(hw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		hw.mu.RLock()
		if !hw.running {
			hw.mu.RUnlock()
			hw.storage.AddLog("info", "후오비 워커가 중지되었습니다.", hw.config.Exchange, hw.config.Symbol)
			return
		}
		hw.mu.RUnlock()

		select {
		case <-ctx.Done():
			hw.mu.Lock()
			hw.running = false
			hw.mu.Unlock()
			hw.storage.AddLog("info", "후오비 워커가 중지되었습니다.", hw.config.Exchange, hw.config.Symbol)
			return
		case <-hw.stopCh:
			hw.mu.Lock()
			hw.running = false
			hw.mu.Unlock()
			hw.storage.AddLog("info", "후오비 워커가 중지되었습니다.", hw.config.Exchange, hw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			hw.mu.RLock()
			if hw.running {
				hw.mu.RUnlock()
				hw.executeSellOrder()
			} else {
				hw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (hw *HuobiWorker) Stop() {
	hw.mu.Lock()
	defer hw.mu.Unlock()

	if hw.running {
		hw.running = false
		close(hw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (hw *HuobiWorker) IsRunning() bool {
	hw.mu.RLock()
	defer hw.mu.RUnlock()
	return hw.running
}

// executeSellOrder 후오비에서 매도 주문 실행
func (hw *HuobiWorker) executeSellOrder() {
	// 실행 상태 재확인
	hw.mu.RLock()
	if !hw.running {
		hw.mu.RUnlock()
		return
	}
	hw.mu.RUnlock()

	// Spot 계정 ID 조회
	accountID, err := hw.getSpotAccountID()
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("계정 ID 조회 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	// 심볼 정보 조회
	symbolInfo, err := hw.getSymbolInfo(hw.config.Symbol)
	if err != nil {
		hw.storage.AddLog("warning", fmt.Sprintf("심볼 정보 조회 실패, 기본 정밀도로 진행합니다: %v", err), hw.config.Exchange, hw.config.Symbol)
		symbolInfo = nil
	}

	// 수량은 정수만, 가격은 원래 정밀도 사용
	pricePrecision := 8
	if symbolInfo != nil && symbolInfo.PricePrecision > 0 {
		pricePrecision = symbolInfo.PricePrecision
	}
	formattedAmount := truncateToPrecision(hw.config.SellAmount, 0)
	formattedPrice := truncateToPrecision(hw.config.SellPrice, pricePrecision)

	// 주문 요청 본문 생성
	orderBody := map[string]interface{}{
		"account-id":      accountID,
		"symbol":          strings.ToLower(strings.ReplaceAll(hw.config.Symbol, "/", "")),
		"type":            "sell-limit",
		"source":          "spot-api",
		"amount":          formattedAmount, // 수량 포맷팅 함수 사용
		"price":           formattedPrice,  // 가격 포맷팅 함수 사용
		"client-order-id": fmt.Sprintf("huobi-sell-%d", time.Now().UnixNano()),
	}

	// API 호출
	result, err := hw.callAPI("POST", "/v1/order/orders/place", orderBody)
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("%s, 매도수량: %s, 가격: %s, 심볼: %s 매도 주문에 실패했습니다. 거래소 응답 메세지: %v",
			hw.GetPlatformName(), formattedAmount, formattedPrice, hw.config.Symbol, err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	// 주문 성공 처리 (중복 방지)
	hw.mu.Lock()
	if orderID, ok := result["data"].(string); ok {
		if hw.lastSuccessOrderID != orderID {
			hw.lastSuccessOrderID = orderID
			hw.mu.Unlock()
			hw.storage.AddLog("success", fmt.Sprintf("%s, 매도수량: %s, 가격: %s, 심볼: %s 매도 주문에 성공했습니다.",
				hw.GetPlatformName(), formattedAmount, formattedPrice, hw.config.Symbol), hw.config.Exchange, hw.config.Symbol)
		} else {
			hw.mu.Unlock()
		}
	} else {
		if hw.lastSuccessOrderID != "success" {
			hw.lastSuccessOrderID = "success"
			hw.mu.Unlock()
			hw.storage.AddLog("success", fmt.Sprintf("%s, 매도수량: %s, 가격: %s, 심볼: %s 매도 주문에 성공했습니다.",
				hw.GetPlatformName(), formattedAmount, formattedPrice, hw.config.Symbol), hw.config.Exchange, hw.config.Symbol)
		} else {
			hw.mu.Unlock()
		}
	}
}

// createSignature API 호출 시 필요한 서명(Signature)을 생성하는 함수
func (hw *HuobiWorker) createSignature(method, path string, params map[string]string) string {
	// 1. 쿼리 파라미터 정렬 및 문자열 생성
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var queryParams string
	for _, k := range keys {
		// URL 인코딩은 Go의 url.Values를 사용하여 처리
		queryParams += k + "=" + url.QueryEscape(params[k]) + "&"
	}
	queryParams = strings.TrimSuffix(queryParams, "&")

	// 2. 서명 문자열 생성
	// GET\nHost\nPath\nQueryString
	stringToSign := fmt.Sprintf("%s\napi.huobi.pro\n%s\n%s", method, path, queryParams)

	// 3. HMAC-SHA256 서명 생성
	mac := hmac.New(sha256.New, []byte(hw.secretKey))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// callAPI HTX API를 호출하는 일반 함수
func (hw *HuobiWorker) callAPI(method, path string, body map[string]interface{}) (map[string]interface{}, error) {
	// 쿼리 파라미터 준비
	params := map[string]string{
		"AccessKeyId":      hw.accessKey,
		"SignatureMethod":  "HmacSHA256",
		"SignatureVersion": "2",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05"), // ISO 8601 형식
	}

	// 서명 생성
	signature := hw.createSignature(method, path, params)
	params["Signature"] = signature

	// 요청 URL 생성
	reqURL := fmt.Sprintf("%s%s", hw.url, path)

	// 최종 쿼리 문자열 생성
	var finalQuery string
	for k, v := range params {
		finalQuery += k + "=" + url.QueryEscape(v) + "&"
	}
	finalQuery = strings.TrimSuffix(finalQuery, "&")
	fullURL := reqURL + "?" + finalQuery

	// HTTP 요청 생성
	var req *http.Request
	var err error

	if method == "POST" && body != nil {
		jsonBody, _ := json.Marshal(body)
		req, err = http.NewRequest(method, fullURL, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, fullURL, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}

	// API 호출 및 응답 처리
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 호출 실패: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 본문 읽기 실패: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("응답 JSON 파싱 실패: %v", err)
	}

	if status, ok := result["status"]; ok && status != "ok" {
		return nil, fmt.Errorf("API 오류: %v (코드: %v)", result["err-msg"], result["err-code"])
	}

	return result, nil
}

// getSymbolInfo 심볼 정보 조회 (GET /v1/common/symbols)
func (hw *HuobiWorker) getSymbolInfo(symbol string) (*SymbolInfo, error) {
	// 캐시 확인
	hw.symbolInfoCacheMu.RLock()
	if cached, ok := hw.symbolInfoCache[symbol]; ok {
		hw.symbolInfoCacheMu.RUnlock()
		return cached, nil
	}
	hw.symbolInfoCacheMu.RUnlock()

	// API 호출 (공개 API이므로 인증 불필요)
	client := &http.Client{Timeout: 10 * time.Second}
	reqURL := fmt.Sprintf("%s/v1/common/symbols", hw.url)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 호출 실패: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 본문 읽기 실패: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("응답 JSON 파싱 실패: %v", err)
	}

	if status, ok := result["status"].(string); !ok || status != "ok" {
		return nil, fmt.Errorf("API 오류: %v", result)
	}

	// 심볼 변환 (예: BTC/USDT -> btcusdt)
	huobiSymbol := strings.ToLower(strings.ReplaceAll(symbol, "/", ""))

	// 심볼 정보 찾기
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			symbolData := item.(map[string]interface{})
			if symbolStr, ok := symbolData["symbol"].(string); ok && strings.ToLower(symbolStr) == huobiSymbol {
				info := &SymbolInfo{
					Symbol: symbolStr,
				}

				// base-currency
				if base, ok := symbolData["base-currency"].(string); ok {
					info.BaseCurrency = base
				}

				// quote-currency
				if quote, ok := symbolData["quote-currency"].(string); ok {
					info.QuoteCurrency = quote
				}

				// price-precision (가격 소수점 자리수)
				if pricePrecision, ok := symbolData["price-precision"].(float64); ok {
					info.PricePrecision = int(pricePrecision)
				} else {
					info.PricePrecision = 8
				}

				// amount-precision (수량 소수점 자리수)
				if amountPrecision, ok := symbolData["amount-precision"].(float64); ok {
					info.AmountPrecision = int(amountPrecision)
				} else {
					info.AmountPrecision = 8
				}

				// min-order-amt (최소 주문 수량)
				if minOrderAmt, ok := symbolData["min-order-amt"].(float64); ok {
					info.MinOrderAmount = minOrderAmt
				} else if minOrderAmt, ok := symbolData["min-order-amt"].(string); ok {
					parsed, _ := strconv.ParseFloat(minOrderAmt, 64)
					info.MinOrderAmount = parsed
				}

				// min-order-value (최소 주문 가치)
				if minOrderValue, ok := symbolData["min-order-value"].(float64); ok {
					info.MinOrderValue = minOrderValue
				} else if minOrderValue, ok := symbolData["min-order-value"].(string); ok {
					parsed, _ := strconv.ParseFloat(minOrderValue, 64)
					info.MinOrderValue = parsed
				}

				// 캐시에 저장
				hw.symbolInfoCacheMu.Lock()
				hw.symbolInfoCache[symbol] = info
				hw.symbolInfoCacheMu.Unlock()

				return info, nil
			}
		}
	}

	return nil, fmt.Errorf("심볼 정보를 찾을 수 없습니다: %s", symbol)
}

// getSpotAccountID Spot Account ID 조회
func (hw *HuobiWorker) getSpotAccountID() (string, error) {
	result, err := hw.callAPI("GET", "/v1/account/accounts", nil)
	if err != nil {
		return "", err
	}

	// 응답에서 Spot 계정 ID 찾기
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			account := item.(map[string]interface{})
			if accountType, ok := account["type"].(string); ok && accountType == "spot" {
				if id, ok := account["id"].(float64); ok {
					accountID := strconv.FormatFloat(id, 'f', 0, 64)
					return accountID, nil
				}
			}
		}
	}
	return "", fmt.Errorf("spot 계정 ID를 찾을 수 없습니다. API 권한을 확인하세요")
}

// formatPrice 가격을 후오비 API 형식에 맞게 포맷팅 (심볼 정보 기반, 반올림 방지, 뒤의 0 제거)
func (hw *HuobiWorker) formatPrice(price float64, symbolInfo *SymbolInfo) string {
	if symbolInfo == nil {
		symbolInfo = &SymbolInfo{PricePrecision: 8}
	}

	precision := symbolInfo.PricePrecision
	if precision > 10 {
		precision = 10
	}
	if precision < 0 {
		precision = 0
	}

	multiplier := 1.0
	for i := 0; i < precision; i++ {
		multiplier *= 10
	}

	// 반올림 방지를 위해 절삭 처리
	truncated := float64(int(price*multiplier)) / multiplier

	// 포맷팅
	formatted := fmt.Sprintf("%."+fmt.Sprintf("%d", precision)+"f", truncated)

	// 뒤의 불필요한 0 제거
	formatted = strings.TrimRight(formatted, "0")
	// 만약 소수점만 남았다면 제거 (예: "5." -> "5")
	formatted = strings.TrimRight(formatted, ".")

	// 소수점이 있지만 2자리 미만인 경우 최소 2자리까지 0 채우기 (후오비 API 요구사항)
	if strings.Contains(formatted, ".") {
		parts := strings.Split(formatted, ".")
		if len(parts) == 2 && len(parts[1]) < 2 {
			formatted = parts[0] + "." + parts[1] + strings.Repeat("0", 2-len(parts[1]))
		}
	} else {
		// 정수인 경우 소수점 2자리 추가
		formatted = formatted + ".00"
	}

	return formatted
}

// formatAmount 수량을 후오비 API 형식에 맞게 포맷팅 (심볼 정보 기반, 반올림 방지, 뒤의 0 제거)
func (hw *HuobiWorker) formatAmount(amount float64, symbolInfo *SymbolInfo) string {
	if symbolInfo == nil {
		symbolInfo = &SymbolInfo{AmountPrecision: 8}
	}

	precision := symbolInfo.AmountPrecision
	if precision > 10 {
		precision = 10
	}
	if precision < 0 {
		precision = 0
	}

	multiplier := 1.0
	for i := 0; i < precision; i++ {
		multiplier *= 10
	}

	// 반올림 방지를 위해 절삭 처리
	truncated := float64(int(amount*multiplier)) / multiplier

	// 포맷팅
	formatted := fmt.Sprintf("%."+fmt.Sprintf("%d", precision)+"f", truncated)

	// 뒤의 불필요한 0 제거
	formatted = strings.TrimRight(formatted, "0")
	// 만약 소수점만 남았다면 제거 (예: "5." -> "5")
	formatted = strings.TrimRight(formatted, ".")

	// 소수점이 있지만 2자리 미만인 경우 최소 2자리까지 0 채우기 (후오비 API 요구사항)
	if strings.Contains(formatted, ".") {
		parts := strings.Split(formatted, ".")
		if len(parts) == 2 && len(parts[1]) < 2 {
			formatted = parts[0] + "." + parts[1] + strings.Repeat("0", 2-len(parts[1]))
		}
	} else {
		// 정수인 경우 소수점 2자리 추가
		formatted = formatted + ".00"
	}

	return formatted
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (hw *HuobiWorker) GetPlatformName() string {
	return "Huobi"
}
