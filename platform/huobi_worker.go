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

// HuobiWorker 후오비 거래소 워커
type HuobiWorker struct {
	mu        sync.RWMutex
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
}

// NewHuobiWorker 새로운 후오비 워커를 생성합니다
func NewHuobiWorker(config *WorkerConfig, storage *MemoryStorage) *HuobiWorker {
	return &HuobiWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.huobi.pro",
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

	hw.storage.AddLog("debug", "매도 주문 시작", hw.config.Exchange, hw.config.Symbol)

	// Spot 계정 ID 조회
	accountID, err := hw.getSpotAccountID()
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("계정 ID 조회 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	// 가격과 수량 포맷팅
	formattedPrice := hw.formatPrice(hw.config.SellPrice)
	formattedAmount := hw.formatAmount(hw.config.SellAmount)
	hw.storage.AddLog("debug", fmt.Sprintf("원본 가격: %.8f, 포맷팅된 가격: %s", hw.config.SellPrice, formattedPrice), hw.config.Exchange, hw.config.Symbol)
	hw.storage.AddLog("debug", fmt.Sprintf("원본 수량: %.8f, 포맷팅된 수량: %s", hw.config.SellAmount, formattedAmount), hw.config.Exchange, hw.config.Symbol)

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
		hw.storage.AddLog("error", fmt.Sprintf("매도 주문 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	// 주문 성공 처리
	if orderID, ok := result["data"].(string); ok {
		hw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
			orderID, hw.config.SellPrice, hw.config.SellAmount), hw.config.Exchange, hw.config.Symbol)
	} else {
		hw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
			hw.config.SellPrice, hw.config.SellAmount), hw.config.Exchange, hw.config.Symbol)
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

	// 디버그 로그
	hw.storage.AddLog("debug", fmt.Sprintf("API 요청: %s %s", method, fullURL), hw.config.Exchange, hw.config.Symbol)
	if body != nil {
		bodyJSON, _ := json.Marshal(body)
		hw.storage.AddLog("debug", "요청 본문: "+string(bodyJSON), hw.config.Exchange, hw.config.Symbol)
	}

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

	// 디버그 로그 - 응답
	hw.storage.AddLog("debug", fmt.Sprintf("API 응답 상태: %d", resp.StatusCode), hw.config.Exchange, hw.config.Symbol)
	hw.storage.AddLog("debug", "API 응답 본문: "+string(respBody), hw.config.Exchange, hw.config.Symbol)

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("응답 JSON 파싱 실패: %v", err)
	}

	if status, ok := result["status"]; ok && status != "ok" {
		return nil, fmt.Errorf("API 오류: %v (코드: %v)", result["err-msg"], result["err-code"])
	}

	return result, nil
}

// getSpotAccountID Spot Account ID 조회
func (hw *HuobiWorker) getSpotAccountID() (string, error) {
	hw.storage.AddLog("debug", "Spot Account ID 조회 중...", hw.config.Exchange, hw.config.Symbol)
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
					hw.storage.AddLog("debug", fmt.Sprintf("Spot Account ID 찾음: %s", accountID), hw.config.Exchange, hw.config.Symbol)
					return accountID, nil
				}
			}
		}
	}
	return "", fmt.Errorf("spot 계정 ID를 찾을 수 없습니다. API 권한을 확인하세요")
}

// formatPrice 가격을 후오비 API 형식에 맞게 포맷팅
func (hw *HuobiWorker) formatPrice(price float64) string {
	// 6자리로 자르기 (8자리 입력 시 마지막 2자리 제거)
	formatted := fmt.Sprintf("%.6f", price)

	// 소수점 이하 부분만 추출
	parts := strings.Split(formatted, ".")
	if len(parts) != 2 {
		return formatted
	}

	integerPart := parts[0]
	decimalPart := parts[1]

	// 소수점 이하가 2자리보다 적으면 2자리까지 채우기
	if len(decimalPart) < 2 {
		decimalPart = decimalPart + strings.Repeat("0", 2-len(decimalPart))
	}

	// 최소 소수점 둘째자리까지는 항상 표시
	result := integerPart + "." + decimalPart

	return result
}

// formatAmount 수량을 후오비 API 형식에 맞게 포맷팅 (소수점 2자리까지만)
func (hw *HuobiWorker) formatAmount(amount float64) string {
	// 소수점 2자리로 자르기
	formatted := fmt.Sprintf("%.2f", amount)
	return formatted
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (hw *HuobiWorker) GetPlatformName() string {
	return "Huobi"
}
