package platform

import (
	"bitbit-app/local_file"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// CoinoneWorker Coinone 거래소 워커
type CoinoneWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	url       string
}

// NewCoinoneWorker 새로운 Coinone 워커를 생성합니다
func NewCoinoneWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *CoinoneWorker {
	// CCXT는 생성만 하고 사용하지 않음 (코인원은 직접 HTTP API 사용)
	config := map[string]interface{}{
		"apiKey":          accessKey,
		"secret":          secretKey,
		"enableRateLimit": true,
	}
	_ = ccxt.NewCoinone(config) // 생성만 하고 사용하지 않음

	return &CoinoneWorker{
		BaseWorker: NewBaseWorker(order, manager), // CCXT 없이 생성
		accessKey:  accessKey,
		secretKey:  secretKey,
		url:        "https://api.coinone.co.kr/v2.1/order",
	}
}

// Start 워커를 시작합니다
func (cw *CoinoneWorker) Start(ctx context.Context) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.status.IsRunning {
		return fmt.Errorf("이미 실행 중입니다")
	}

	cw.ctx, cw.cancel = context.WithCancel(ctx)
	cw.status.IsRunning = true

	go cw.run() // 워커 루프 시작

	fmt.Printf("Coinone 워커 시작: %s", cw.order.Name)
	return nil
}

// Stop 워커를 중지합니다
func (cw *CoinoneWorker) Stop() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if !cw.status.IsRunning {
		return nil
	}

	cw.status.IsRunning = false
	cw.cancel() // 컨텍스트 취소

	fmt.Printf("Coinone 워커 중지: %s", cw.order.Name)
	return nil
}

// run 워커의 메인 루프
func (cw *CoinoneWorker) run() {
	ticker := time.NewTicker(time.Duration(cw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	cw.sendLog("Coinone 워커가 시작되었습니다", "info")
	fmt.Printf("[Coinone] 워커 시작 - 주문명: %s, 심볼: %s, 지정가: %.2f, 주기: %.1f초\n",
		cw.order.Name, cw.order.Symbol, cw.order.Price, cw.order.Term)

	for {
		select {
		case <-cw.ctx.Done():
			cw.sendLog("Coinone 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			cw.executeSellOrder()
		}
	}
}

// executeSellOrder Coinone에서 매도 주문 실행
func (cw *CoinoneWorker) executeSellOrder() {
	cw.mu.Lock()
	cw.status.LastCheck = time.Now()
	cw.status.CheckCount++
	cw.mu.Unlock()

	// 심볼 변환 (BTC/KRW -> BTC)
	coinoneSymbol := convertToCoinoneSymbol(cw.order.Symbol)
	cw.sendLog(fmt.Sprintf("변환된 심볼: %s", coinoneSymbol), "info")

	// 주문 시도 로그
	cw.sendLog(fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		coinoneSymbol, cw.order.Quantity, cw.order.Price), "info")

	// Coinone API 직접 호출
	orderID, err := cw.createCoinoneOrder(coinoneSymbol)
	if err != nil {
		cw.mu.Lock()
		cw.status.ErrorCount++
		cw.status.LastError = err.Error()
		cw.mu.Unlock()

		cw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error")
		cw.manager.SendSystemLog("CoinoneWorker", "executeSellOrder",
			fmt.Sprintf("매도 주문 실패: %v", err), "error", "", cw.order.Name, err.Error())
		return
	}

	// 성공 로그
	cw.sendLog(fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		cw.order.Price, cw.order.Quantity, orderID), "success", cw.order.Price, cw.order.Quantity)
}

// createCoinoneOrder Coinone API를 직접 호출하여 주문 생성
func (cw *CoinoneWorker) createCoinoneOrder(coinoneSymbol string) (string, error) {
	// 1. 요청 바디 구성 (Coinone API 문서 기준)
	nonce := strconv.FormatInt(time.Now().UnixMilli(), 10)

	requestBody := map[string]interface{}{
		"access_token":    cw.accessKey,
		"nonce":           nonce,
		"side":            "sell", // 매도
		"quote_currency":  "KRW",
		"target_currency": coinoneSymbol,
		"type":            "limit",
		"price":           fmt.Sprintf("%.0f", cw.order.Price),
		"qty":             fmt.Sprintf("%.8f", cw.order.Quantity),
		"post_only":       "1",
	}

	cw.sendLog(fmt.Sprintf("Coinone 주문 요청: %+v", requestBody), "info")

	// 2. JSON 문자열로 변환
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON 변환 실패: %v", err)
	}

	// 3. Base64 인코딩 (페이로드)
	payload := base64.StdEncoding.EncodeToString(jsonBody)

	// 4. HMAC-SHA512 서명 생성
	signature := cw.createCoinoneSignature(payload)

	cw.sendLog(fmt.Sprintf("Coinone 서명: %s", signature), "info")

	// 5. HTTP 요청 생성
	req, err := http.NewRequestWithContext(cw.ctx, "POST", cw.url, bytes.NewReader(jsonBody))
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

	// 9. 응답 검증
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API 오류 (status=%d): %v", resp.StatusCode, response)
	}

	// 10. 주문 ID 추출
	orderID, ok := response["order_id"].(string)
	if !ok || orderID == "" {
		return "", fmt.Errorf("주문 ID 없음: %v", response)
	}

	return orderID, nil
}

// createCoinoneSignature Coinone HMAC-SHA512 서명 생성
func (cw *CoinoneWorker) createCoinoneSignature(payload string) string {
	// HMAC-SHA512 서명 생성
	h := hmac.New(sha512.New, []byte(cw.secretKey))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// convertToCoinoneSymbol 심볼을 Coinone 형식으로 변환
func convertToCoinoneSymbol(symbol string) string {
	// BTC/KRW -> BTC
	// USDT/KRW -> USDT
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return symbol
}

// GetPlatformName 플랫폼 이름 반환
func (cw *CoinoneWorker) GetPlatformName() string {
	return "coinone"
}
