package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/google/uuid"
)

// CoinoneWorker 코인원 거래소 워커
type CoinoneWorker struct {
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	url       string
	exchange  ccxt.IExchange
}

// NewCoinoneWorker 새로운 코인원 워커를 생성합니다
func NewCoinoneWorker(config *WorkerConfig, storage *MemoryStorage) *CoinoneWorker {
	// CCXT는 생성만 하고 사용하지 않음 (코인원은 직접 HTTP API 사용)
	exchangeConfig := map[string]interface{}{
		"apiKey":          config.AccessKey,
		"secret":          config.SecretKey,
		"timeout":         30000, // 30초
		"sandbox":         false, // 실제 거래
		"enableRateLimit": true,
	}

	// Password Phrase가 있으면 추가
	if config.PasswordPhrase != "" {
		exchangeConfig["password"] = config.PasswordPhrase
	}

	exchange := ccxt.CreateExchange("coinone", exchangeConfig)

	return &CoinoneWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		url:       "https://api.coinone.co.kr/v2.1/order",
		exchange:  exchange,
	}
}

// Start 워커를 시작합니다
func (cw *CoinoneWorker) Start(ctx context.Context) {
	cw.running = true
	cw.storage.AddLog("info", "코인원 워커가 시작되었습니다.", cw.config.Exchange, cw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(cw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cw.running = false
			cw.storage.AddLog("info", "코인원 워커가 중지되었습니다.", cw.config.Exchange, cw.config.Symbol)
			return
		case <-cw.stopCh:
			cw.running = false
			cw.storage.AddLog("info", "코인원 워커가 중지되었습니다.", cw.config.Exchange, cw.config.Symbol)
			return
		case <-ticker.C:
			cw.executeSellOrder()
		}
	}
}

// Stop 워커를 중지합니다
func (cw *CoinoneWorker) Stop() {
	if cw.running {
		close(cw.stopCh)
		cw.running = false
	}
}

// IsRunning 워커 실행 상태 확인
func (cw *CoinoneWorker) IsRunning() bool {
	return cw.running
}

// executeSellOrder 코인원에서 매도 주문 실행
func (cw *CoinoneWorker) executeSellOrder() {
	// 심볼 변환 (BTC/KRW -> BTC)
	coinoneSymbol := cw.convertToCoinoneSymbol(cw.config.Symbol)
	cw.storage.AddLog("info", fmt.Sprintf("변환된 심볼: %s", coinoneSymbol), cw.config.Exchange, cw.config.Symbol)

	// 주문 시도 로그
	cw.storage.AddLog("info", fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		coinoneSymbol, cw.config.SellAmount, cw.config.SellPrice), cw.config.Exchange, cw.config.Symbol)

	// Coinone API 직접 호출
	orderID, err := cw.createCoinoneOrder(coinoneSymbol)
	if err != nil {
		cw.storage.AddLog("error", fmt.Sprintf("매도 주문 실패: %v", err), cw.config.Exchange, cw.config.Symbol)
		return
	}

	// 성공 로그
	cw.storage.AddLog("success", fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		cw.config.SellPrice, cw.config.SellAmount, orderID), cw.config.Exchange, cw.config.Symbol)
}

// createCoinoneOrder Coinone API를 직접 호출하여 주문 생성
func (cw *CoinoneWorker) createCoinoneOrder(coinoneSymbol string) (string, error) {
	// 1. 요청 바디 구성 (Coinone API 문서 기준)
	nonce := uuid.New().String() // UUID v4 형식

	requestBody := map[string]interface{}{
		"access_token":    cw.accessKey,
		"nonce":           nonce,
		"side":            "SELL", // 매도 (대문자)
		"quote_currency":  "KRW",
		"target_currency": coinoneSymbol,
		"type":            "LIMIT", // 지정가 (대문자)
		"price":           fmt.Sprintf("%.0f", cw.config.SellPrice),
		"qty":             fmt.Sprintf("%.8f", cw.config.SellAmount),
		"post_only":       true, // Boolean 타입
	}

	cw.storage.AddLog("info", fmt.Sprintf("Coinone 주문 요청: %+v", requestBody), cw.config.Exchange, cw.config.Symbol)

	// 2. JSON 문자열로 변환
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON 변환 실패: %v", err)
	}

	// 3. Base64 인코딩 (페이로드)
	payload := base64.StdEncoding.EncodeToString(jsonBody)

	// 4. HMAC-SHA512 서명 생성
	signature := cw.createCoinoneSignature(payload)

	cw.storage.AddLog("info", fmt.Sprintf("Coinone 서명: %s", signature), cw.config.Exchange, cw.config.Symbol)

	// 5. HTTP 요청 생성
	req, err := http.NewRequest("POST", cw.url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 6. 헤더 설정 (Coinone API 문서 기준)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-COINONE-PAYLOAD", payload)
	req.Header.Set("X-COINONE-SIGNATURE", signature)

	// 7. HTTP 요청 전송
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("요청 전송 실패: %v", err)
	}
	defer resp.Body.Close()

	// 8. 응답 파싱
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 응답 전체를 로그에 출력 (디버깅용)
	cw.storage.AddLog("info", fmt.Sprintf("코인원 API 응답 (status=%d): %+v", resp.StatusCode, response), cw.config.Exchange, cw.config.Symbol)

	// 9. 응답 검증
	if resp.StatusCode != 200 {
		errorMsg := "알 수 없는 오류"
		if response["errorCode"] != nil {
			errorMsg = fmt.Sprintf("에러코드: %v", response["errorCode"])
		}
		if response["errorMsg"] != nil {
			errorMsg += fmt.Sprintf(", 메시지: %v", response["errorMsg"])
		}
		if response["message"] != nil {
			errorMsg += fmt.Sprintf(", 상세: %v", response["message"])
		}
		return "", fmt.Errorf("API 오류 (status=%d): %s", resp.StatusCode, errorMsg)
	}

	// 10. 주문 ID 추출
	orderID, ok := response["order_id"].(string)
	if !ok || orderID == "" {
		return "", fmt.Errorf("주문 ID 없음: %v", response)
	}

	return orderID, nil
}

// createCoinoneSignature 코인원 HMAC-SHA512 서명 생성
func (cw *CoinoneWorker) createCoinoneSignature(payload string) string {
	// HMAC-SHA512 서명 생성
	h := hmac.New(sha512.New, []byte(cw.secretKey))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// convertToCoinoneSymbol 심볼을 코인원 형식으로 변환
func (cw *CoinoneWorker) convertToCoinoneSymbol(symbol string) string {
	// BTC/KRW -> BTC
	// USDT/KRW -> USDT
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return symbol
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (cw *CoinoneWorker) GetPlatformName() string {
	return "Coinone"
}
