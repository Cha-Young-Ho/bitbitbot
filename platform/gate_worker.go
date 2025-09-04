package platform

import (
	"bitbit-app/local_file"
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

// GateWorker Gate.io 플랫폼용 워커 (APIv4 직접 구현)
type GateWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	url       string
}

// NewGateWorker 새로운 Gate.io 워커를 생성합니다
func NewGateWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *GateWorker {
	return &GateWorker{
		BaseWorker: NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase),
		accessKey:  accessKey,
		secretKey:  secretKey,
		url:        "https://api.gateio.ws/api/v4/spot/orders",
	}
}

// Start 워커를 시작합니다
func (gw *GateWorker) Start(ctx context.Context) error {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	if gw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", gw.order.Name)
	}

	gw.ctx, gw.cancel = context.WithCancel(ctx)
	gw.isRunning = true
	gw.status.IsRunning = true

	// 워커 고루틴 시작
	go gw.run()
	return nil
}

// run 워커의 메인 루프
func (gw *GateWorker) run() {
	// Term(초)이 소수일 수 있으므로 밀리초로 변환하여 절삭 방지, 최소 1ms 보장
	intervalMs := int64(gw.order.Term * 1000)
	if intervalMs < 1 {
		intervalMs = 1
	}
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-gw.ctx.Done():
			gw.sendLog("Gate.io 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			// 매 tick마다 반드시 실행: 비동기 고루틴으로 처리 (이전 요청 진행 중이어도 새 요청 즉시 시작)
			go gw.executeSellOrder(gw.order.Price)
		}
	}
}

// executeSellOrder Gate.io APIv4로 매도 주문 실행
func (gw *GateWorker) executeSellOrder(price float64) {
	// APIv4 직접 구현으로 매도 주문 실행
	result := gw.executeGateAPISellOrder(price)
	
	if success, ok := result["success"].(bool); ok && success {
		// 성공 시 간단한 로그만 출력
		gw.sendLog("주문 성공", "success", price, gw.order.Quantity)
	} else {
		errorMsg := "알 수 없는 오류"
		if msg, exists := result["errorMessage"].(string); exists {
			errorMsg = msg
		}
		// Gate.io 에러 메시지에서 핵심 메시지만 추출
		cleanErrorMsg := gw.parseGateError(errorMsg)
		
		// 실패 시 일관된 로그 포맷으로 출력
		gw.sendLog(fmt.Sprintf("주문 실패\n이유: %s\n심볼: %s\n가격: %.8f", 
			cleanErrorMsg, gw.order.Symbol, price), "error", price, gw.order.Quantity)
	}
}

