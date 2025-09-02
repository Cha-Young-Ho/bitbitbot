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
	"time"
)

// GateWorker Gate.io 거래소 워커 (APIv4 직접 구현)
type GateWorker struct {
	config  *WorkerConfig
	storage *MemoryStorage
	running bool
	stopCh  chan struct{}
}

// NewGateWorker 새로운 Gate.io 워커를 생성합니다
func NewGateWorker(config *WorkerConfig, storage *MemoryStorage) *GateWorker {
	return &GateWorker{
		config:   config,
		storage:  storage,
		running:  false,
		stopCh:   make(chan struct{}),
	}
}

// Start 워커를 시작합니다
func (gw *GateWorker) Start(ctx context.Context) {
	if gw.running {
		return
	}

	gw.running = true
	gw.storage.AddLog("info", "Gate.io APIv4 워커가 시작되었습니다.", gw.config.Exchange, gw.config.Symbol)

	// 주기적으로 매도 주문 실행
	go func() {
		ticker := time.NewTicker(time.Duration(float64(time.Second) * gw.config.RequestInterval))
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if gw.running {
					gw.executeSellOrder()
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
	if !gw.running {
		return
	}

	gw.running = false
	close(gw.stopCh)
	gw.storage.AddLog("info", "Gate.io APIv4 워커가 중지되었습니다.", gw.config.Exchange, gw.config.Symbol)
}

// IsRunning 워커 실행 상태를 반환합니다
func (gw *GateWorker) IsRunning() bool {
	return gw.running
}

// executeSellOrder Gate.io APIv4로 매도 주문 실행
func (gw *GateWorker) executeSellOrder() {
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
	result := gw.executeGateAPISellOrder()
	
	if result.Success {
		gw.storage.AddLog("success", fmt.Sprintf("Gate.io APIv4 매도 주문 성공: 주문번호=%s, 가격=%.8f, 수량=%.8f, 통화쌍=%s",
			result.OrderID, gw.config.SellPrice, gw.config.SellAmount, currencyPair), gw.config.Exchange, gw.config.Symbol)
	} else {
		gw.storage.AddLog("error", fmt.Sprintf("Gate.io APIv4 매도 주문 실패: %s", result.ErrorMessage), gw.config.Exchange, gw.config.Symbol)
	}
}

// executeGateAPISellOrder Gate.io APIv4 직접 호출로 매도 주문 실행
func (gw *GateWorker) executeGateAPISellOrder() OrderResult {
	apiURL := "https://api.gateio.ws/api/v4/spot/orders"

	// Unix timestamp in seconds
	timestamp := time.Now().Unix()

	// 심볼을 Gate.io 형식으로 변환
	currencyPair := strings.ReplaceAll(gw.config.Symbol, "/", "_")

	// 요청 바디 구성
	requestBody := map[string]interface{}{
		"currency_pair": currencyPair,
		"side":          "sell",
		"type":          "limit",
		"amount":        fmt.Sprintf("%.8f", gw.config.SellAmount),
		"price":         fmt.Sprintf("%.8f", gw.config.SellPrice),
		"time_in_force": "gtc", // Good Till Cancelled
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
			Success:     true,
			OrderID:     orderID,
			Price:       gw.config.SellPrice,
			Amount:      gw.config.SellAmount,
			TotalAmount: gw.config.SellAmount * gw.config.SellPrice,
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
			Price:        0,
			Amount:       0,
			TotalAmount:  0,
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
