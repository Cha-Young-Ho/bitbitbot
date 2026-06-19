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
	"strconv"
	"strings"
	"sync"
	"time"
)

// KuCoinWorker 쿠코인 거래소 워커
type KuCoinWorker struct {
	mu                 sync.RWMutex
	config             *WorkerConfig
	storage            *MemoryStorage
	running            bool
	stopCh             chan struct{}
	accessKey          string
	secretKey          string
	url                string
	lastSuccessOrderID string
	symbolInfoCache    map[string]*KuCoinSymbolInfo
	symbolInfoMu       sync.RWMutex
}

// KuCoinSymbolInfo 심볼 정밀도/규칙 정보
type KuCoinSymbolInfo struct {
	Symbol          string
	BaseIncrement   string
	PriceIncrement  string
	AmountPrecision int
	PricePrecision  int
}

// NewKuCoinWorker 새로운 쿠코인 워커를 생성합니다
func NewKuCoinWorker(config *WorkerConfig, storage *MemoryStorage) *KuCoinWorker {
	return &KuCoinWorker{
		config:          config,
		storage:         storage,
		running:         false,
		stopCh:          make(chan struct{}),
		accessKey:       config.AccessKey,
		secretKey:       config.SecretKey,
		url:             "https://api.kucoin.com/api/v1/orders",
		symbolInfoCache: make(map[string]*KuCoinSymbolInfo),
	}
}

