package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Worker 워커
type Worker struct {
	config  *WorkerConfig
	storage *MemoryStorage
	running bool
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// NewWorker 새로운 워커 생성
func NewWorker(config *WorkerConfig, storage *MemoryStorage) *Worker {
	return &Worker{
		config:  config,
		storage: storage,
		running: false,
		stopCh:  make(chan struct{}),
	}
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (w *Worker) GetPlatformName() string {
	return w.config.Exchange
}

// Start 워커 시작
func (w *Worker) Start(ctx context.Context) {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()
	
	w.storage.AddLog("info", "워커가 시작되었습니다.", w.config.Exchange, w.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(w.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	// 디버깅 로그 추가
	intervalLog := fmt.Sprintf("요청 간격 설정: %.2f초 (%d밀리초)", w.config.RequestInterval, interval.Milliseconds())
	w.storage.AddLog("info", intervalLog, w.config.Exchange, w.config.Symbol)
	log.Printf("[%s] %s - %s", w.config.Exchange, w.config.Symbol, intervalLog)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		w.mu.RLock()
		if !w.running {
			w.mu.RUnlock()
			w.storage.AddLog("info", "워커가 중지되었습니다.", w.config.Exchange, w.config.Symbol)
			return
		}
		w.mu.RUnlock()

		select {
		case <-ctx.Done():
			w.mu.Lock()
			w.running = false
			w.mu.Unlock()
			w.storage.AddLog("info", "워커가 중지되었습니다.", w.config.Exchange, w.config.Symbol)
			return
		case <-w.stopCh:
			w.mu.Lock()
			w.running = false
			w.mu.Unlock()
			w.storage.AddLog("info", "워커가 중지되었습니다.", w.config.Exchange, w.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			w.mu.RLock()
			if w.running {
				w.mu.RUnlock()
				w.processRequest()
			} else {
				w.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커 중지
func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.running {
		w.running = false
		close(w.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (w *Worker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// processRequest 요청 처리
func (w *Worker) processRequest() {
	// 실행 상태 재확인
	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()

	currentTime := time.Now()

	// 상태 업데이트
	statusStr := w.storage.GetWorkerStatus("main")
	if statusStr == "running" {
		// 상태가 running인 경우에만 업데이트
		w.storage.SetWorkerStatus("main", "running")
	}

	// 요청 시간 로그
	timeLog := fmt.Sprintf("매도 주문 요청 시작: %s", currentTime.Format("15:04:05.000"))
	w.storage.AddLog("info", timeLog, w.config.Exchange, w.config.Symbol)

	// 설정된 가격으로 매도 주문 실행
	orderResult := w.executeSellOrder(w.config.SellPrice)

	// 실행 상태 재확인 (주문 처리 후)
	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return
	}
	w.mu.RUnlock()

	if orderResult.Success {
		// 매도 주문 성공 로그
		successMessage := fmt.Sprintf("매도 주문 성공: 주문번호=%s, 가격=%.2f, 수량=%.4f, 총액=%.2f",
			orderResult.OrderID, orderResult.Price, orderResult.Amount, orderResult.TotalAmount)
		w.storage.AddLog("success", successMessage, w.config.Exchange, w.config.Symbol)
		log.Printf("[%s] %s - %s", w.config.Exchange, w.config.Symbol, successMessage)
	} else {
		// 매도 주문 실패 로그
		errorMessage := fmt.Sprintf("매도 주문 실패: %s", orderResult.ErrorMessage)
		w.storage.AddLog("error", errorMessage, w.config.Exchange, w.config.Symbol)
		log.Printf("[%s] %s - %s", w.config.Exchange, w.config.Symbol, errorMessage)
	}
}

// generateSimulatedPrice 시뮬레이션된 가격 생성
func (w *Worker) generateSimulatedPrice() float64 {
	// 기본 가격 + 랜덤 변동
	basePrice := 100.0
	variation := rand.Float64()*20 - 10 // -10 ~ +10 범위
	return basePrice + variation
}

// generateSimulatedVolume 시뮬레이션된 거래량 생성
func (w *Worker) generateSimulatedVolume() float64 {
	// 기본 거래량 + 랜덤 변동
	baseVolume := 1000.0
	variation := rand.Float64()*500 - 250 // -250 ~ +250 범위
	return baseVolume + variation
}

// OrderResult 주문 결과
type OrderResult struct {
	Success      bool    `json:"success"`
	OrderID      string  `json:"orderId"`
	Price        float64 `json:"price"`
	Amount       float64 `json:"amount"`
	TotalAmount  float64 `json:"totalAmount"`
	ErrorMessage string  `json:"errorMessage"`
}

// executeSellOrder 실제 매도 주문 실행
func (w *Worker) executeSellOrder(sellPrice float64) OrderResult {
	// 거래소별 실제 API 호출
	switch w.config.Exchange {
	case "Binance":
		return w.executeBinanceSellOrder(sellPrice)
	case "Bitget":
		return w.executeBitgetSellOrder(sellPrice)
	case "Bybit":
		return w.executeBybitSellOrder(sellPrice)
	case "KuCoin":
		return w.executeKuCoinSellOrder(sellPrice)
	case "Upbit":
		return w.executeUpbitSellOrder(sellPrice)
	case "Bithumb":
		return w.executeBithumbSellOrder(sellPrice)
	case "Coinbase":
		return w.executeCoinbaseSellOrder(sellPrice)
	case "Huobi":
		return w.executeHuobiSellOrder(sellPrice)
	case "Mexc":
		return w.executeMexcSellOrder(sellPrice)
	case "Coinone":
		return w.executeCoinoneSellOrder(sellPrice)
	case "Korbit":
		return w.executeKorbitSellOrder(sellPrice)
	case "Gate":
		// Gate.io는 별도 GateWorker에서 처리
		return w.executeSimulatedSellOrder(sellPrice)
	case "OKX":
		return w.executeOKXSellOrder(sellPrice)
	default:
		return w.executeSimulatedSellOrder(sellPrice)
	}
}

// executeBinanceSellOrder 바이낸스 매도 주문 실행
func (w *Worker) executeBinanceSellOrder(sellPrice float64) OrderResult {
	// 바이낸스 API 엔드포인트
	apiURL := "https://api.binance.com/api/v3/order"

	// 타임스탬프
	timestamp := time.Now().UnixMilli()

	// 쿼리 파라미터
	params := url.Values{}
	params.Set("symbol", strings.ReplaceAll(w.config.Symbol, "/", ""))
	params.Set("side", "SELL")
	params.Set("type", "LIMIT")
	params.Set("timeInForce", "GTC")
	params.Set("quantity", fmt.Sprintf("%.4f", w.config.SellAmount))
	params.Set("price", fmt.Sprintf("%.8f", sellPrice))
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))

	// 서명 생성
	queryString := params.Encode()
	signature := w.generateBinanceSignature(queryString, w.config.SecretKey)
	params.Set("signature", signature)

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return OrderResult{
			Success:      false,
			ErrorMessage: "HTTP 요청 생성 실패: " + err.Error(),
		}
	}

	// 헤더 설정
	req.Header.Set("X-MBX-APIKEY", w.config.AccessKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 요청 실행
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{
			Success:      false,
			ErrorMessage: "HTTP 요청 실패: " + err.Error(),
		}
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OrderResult{
			Success:      false,
			ErrorMessage: "응답 읽기 실패: " + err.Error(),
		}
	}

	// 응답 파싱
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return OrderResult{
			Success:      false,
			ErrorMessage: "응답 파싱 실패: " + err.Error(),
		}
	}

	// 성공 여부 확인
	if resp.StatusCode == 200 && result["orderId"] != nil {
		orderID := fmt.Sprintf("%.0f", result["orderId"])
		price, _ := strconv.ParseFloat(fmt.Sprintf("%v", result["price"]), 64)
		quantity, _ := strconv.ParseFloat(fmt.Sprintf("%v", result["origQty"]), 64)
		totalAmount := quantity * price

		return OrderResult{
			Success:      true,
			OrderID:      orderID,
			Price:        price,
			Amount:       quantity,
			TotalAmount:  totalAmount,
			ErrorMessage: "",
		}
	} else {
		// 에러 메시지 추출
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}

		return OrderResult{
			Success:      false,
			ErrorMessage: "바이낸스 API 오류: " + errorMsg,
		}
	}
}

// executeSimulatedSellOrder 시뮬레이션된 매도 주문 실행
func (w *Worker) executeSimulatedSellOrder(sellPrice float64) OrderResult {
	// 주문 성공 확률 95%
	if rand.Float64() < 0.95 {
		orderID := fmt.Sprintf("SELL_%d_%d", time.Now().Unix(), rand.Intn(1000))
		totalAmount := w.config.SellAmount * sellPrice

		return OrderResult{
			Success:      true,
			OrderID:      orderID,
			Price:        sellPrice,
			Amount:       w.config.SellAmount,
			TotalAmount:  totalAmount,
			ErrorMessage: "",
		}
	} else {
		// 5% 확률로 주문 실패
		return OrderResult{
			Success:      false,
			OrderID:      "",
			Price:        0,
			Amount:       0,
			TotalAmount:  0,
			ErrorMessage: "거래소 API 오류 또는 잔액 부족",
		}
	}
}

// generateBinanceSignature 바이낸스 서명 생성
func (w *Worker) generateBinanceSignature(queryString, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// executeBitgetSellOrder 비트겟 매도 주문 실행
func (w *Worker) executeBitgetSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.bitget.com/api/spot/v1/trade/placeOrder"

	timestamp := time.Now().UnixMilli()

	// 요청 바디
	requestBody := map[string]interface{}{
		"symbol":    strings.ReplaceAll(w.config.Symbol, "/", ""),
		"side":      "sell",
		"orderType": "limit",
		"force":     "normal",
		"quantity":  fmt.Sprintf("%.8f", w.config.SellAmount),
		"price":     fmt.Sprintf("%.8f", sellPrice),
		"timestamp": timestamp,
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 서명 생성 (비트겟 스펙에 맞춤)
	signString := fmt.Sprintf("%d%s%s", timestamp, "POST", "/api/spot/v1/trade/placeOrder") + string(jsonBody)
	signature := w.generateSignature(signString, w.config.SecretKey)

	// HTTP 요청
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("ACCESS-KEY", w.config.AccessKey)
	req.Header.Set("ACCESS-SIGN", signature)
	req.Header.Set("ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("ACCESS-PASSPHRASE", w.config.PasswordPhrase)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 && result["code"] == "00000" {
		data := result["data"].(map[string]interface{})
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", data["orderId"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		return OrderResult{Success: false, ErrorMessage: "비트겟 API 오류: " + errorMsg}
	}
}

// executeBybitSellOrder 바이비트 매도 주문 실행
func (w *Worker) executeBybitSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.bybit.com/v5/order/create"

	timestamp := time.Now().UnixMilli()

	requestBody := map[string]interface{}{
		"category":    "spot",
		"symbol":      strings.ReplaceAll(w.config.Symbol, "/", ""),
		"side":        "Sell",
		"orderType":   "Limit",
		"qty":         fmt.Sprintf("%.8f", w.config.SellAmount),
		"price":       fmt.Sprintf("%.8f", sellPrice),
		"timeInForce": "GTC",
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 서명 생성 (바이비트 스펙에 맞춤)
	signString := strconv.FormatInt(timestamp, 10) + w.config.AccessKey + "5000" + string(jsonBody)
	signature := w.generateSignature(signString, w.config.SecretKey)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("X-BAPI-API-KEY", w.config.AccessKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-SIGN-TYPE", "2")
	req.Header.Set("X-BAPI-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 && result["retCode"] == float64(0) {
		data := result["result"].(map[string]interface{})
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", data["orderId"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["retMsg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["retMsg"])
		}
		return OrderResult{Success: false, ErrorMessage: "바이비트 API 오류: " + errorMsg}
	}
}

// executeKuCoinSellOrder 쿠코인 매도 주문 실행
func (w *Worker) executeKuCoinSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.kucoin.com/api/v1/orders"

	timestamp := time.Now().UnixMilli()

	requestBody := map[string]interface{}{
		"clientOid": fmt.Sprintf("sell_%d", timestamp),
		"side":      "sell",
		"symbol":    w.config.Symbol,
		"type":      "limit",
		"size":      fmt.Sprintf("%.8f", w.config.SellAmount),
		"price":     fmt.Sprintf("%.8f", sellPrice),
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 서명 생성 (쿠코인 스펙에 맞춤)
	signString := strconv.FormatInt(timestamp, 10) + "POST" + "/api/v1/orders" + string(jsonBody)
	signature := w.generateSignature(signString, w.config.SecretKey)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("KC-API-KEY", w.config.AccessKey)
	req.Header.Set("KC-API-SIGN", signature)
	req.Header.Set("KC-API-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("KC-API-PASSPHRASE", w.config.PasswordPhrase)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 && result["code"] == "200000" {
		data := result["data"].(map[string]interface{})
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", data["orderId"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		return OrderResult{Success: false, ErrorMessage: "쿠코인 API 오류: " + errorMsg}
	}
}

// executeUpbitSellOrder 업비트 매도 주문 실행
func (w *Worker) executeUpbitSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.upbit.com/v1/orders"

	// 업비트 마켓 형식으로 변환 (BTC/KRW -> KRW-BTC)
	market := w.toUpbitMarket(w.config.Symbol)

	// 요청 파라미터
	params := url.Values{}
	params.Set("market", market)
	params.Set("side", "ask")
	params.Set("volume", fmt.Sprintf("%.8f", w.config.SellAmount))
	params.Set("price", fmt.Sprintf("%.8f", sellPrice))
	params.Set("ord_type", "limit")

	// JWT 토큰 생성
	jwtToken, err := w.createUpbitJWTToken(params)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "JWT 생성 실패: " + err.Error()}
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
		return OrderResult{Success: false, ErrorMessage: "바디 변환 실패: " + err.Error()}
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(bodyBytes, &result)

	if resp.StatusCode == 201 {
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", result["uuid"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["error"] != nil {
			errorMap := result["error"].(map[string]interface{})
			if errorMap["message"] != nil {
				errorMsg = fmt.Sprintf("%v", errorMap["message"])
			}
		}
		return OrderResult{Success: false, ErrorMessage: "업비트 API 오류: " + errorMsg}
	}
}

// executeBithumbSellOrder 빗썸 매도 주문 실행
func (w *Worker) executeBithumbSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.bithumb.com/trade/place"

	params := url.Values{}
	params.Set("order_currency", strings.Split(w.config.Symbol, "/")[0])
	params.Set("payment_currency", strings.Split(w.config.Symbol, "/")[1])
	params.Set("units", fmt.Sprintf("%.4f", w.config.SellAmount))
	params.Set("price", fmt.Sprintf("%.8f", sellPrice))
	params.Set("type", "ask")

	// 서명 생성
	signString := params.Encode()
	signature := w.generateSignature(signString, w.config.SecretKey)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("Api-Key", w.config.AccessKey)
	req.Header.Set("Api-Sign", signature)
	req.Header.Set("Api-Nonce", strconv.FormatInt(time.Now().UnixMilli(), 10))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 && result["status"] == "0000" {
		data := result["data"].(map[string]interface{})
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", data["order_id"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["message"] != nil {
			errorMsg = fmt.Sprintf("%v", result["message"])
		}
		return OrderResult{Success: false, ErrorMessage: "빗썸 API 오류: " + errorMsg}
	}
}