// executeGateAPISellOrder Gate.io APIv4 직접 호출로 매도 주문 실행
func (gw *GateWorker) executeGateAPISellOrder(price float64) map[string]interface{} {
	// Unix timestamp in seconds
	timestamp := time.Now().Unix()

	// 심볼을 Gate.io 형식으로 변환 (예: BTC/USDT -> BTC_USDT)
	currencyPair := strings.ReplaceAll(gw.order.Symbol, "/", "_")

	// 요청 바디 구성
	requestBody := map[string]interface{}{
		"currency_pair": currencyPair,
		"side":          "sell",
		"type":          "limit",
		"amount":        fmt.Sprintf("%.8f", gw.order.Quantity),
		"price":         fmt.Sprintf("%.8f", price),
		"time_in_force": "gtc", // Good Till Cancelled
		"text":          fmt.Sprintf("t-bitbitbot_%d", time.Now().Unix()), // 사용자 정의 정보
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return map[string]interface{}{
			"success":      false,
			"errorMessage": "요청 바디 생성 실패: " + err.Error(),
		}
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
	signature := gw.generateGateSignature(signatureString, gw.secretKey)

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", gw.url, bytes.NewReader(jsonBody))
	if err != nil {
		return map[string]interface{}{
			"success":      false,
			"errorMessage": "HTTP 요청 생성 실패: " + err.Error(),
		}
	}

	// APIv4 헤더 설정
	req.Header.Set("KEY", gw.accessKey)
	req.Header.Set("Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("SIGN", signature)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 요청 실행
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"success":      false,
			"errorMessage": "HTTP 요청 실패: " + err.Error(),
		}
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"success":      false,
			"errorMessage": "응답 읽기 실패: " + err.Error(),
		}
	}

	// 응답 파싱
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]interface{}{
			"success":      false,
			"errorMessage": "응답 파싱 실패: " + err.Error(),
		}
	}

	// 응답을 콘솔에 출력 (로그 포맷터 사용)
	gw.printExchangeResponse("Gate.io", resp.StatusCode, body, result)

	// 성공 여부 확인 (201 Created)
	if resp.StatusCode == 201 {
		orderID := ""
		if result["id"] != nil {
			orderID = fmt.Sprintf("%v", result["id"])
		}

		return map[string]interface{}{
			"success":     true,
			"orderId":     orderID,
			"price":       price,
			"amount":      gw.order.Quantity,
			"totalAmount": gw.order.Quantity * price,
			"errorMessage": "",
		}
	} else {
		// 에러 메시지 추출
		errorMsg := "알 수 없는 오류"
		if result["message"] != nil {
			errorMsg = fmt.Sprintf("%v", result["message"])
		} else if result["error"] != nil {
			errorMsg = fmt.Sprintf("%v", result["error"])
		}

		return map[string]interface{}{
			"success":      false,
			"orderId":      "",
			"price":        0,
			"amount":       0,
			"totalAmount":  0,
			"errorMessage": fmt.Sprintf("Gate.io API 오류 (상태코드: %d): %s", resp.StatusCode, errorMsg),
		}
	}
}

// generateGateSignature Gate.io APIv4 전용 서명 생성 (HMAC-SHA512)
func (gw *GateWorker) generateGateSignature(message, secretKey string) string {
	h := hmac.New(sha512.New, []byte(secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// printExchangeResponse 거래소 응답을 콘솔에 출력
func (gw *GateWorker) printExchangeResponse(exchangeName string, statusCode int, rawBody []byte, parsedResult map[string]interface{}) {
	fmt.Printf("\n=== %s API 응답 ===\n", exchangeName)
	fmt.Printf("상태 코드: %d\n", statusCode)
	fmt.Printf("원본 응답: %s\n", string(rawBody))
	fmt.Printf("파싱된 결과: %+v\n", parsedResult)
	fmt.Printf("=== %s 응답 끝 ===\n\n", exchangeName)
}

// parseGateError Gate.io 에러 메시지에서 핵심 메시지만 추출
func (gw *GateWorker) parseGateError(errorMsg string) string {
	// "Gate.io API 오류 (상태코드: 400): Not enough balance" 형태에서 "Not enough balance"만 추출
	if strings.Contains(errorMsg, "Gate.io API 오류") && strings.Contains(errorMsg, ":") {
		parts := strings.Split(errorMsg, ":")
		if len(parts) >= 2 {
			// 마지막 부분에서 실제 에러 메시지 추출
			cleanMsg := strings.TrimSpace(parts[len(parts)-1])
			return cleanMsg
		}
	}
	
	// 다른 형태의 에러 메시지는 그대로 반환
	return errorMsg
}

// formatLogMessage 로그 메시지 포맷팅
func (gw *GateWorker) formatLogMessage(messageType, message string, price, quantity float64) string {
	timestamp := time.Now().Format("15:04:05")
	
	switch messageType {
	case "order":
		return fmt.Sprintf("[%s] %s | 가격: %.8f | 수량: %.8f", timestamp, message, price, quantity)
	case "success":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	case "error":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	case "info":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	case "warning":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	default:
		return fmt.Sprintf("[%s] %s", timestamp, message)
	}
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (gw *GateWorker) GetPlatformName() string {
	return "Gate.io"
}