// Start 워커를 시작합니다
func (kcw *KuCoinWorker) Start(ctx context.Context) {
	kcw.mu.Lock()
	kcw.running = true
	kcw.mu.Unlock()

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(kcw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		kcw.mu.RLock()
		if !kcw.running {
			kcw.mu.RUnlock()
			kcw.storage.AddLog("info", "쿠코인 워커가 중지되었습니다.", kcw.config.Exchange, kcw.config.Symbol)
			return
		}
		kcw.mu.RUnlock()

		select {
		case <-ctx.Done():
			kcw.mu.Lock()
			kcw.running = false
			kcw.mu.Unlock()
			kcw.storage.AddLog("info", "쿠코인 워커가 중지되었습니다.", kcw.config.Exchange, kcw.config.Symbol)
			return
		case <-kcw.stopCh:
			kcw.mu.Lock()
			kcw.running = false
			kcw.mu.Unlock()
			kcw.storage.AddLog("info", "쿠코인 워커가 중지되었습니다.", kcw.config.Exchange, kcw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			kcw.mu.RLock()
			if kcw.running {
				kcw.mu.RUnlock()
				kcw.executeSellOrder()
			} else {
				kcw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (kcw *KuCoinWorker) Stop() {
	kcw.mu.Lock()
	defer kcw.mu.Unlock()

	if kcw.running {
		kcw.running = false
		close(kcw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (kcw *KuCoinWorker) IsRunning() bool {
	kcw.mu.RLock()
	defer kcw.mu.RUnlock()
	return kcw.running
}

// executeSellOrder 쿠코인에서 매도 주문 실행
func (kcw *KuCoinWorker) executeSellOrder() {
	// 실행 상태 재확인
	kcw.mu.RLock()
	if !kcw.running {
		kcw.mu.RUnlock()
		return
	}
	kcw.mu.RUnlock()

	timestamp := time.Now().UnixMilli()

	// 심볼 정밀도 정보 조회 (최초 1회만 API 호출, 이후 캐시 사용)
	symbolInfo, err := kcw.getSymbolInfo(kcw.config.Symbol)
	if err != nil {
		kcw.storage.AddLog("warning", fmt.Sprintf("쿠코인 심볼 정밀도 조회 실패, 기본 정밀도로 진행합니다: %v", err), kcw.config.Exchange, kcw.config.Symbol)
	}

	// 수량은 정수만, 가격은 원래 정밀도 사용
	pricePrecision := 8
	if symbolInfo != nil && symbolInfo.PricePrecision > 0 {
		pricePrecision = symbolInfo.PricePrecision
	}
	formattedSize := truncateToPrecision(kcw.config.SellAmount, 0)
	formattedPrice := truncateToPrecision(kcw.config.SellPrice, pricePrecision)

	requestBody := map[string]interface{}{
		"clientOid": fmt.Sprintf("sell_%d", timestamp),
		"symbol":    strings.ReplaceAll(kcw.config.Symbol, "/", "-"),
		"side":      "sell",
		"type":      "limit",
		"size":      formattedSize,
		"price":     formattedPrice,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("JSON 변환 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}

	// 서명 생성
	signature := kcw.createKuCoinSignature(string(jsonBody), timestamp)

	req, err := http.NewRequest("POST", kcw.url, bytes.NewReader(jsonBody))
	if err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}

	req.Header.Set("KC-API-KEY", kcw.accessKey)
	req.Header.Set("KC-API-SIGN", signature)
	req.Header.Set("KC-API-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("KC-API-PASSPHRASE", kcw.config.PasswordPhrase)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		kcw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), kcw.config.Exchange, kcw.config.Symbol)
		return
	}

	// 디버그 로그: 응답 상태와 내용 출력
	fmt.Printf("[DEBUG] 쿠코인 응답 상태: %d\n", resp.StatusCode)
	fmt.Printf("[DEBUG] 쿠코인 응답 내용: %+v\n", result)
	fmt.Printf("[DEBUG] 쿠코인 응답 내용: %+v\n", result["code"])
	fmt.Printf("[DEBUG] 쿠코인 응답 내용: %+v\n", result["msg"])

	// KuCoin API는 HTTP 200으로 응답하지만 code 필드로 실제 성공/실패를 알려줌
	code, ok := result["code"].(string)
	if !ok {
		codeFloat, ok := result["code"].(float64)
		if ok {
			code = fmt.Sprintf("%.0f", codeFloat)
		}
	}
	fmt.Printf("code: %s\n", code)
	fmt.Printf("비교: %+v\n", code == "200000")
	if resp.StatusCode == 200 && code == "200000" {
		kcw.mu.Lock()
		if kcw.lastSuccessOrderID == "" {
			kcw.lastSuccessOrderID = "success"
			kcw.mu.Unlock()
				kcw.storage.AddLog("success", fmt.Sprintf("%s, 매도수량: %s, 가격: %s, 심볼: %s 매도 주문에 성공했습니다.",
					kcw.GetPlatformName(), formattedSize, formattedPrice, kcw.config.Symbol), kcw.config.Exchange, kcw.config.Symbol)
		} else {
			kcw.mu.Unlock()
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		fmt.Printf("[DEBUG] 쿠코인 API 오류 (code: %s): %s\n", code, errorMsg)
		kcw.storage.AddLog("error", fmt.Sprintf("쿠코인 API 오류 (code: %s): %s (요청 수량: %s, 가격: %s)", code, errorMsg, formattedSize, formattedPrice), kcw.config.Exchange, kcw.config.Symbol)
	}
}

// createKuCoinSignature 쿠코인 HMAC-SHA256 서명 생성
func (kcw *KuCoinWorker) createKuCoinSignature(body string, timestamp int64) string {
	message := strconv.FormatInt(timestamp, 10) + "POST" + "/api/v1/orders" + body
	h := hmac.New(sha256.New, []byte(kcw.secretKey))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// getSymbolInfo 쿠코인 심볼 정밀도 정보를 조회하고 캐시합니다.
func (kcw *KuCoinWorker) getSymbolInfo(symbol string) (*KuCoinSymbolInfo, error) {
	// 캐시 확인
	kcw.symbolInfoMu.RLock()
	if info, ok := kcw.symbolInfoCache[symbol]; ok {
		kcw.symbolInfoMu.RUnlock()
		return info, nil
	}
	kcw.symbolInfoMu.RUnlock()

	type kuCoinSymbol struct {
		Symbol         string `json:"symbol"`
		BaseIncrement  string `json:"baseIncrement"`
		PriceIncrement string `json:"priceIncrement"`
	}

	type kuCoinResp struct {
		Code string         `json:"code"`
		Data []kuCoinSymbol `json:"data"`
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.kucoin.com/api/v1/symbols", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info kuCoinResp
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	if info.Code != "200000" {
		return nil, fmt.Errorf("쿠코인 심볼 정보 API 오류 코드: %s", info.Code)
	}

	exSymbol := strings.ReplaceAll(symbol, "/", "-")
	for _, s := range info.Data {
		if s.Symbol != exSymbol {
			continue
		}

	result := &KuCoinSymbolInfo{
			Symbol:         s.Symbol,
			BaseIncrement:  s.BaseIncrement,
			PriceIncrement: s.PriceIncrement,
		}

		// 증가 단위를 소수 자릿수로 변환
		result.AmountPrecision = countDecimalPlaces(result.BaseIncrement)
		result.PricePrecision = countDecimalPlaces(result.PriceIncrement)

		if result.AmountPrecision <= 0 {
			result.AmountPrecision = 8
		}
		if result.PricePrecision <= 0 {
			result.PricePrecision = 8
		}

		kcw.symbolInfoMu.Lock()
		kcw.symbolInfoCache[symbol] = result
		kcw.symbolInfoMu.Unlock()

		return result, nil
	}

	return nil, fmt.Errorf("쿠코인 심볼 정보를 찾을 수 없습니다: %s", symbol)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (kcw *KuCoinWorker) GetPlatformName() string {
	return "KuCoin"
}