// executeCoinbaseSellOrder 코인베이스 매도 주문 실행
func (w *Worker) executeCoinbaseSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.coinbase.com/api/v3/brokerage/orders"

	timestamp := time.Now().Unix()

	requestBody := map[string]interface{}{
		"client_order_id": fmt.Sprintf("sell_%d", timestamp),
		"product_id":      w.config.Symbol,
		"side":            "SELL",
		"order_configuration": map[string]interface{}{
			"limit_limit_gtc": map[string]interface{}{
				"base_size":   fmt.Sprintf("%.4f", w.config.SellAmount),
				"limit_price": fmt.Sprintf("%.8f", sellPrice),
			},
		},
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 서명 생성
	signString := strconv.FormatInt(timestamp, 10) + "POST" + "/api/v3/brokerage/orders" + string(jsonBody)
	signature := w.generateSignature(signString, w.config.SecretKey)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("CB-ACCESS-KEY", w.config.AccessKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 {
		data := result["order"].(map[string]interface{})
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", data["order_id"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["error"] != nil {
			errorMsg = fmt.Sprintf("%v", result["error"])
		}
		return OrderResult{Success: false, ErrorMessage: "코인베이스 API 오류: " + errorMsg}
	}
}

// executeHuobiSellOrder 후오비 매도 주문 실행
func (w *Worker) executeHuobiSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.huobi.pro/v1/order/orders/place"

	requestBody := map[string]interface{}{
		"account-id": w.config.AccessKey,
		"symbol":     strings.ToLower(strings.ReplaceAll(w.config.Symbol, "/", "")),
		"type":       "sell-limit",
		"amount":     fmt.Sprintf("%.4f", w.config.SellAmount),
		"price":      fmt.Sprintf("%.8f", sellPrice),
		"source":     "api",
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 서명 생성
	signString := "POST\napi.huobi.pro\n/v1/order/orders/place\n" + string(jsonBody)
	signature := w.generateSignature(signString, w.config.SecretKey)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	// 쿼리 파라미터로 서명 전달
	q := req.URL.Query()
	q.Set("AccessKeyId", w.config.AccessKey)
	q.Set("SignatureMethod", "HmacSHA256")
	q.Set("SignatureVersion", "2")
	q.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))
	q.Set("Signature", signature)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 && result["status"] == "ok" {
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", result["data"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["err-msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["err-msg"])
		}
		return OrderResult{Success: false, ErrorMessage: "후오비 API 오류: " + errorMsg}
	}
}

