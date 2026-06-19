package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GateWorker Gate.io 거래소 워커 (APIv4 직접 구현)
type GateWorker struct {
	mu                 sync.RWMutex
	config             *WorkerConfig
	storage            *MemoryStorage
	running            bool
	stopCh             chan struct{}
	lastSuccessOrderID string
	symbolInfoCache    map[string]*GateSymbolInfo
	symbolInfoMu       sync.RWMutex
}

// GateSymbolInfo Gate.io 심볼 정밀도/규칙 정보
type GateSymbolInfo struct {
	CurrencyPair    string `json:"currency_pair"`
	AmountPrecision int    `json:"amount_precision,omitempty"` // 수량 소수 자릿수
	Precision       int    `json:"precision,omitempty"`        // 가격 소수 자릿수

	MinBaseAmount  string `json:"min_base_amount,omitempty"`
	MinQuoteAmount string `json:"min_quote_amount,omitempty"`

	PricePrecision  int // 우리가 계산해 쓰는 가격 정밀도
	AmountPrecision2 int // 우리가 계산해 쓰는 수량 정밀도
}

// NewGateWorker 새로운 Gate.io 워커를 생성합니다
func NewGateWorker(config *WorkerConfig, storage *MemoryStorage) *GateWorker {
	return &GateWorker{
		config:          config,
		storage:         storage,
		running:         false,
		stopCh:          make(chan struct{}),
		symbolInfoCache: make(map[string]*GateSymbolInfo),
	}
}

// Start 워커를 시작합니다
func (gw *GateWorker) Start(ctx context.Context) {
	gw.mu.Lock()
	if gw.running {
		gw.mu.Unlock()
		return
	}

	gw.running = true
	gw.mu.Unlock()

	// 주기적으로 매도 주문 실행
	go func() {
		ticker := time.NewTicker(time.Duration(float64(time.Second) * gw.config.RequestInterval))
		defer ticker.Stop()

		for {
			// 실행 상태 확인
			gw.mu.RLock()
			if !gw.running {
				gw.mu.RUnlock()
				return
			}
			gw.mu.RUnlock()

			select {
			case <-ticker.C:
				// 실행 상태 재확인 후 요청 처리
				gw.mu.RLock()
				if gw.running {
					gw.mu.RUnlock()
					gw.executeSellOrder()
				} else {
					gw.mu.RUnlock()
					return
				}
			case <-gw.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop 워커를 중지합니다
func (gw *GateWorker) Stop() {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	if !gw.running {
		return
	}

	gw.running = false
	close(gw.stopCh)
	gw.storage.AddLog("info", "Gate.io APIv4 워커가 중지되었습니다.", gw.config.Exchange, gw.config.Symbol)
}

// IsRunning 워커 실행 상태를 반환합니다
func (gw *GateWorker) IsRunning() bool {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	return gw.running
}

// executeSellOrder Gate.io APIv4로 매도 주문 실행
func (gw *GateWorker) executeSellOrder() {
	// 실행 상태 재확인
	gw.mu.RLock()
	if !gw.running {
		gw.mu.RUnlock()
		return
	}
	gw.mu.RUnlock()

	if gw.config == nil {
		gw.storage.AddLog("error", "Gate.io 설정이 nil입니다.", gw.config.Exchange, gw.config.Symbol)
		return
	}

	// 심볼을 Gate.io 형식으로 변환 (예: BTC/USDT -> BTC_USDT)
	currencyPair := strings.ReplaceAll(gw.config.Symbol, "/", "_")
	if currencyPair == "" {
		gw.storage.AddLog("error", "심볼이 설정되지 않았습니다.", gw.config.Exchange, gw.config.Symbol)
		return
	}

	// APIv4 직접 구현으로 매도 주문 실행
	result := gw.executeGateAPISellOrder(currencyPair)

	if result.Success {
		gw.storage.AddLog("success", fmt.Sprintf("Gate.io APIv4 매도 주문 성공: 주문번호=%s, 가격=%.8f, 수량=%.8f, 통화쌍=%s",
			result.OrderID, result.Price, result.Amount, currencyPair), gw.config.Exchange, gw.config.Symbol)
	} else {
		gw.storage.AddLog("error", fmt.Sprintf("Gate.io APIv4 매도 주문 실패: %s", result.ErrorMessage), gw.config.Exchange, gw.config.Symbol)
	}
}

// executeGateAPISellOrder Gate.io APIv4 직접 호출로 매도 주문 실행
func (gw *GateWorker) executeGateAPISellOrder(currencyPair string) OrderResult {
	apiURL := "https://api.gateio.ws/api/v4/spot/orders"

	// Unix timestamp in seconds
	timestamp := time.Now().Unix()

	// 심볼을 Gate.io 형식으로 변환 (이미 변환된 값이 들어오지만 안전하게 한 번 더 처리)
	if currencyPair == "" {
		currencyPair = strings.ReplaceAll(gw.config.Symbol, "/", "_")
	}

	// 심볼 정밀도 정보 조회 (최초 1회만 API 호출, 이후 캐시 사용)
	symbolInfo, err := gw.getGateSymbolInfo(currencyPair)
	if err != nil {
		gw.storage.AddLog("warning", fmt.Sprintf("Gate.io 심볼 정밀도 조회 실패, 기본 정밀도로 진행합니다: %v", err), gw.config.Exchange, gw.config.Symbol)
	}

	// 수량은 정수만, 가격은 원래 정밀도 사용
	pricePrecision := 8
	if symbolInfo != nil && symbolInfo.PricePrecision > 0 {
		pricePrecision = symbolInfo.PricePrecision
	}
	formattedAmountStr := truncateToPrecision(gw.config.SellAmount, 0)
	formattedPriceStr := truncateToPrecision(gw.config.SellPrice, pricePrecision)

	formattedAmount, _ := strconv.ParseFloat(formattedAmountStr, 64)
	formattedPrice, _ := strconv.ParseFloat(formattedPriceStr, 64)

	requestBody := map[string]interface{}{
		"currency_pair": currencyPair,
		"side":          "sell",
		"type":          "limit",
		"amount":        formattedAmountStr,
		"price":         formattedPriceStr,
		"text":          fmt.Sprintf("t-bitbitbot_%d", time.Now().Unix()), // 사용자 정의 정보
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "요청 바디 생성 실패: " + err.Error()}
	}

	// APIv4 서명 문자열 생성
	// Request Method + "\n" + Request URL + "\n" + Query String + "\n" + HexEncode(SHA512(Request Payload)) + "\n" + Timestamp
	queryString := "" // 쿼리 파라미터 없음

	// SHA512로 요청 바디 해시
	payloadHash := sha512.Sum512(jsonBody)
	payloadHashHex := hex.EncodeToString(payloadHash[:])

	signatureString := fmt.Sprintf("POST\n/api/v4/spot/orders\n%s\n%s\n%d",
		queryString, payloadHashHex, timestamp)

	// HMAC-SHA512 서명 생성
	signature := gw.generateGateSignature(signatureString, gw.config.SecretKey)

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 생성 실패: " + err.Error()}
	}

	// APIv4 헤더 설정
	req.Header.Set("KEY", gw.config.AccessKey)
	req.Header.Set("Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("SIGN", signature)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 요청 실행
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "HTTP 요청 실패: " + err.Error()}
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OrderResult{Success: false, ErrorMessage: "응답 읽기 실패: " + err.Error()}
	}

	// 응답 파싱
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return OrderResult{Success: false, ErrorMessage: "응답 파싱 실패: " + err.Error()}
	}

	// 성공 여부 확인 (201 Created)
	if resp.StatusCode == 201 {
		orderID := ""
		if result["id"] != nil {
			orderID = fmt.Sprintf("%v", result["id"])
		}

		return OrderResult{
			Success:      true,
			OrderID:      orderID,
			Price:        formattedPrice,
			Amount:       formattedAmount,
			TotalAmount:  formattedAmount * formattedPrice,
			ErrorMessage: "",
		}
	} else {
		// 에러 메시지 추출
		errorMsg := "알 수 없는 오류"
		if result["message"] != nil {
			errorMsg = fmt.Sprintf("%v", result["message"])
		} else if result["error"] != nil {
			errorMsg = fmt.Sprintf("%v", result["error"])
		}

		return OrderResult{
			Success:      false,
			OrderID:      "",
			Price:        formattedPrice,
			Amount:       formattedAmount,
			TotalAmount:  formattedAmount * formattedPrice,
			ErrorMessage: fmt.Sprintf("Gate.io API 오류 (상태코드: %d): %s", resp.StatusCode, errorMsg),
		}
	}
}

