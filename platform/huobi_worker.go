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
		url:       "https://api.huobi.pro/v1/order/orders/place",
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

	// 먼저 잔고 확인
	balance, err := hw.getBalance()
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("잔고 조회 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	// 매도할 수량이 충분한지 확인
	if balance < hw.config.SellAmount {
		hw.storage.AddLog("warning", fmt.Sprintf("잔고 부족: 보유량=%.8f, 매도량=%.8f", balance, hw.config.SellAmount), hw.config.Exchange, hw.config.Symbol)
		return
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")

	// 실제 계정 ID 조회
	accountID, err := hw.getAccountID()
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("계정 ID 조회 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	requestBody := map[string]interface{}{
		"account-id": accountID, // 실제 계정 ID 사용
		"symbol":     strings.ToLower(strings.ReplaceAll(hw.config.Symbol, "/", "")),
		"type":       "sell-limit",
		"amount":     fmt.Sprintf("%.8f", hw.config.SellAmount),
		"price":      fmt.Sprintf("%.8f", hw.config.SellPrice),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("JSON 변환 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	// 서명 생성
	signature := hw.createHuobiSignature(string(jsonBody), timestamp)

	req, err := http.NewRequest("POST", hw.url, bytes.NewReader(jsonBody))
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("AccessKeyId", hw.accessKey)
	req.Header.Set("SignatureMethod", "HmacSHA256")
	req.Header.Set("SignatureVersion", "2")
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("Signature", signature)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		hw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), hw.config.Exchange, hw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["data"].(string)
		if ok && orderID != "" {
			hw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.8f",
				orderID, hw.config.SellPrice, hw.config.SellAmount), hw.config.Exchange, hw.config.Symbol)
		} else {
			hw.storage.AddLog("success", fmt.Sprintf("매도 주문 성공: 가격=%.2f, 수량=%.8f",
				hw.config.SellPrice, hw.config.SellAmount), hw.config.Exchange, hw.config.Symbol)
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["err-msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["err-msg"])
		}
		hw.storage.AddLog("error", fmt.Sprintf("후오비 API 오류: %s", errorMsg), hw.config.Exchange, hw.config.Symbol)
	}
}

// getBalance 후오비에서 잔고 조회
func (hw *HuobiWorker) getBalance() (float64, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	
	// 심볼에서 기본 통화 추출 (예: BTC/USDT -> BTC)
	parts := strings.Split(hw.config.Symbol, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("잘못된 심볼 형식: %s", hw.config.Symbol)
	}
	baseCurrency := strings.ToLower(parts[0]) // btc
	
	// 잔고 조회 요청
	balanceURL := "https://api.huobi.pro/v1/account/accounts/balance"
	req, err := http.NewRequest("GET", balanceURL, nil)
	if err != nil {
		return 0, err
	}
	
	// 서명 생성 (GET 요청용)
	message := "GET\napi.huobi.pro\n/v1/account/accounts/balance\n"
	h := hmac.New(sha256.New, []byte(hw.secretKey))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("AccessKeyId", hw.accessKey)
	req.Header.Set("SignatureMethod", "HmacSHA256")
	req.Header.Set("SignatureVersion", "2")
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("Signature", signature)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("잔고 조회 실패: %v", result)
	}
	
	// 잔고 데이터 파싱
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if account, ok := data[0].(map[string]interface{}); ok {
			if list, ok := account["list"].([]interface{}); ok {
				for _, item := range list {
					if balance, ok := item.(map[string]interface{}); ok {
						if currency, ok := balance["currency"].(string); ok && currency == baseCurrency {
							if type_, ok := balance["type"].(string); ok && type_ == "trade" {
								if balanceStr, ok := balance["balance"].(string); ok {
									var balance float64
									if _, err := fmt.Sscanf(balanceStr, "%f", &balance); err == nil {
										return balance, nil
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	return 0, fmt.Errorf("잔고 정보를 찾을 수 없습니다")
}

// getAccountID 후오비에서 계정 ID 조회
func (hw *HuobiWorker) getAccountID() (string, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05")
	
	// 계정 조회 요청
	accountURL := "https://api.huobi.pro/v1/account/accounts"
	req, err := http.NewRequest("GET", accountURL, nil)
	if err != nil {
		return "", err
	}
	
	// 서명 생성 (GET 요청용)
	message := "GET\napi.huobi.pro\n/v1/account/accounts\n"
	h := hmac.New(sha256.New, []byte(hw.secretKey))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("AccessKeyId", hw.accessKey)
	req.Header.Set("SignatureMethod", "HmacSHA256")
	req.Header.Set("SignatureVersion", "2")
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("Signature", signature)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("계정 조회 실패: %v", result)
	}
	
	// 계정 데이터 파싱 (첫 번째 계정의 ID 반환)
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if account, ok := data[0].(map[string]interface{}); ok {
			if id, ok := account["id"].(float64); ok {
				return fmt.Sprintf("%.0f", id), nil
			}
		}
	}
	
	return "", fmt.Errorf("계정 ID를 찾을 수 없습니다")
}

// createHuobiSignature 후오비 HMAC-SHA256 서명 생성
func (hw *HuobiWorker) createHuobiSignature(body string, timestamp string) string {
	message := "POST\napi.huobi.pro\n/v1/order/orders/place\n" + body
	h := hmac.New(sha256.New, []byte(hw.secretKey))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (hw *HuobiWorker) GetPlatformName() string {
	return "Huobi"
}