// executeMexcSellOrder MEXC 매도 주문 실행
func (w *Worker) executeMexcSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://www.mexc.com/open/api/v2/order/place"

	timestamp := time.Now().Unix()

	params := url.Values{}
	params.Set("api_key", w.config.AccessKey)
	params.Set("symbol", strings.ReplaceAll(w.config.Symbol, "/", "_"))
	params.Set("price", fmt.Sprintf("%.8f", sellPrice))
	params.Set("number", fmt.Sprintf("%.4f", w.config.SellAmount))
	params.Set("trade_type", "ASK")
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))

	// 서명 생성
	signString := params.Encode()
	signature := w.generateSignature(signString, w.config.SecretKey)
	params.Set("sign", signature)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 && result["code"] == float64(200) {
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", result["data"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		return OrderResult{Success: false, ErrorMessage: "MEXC API 오류: " + errorMsg}
	}
}

// executeCoinoneSellOrder 코인원 매도 주문 실행
func (w *Worker) executeCoinoneSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.coinone.co.kr/v2.1/order"

	// 심볼 변환 (BTC/KRW -> BTC)
	coinoneSymbol := w.convertToCoinoneSymbol(w.config.Symbol)

	nonce := strconv.FormatInt(time.Now().UnixMilli(), 10)

	requestBody := map[string]interface{}{
		"access_token":    w.config.AccessKey,
		"nonce":           nonce,
		"side":            "sell", // 매도
		"quote_currency":  "KRW",
		"target_currency": coinoneSymbol,
		"type":            "limit",
		"price":           fmt.Sprintf("%.0f", sellPrice),
		"qty":             fmt.Sprintf("%.8f", w.config.SellAmount),
		"post_only":       "1",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "JSON 변환 실패: " + err.Error()}
	}

	// Base64 인코딩 (페이로드)
	payload := base64.StdEncoding.EncodeToString(jsonBody)

	// HMAC-SHA512 서명 생성
	signature := w.createCoinoneSignature(payload)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-COINONE-PAYLOAD", payload)
	req.Header.Set("X-COINONE-SIGNATURE", signature)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 {
		orderID, ok := result["order_id"].(string)
		if ok && orderID != "" {
			return OrderResult{
				Success:     true,
				OrderID:     orderID,
				Price:       sellPrice,
				Amount:      w.config.SellAmount,
				TotalAmount: w.config.SellAmount * sellPrice,
			}
		}
	}

	errorMsg := "알 수 없는 오류"
	if result["errorCode"] != nil {
		errorMsg = fmt.Sprintf("%v", result["errorCode"])
	}
	return OrderResult{Success: false, ErrorMessage: "코인원 API 오류: " + errorMsg}
}