// generateGateSignature Gate.io APIv4 전용 서명 생성 (HMAC-SHA512)
func (gw *GateWorker) generateGateSignature(message, secretKey string) string {
	h := hmac.New(sha512.New, []byte(secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (gw *GateWorker) GetPlatformName() string {
	return "Gate.io"
}

// getGateSymbolInfo Gate.io 심볼 정밀도 정보를 조회하고 캐시합니다.
func (gw *GateWorker) getGateSymbolInfo(currencyPair string) (*GateSymbolInfo, error) {
	// 캐시 확인
	gw.symbolInfoMu.RLock()
	if info, ok := gw.symbolInfoCache[currencyPair]; ok {
		gw.symbolInfoMu.RUnlock()
		return info, nil
	}
	gw.symbolInfoMu.RUnlock()

	type gatePairRaw struct {
		CurrencyPair    string `json:"currency_pair"`
		AmountPrecision int    `json:"amount_precision,omitempty"`
		Precision       int    `json:"precision,omitempty"`

		MinBaseAmount  string `json:"min_base_amount,omitempty"`
		MinQuoteAmount string `json:"min_quote_amount,omitempty"`
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.gateio.ws/api/v4/spot/currency_pairs", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pairs []gatePairRaw
	if err := json.NewDecoder(resp.Body).Decode(&pairs); err != nil {
		return nil, err
	}

	for _, p := range pairs {
		if strings.EqualFold(p.CurrencyPair, currencyPair) {
			info := &GateSymbolInfo{
				CurrencyPair:    p.CurrencyPair,
				AmountPrecision: p.AmountPrecision,
				Precision:       p.Precision,
				MinBaseAmount:   p.MinBaseAmount,
				MinQuoteAmount:  p.MinQuoteAmount,
			}

			// 가격 정밀도 계산
			if info.Precision > 0 {
				info.PricePrecision = info.Precision
			} else if info.MinQuoteAmount != "" {
				info.PricePrecision = countDecimalPlaces(info.MinQuoteAmount)
			}

			// 수량 정밀도 계산
			if info.AmountPrecision > 0 {
				info.AmountPrecision2 = info.AmountPrecision
			} else if info.MinBaseAmount != "" {
				info.AmountPrecision2 = countDecimalPlaces(info.MinBaseAmount)
			}

			// 기본값 보정
			if info.PricePrecision <= 0 {
				info.PricePrecision = 8
			}
			if info.AmountPrecision2 <= 0 {
				info.AmountPrecision2 = 8
			}

			// 캐시에 저장
			gw.symbolInfoMu.Lock()
			gw.symbolInfoCache[currencyPair] = info
			gw.symbolInfoMu.Unlock()

			return info, nil
		}
	}

	return nil, fmt.Errorf("Gate.io 심볼 정보를 찾을 수 없습니다: %s", currencyPair)
}