// executeKorbitSellOrder 코빗 매도 주문 실행
func (w *Worker) executeKorbitSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://api.korbit.co.kr/v2/orders"

	// 심볼 변환 (BTC/KRW -> btc_krw)
	korbitSymbol := w.convertToKorbitSymbol(w.config.Symbol)

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	params := url.Values{}
	params.Set("symbol", korbitSymbol) // btc_krw
	params.Set("side", "sell")         // 매도
	params.Set("price", fmt.Sprintf("%.0f", sellPrice))
	params.Set("qty", fmt.Sprintf("%.8f", w.config.SellAmount))
	params.Set("orderType", "limit") // 지정가
	params.Set("timeInForce", "gtc") // Good Till Cancel
	params.Set("timestamp", timestamp)

	// HMAC-SHA256 서명 생성
	signature := w.createKorbitSignature(params.Encode())
	params.Set("signature", signature)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("X-KAPI-KEY", w.config.AccessKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 {
		return OrderResult{
			Success:     true,
			OrderID:     fmt.Sprintf("%v", result["id"]),
			Price:       sellPrice,
			Amount:      w.config.SellAmount,
			TotalAmount: w.config.SellAmount * sellPrice,
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["error"] != nil {
			errorMsg = fmt.Sprintf("%v", result["error"])
		}
		return OrderResult{Success: false, ErrorMessage: "코빗 API 오류: " + errorMsg}
	}
}

// generateSignature 일반 서명 생성
func (w *Worker) generateSignature(message, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// toUpbitMarket 사용자 입력("BTC/KRW")을 업비트 마켓 포맷("KRW-BTC")으로 변환
func (w *Worker) toUpbitMarket(symbol string) string {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return symbol // 포맷이 다르면 원본 반환
	}
	base := strings.TrimSpace(strings.ToUpper(parts[0]))
	quote := strings.TrimSpace(strings.ToUpper(parts[1]))
	return quote + "-" + base
}

// createUpbitJWTToken 업비트 JWT 토큰 생성
func (w *Worker) createUpbitJWTToken(params url.Values) (string, error) {
	claims := jwt.MapClaims{
		"access_key": w.config.AccessKey,
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
		// 2) key=value 형식으로 연결 (값은 인코딩하지 않음), 여러 값이면 key=value1&key=value2 순서로
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
	return token.SignedString([]byte(w.config.SecretKey))
}

// createKorbitJWTToken 코빗 JWT 토큰 생성
func (w *Worker) createKorbitJWTToken() string {
	// 코빗 JWT 토큰 생성 로직 (실제로는 더 복잡함)
	// 실제 구현에서는 코빗의 인증 방식을 따라야 함
	return "korbit_jwt_token_" + w.config.AccessKey
}

// convertToCoinoneSymbol 심볼을 Coinone 형식으로 변환
func (w *Worker) convertToCoinoneSymbol(symbol string) string {
	// BTC/KRW -> BTC
	// USDT/KRW -> USDT
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return symbol
}

// createCoinoneSignature Coinone HMAC-SHA512 서명 생성
func (w *Worker) createCoinoneSignature(payload string) string {
	// HMAC-SHA512 서명 생성
	h := hmac.New(sha512.New, []byte(w.config.SecretKey))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// convertToKorbitSymbol 심볼을 Korbit 형식으로 변환
func (w *Worker) convertToKorbitSymbol(symbol string) string {
	// BTC/KRW -> btc_krw
	// USDT/KRW -> usdt_krw
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return strings.ToLower(parts[0]) + "_" + strings.ToLower(parts[1])
	}
	return strings.ToLower(symbol)
}

// createKorbitSignature Korbit HMAC-SHA256 서명 생성
func (w *Worker) createKorbitSignature(queryString string) string {
	// HMAC-SHA256 서명 생성
	h := hmac.New(sha256.New, []byte(w.config.SecretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// Gate.io는 별도 GateWorker에서 처리하므로 이 함수는 제거됨

// executeOKXSellOrder OKX 매도 주문 실행
func (w *Worker) executeOKXSellOrder(sellPrice float64) OrderResult {
	apiURL := "https://www.okx.com/api/v5/trade/order"

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	requestBody := map[string]interface{}{
		"instId":  w.config.Symbol,
		"tdMode":  "cash",
		"side":    "sell",
		"ordType": "limit",
		"sz":      fmt.Sprintf("%.8f", w.config.SellAmount),
		"px":      fmt.Sprintf("%.8f", sellPrice),
	}

	jsonBody, _ := json.Marshal(requestBody)

	// 서명 생성 (OKX 스펙에 맞춤)
	signString := timestamp + "POST" + "/api/v5/trade/order" + string(jsonBody)
	signature := w.generateSignature(signString, w.config.SecretKey)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	req.Header.Set("OK-ACCESS-KEY", w.config.AccessKey)
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", w.config.PasswordPhrase)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode == 200 && result["code"] == "0" {
		data := result["data"].([]interface{})
		if len(data) > 0 {
			orderData := data[0].(map[string]interface{})
			return OrderResult{
				Success:     true,
				OrderID:     fmt.Sprintf("%v", orderData["ordId"]),
				Price:       sellPrice,
				Amount:      w.config.SellAmount,
				TotalAmount: w.config.SellAmount * sellPrice,
			}
		}
	}

	errorMsg := "알 수 없는 오류"
	if result["msg"] != nil {
		errorMsg = fmt.Sprintf("%v", result["msg"])
	}
	return OrderResult{Success: false, ErrorMessage: "OKX API 오류: " + errorMsg}
}
