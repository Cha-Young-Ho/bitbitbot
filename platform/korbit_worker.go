package platform

import (
	"bitbit-app/local_file"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// KorbitWorker Korbit 거래소 워커
type KorbitWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	url       string
}

// NewKorbitWorker 새로운 Korbit 워커를 생성합니다
func NewKorbitWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *KorbitWorker {
	// CCXT는 생성만 하고 사용하지 않음 (코빗은 직접 HTTP API 사용)
	config := map[string]interface{}{
		"apiKey":          accessKey,
		"secret":          secretKey,
		"enableRateLimit": true,
	}
	_ = ccxt.CreateExchange("korbit", config) // 생성만 하고 사용하지 않음

	return &KorbitWorker{
		BaseWorker: NewBaseWorker(order, manager), // CCXT 없이 생성
		accessKey:  accessKey,
		secretKey:  secretKey,
		url:        "https://api.korbit.co.kr/v2/orders",
	}
}

// Start 워커를 시작합니다
func (kw *KorbitWorker) Start(ctx context.Context) error {
	kw.mu.Lock()
	defer kw.mu.Unlock()

	if kw.status.IsRunning {
		return fmt.Errorf("이미 실행 중입니다")
	}

	kw.ctx, kw.cancel = context.WithCancel(ctx)
	kw.status.IsRunning = true

	go kw.run() // 워커 루프 시작

	fmt.Printf("Korbit 워커 시작: %s", kw.order.Name)
	return nil
}

// Stop 워커를 중지합니다
func (kw *KorbitWorker) Stop() error {
	kw.mu.Lock()
	defer kw.mu.Unlock()

	if !kw.status.IsRunning {
		return nil
	}

	kw.status.IsRunning = false
	kw.cancel() // 컨텍스트 취소

	fmt.Printf("Korbit 워커 중지: %s", kw.order.Name)
	return nil
}

// run 워커의 메인 루프
func (kw *KorbitWorker) run() {
	ticker := time.NewTicker(time.Duration(kw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	kw.sendLog("Korbit 워커가 시작되었습니다", "info")
	fmt.Printf("[Korbit] 워커 시작 - 주문명: %s, 심볼: %s, 지정가: %.2f, 주기: %.1f초\n",
		kw.order.Name, kw.order.Symbol, kw.order.Price, kw.order.Term)

	for {
		select {
		case <-kw.ctx.Done():
			kw.sendLog("Korbit 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			kw.executeSellOrder()
		}
	}
}

// executeSellOrder Korbit에서 매도 주문 실행
func (kw *KorbitWorker) executeSellOrder() {
	kw.mu.Lock()
	kw.status.LastCheck = time.Now()
	kw.status.CheckCount++
	kw.mu.Unlock()

	// 심볼 변환 (BTC/KRW -> btc_krw)
	korbitSymbol := convertToKorbitSymbol(kw.order.Symbol)
	kw.sendLog(fmt.Sprintf("변환된 심볼: %s", korbitSymbol), "info")

	// 주문 시도 로그
	kw.sendLog(fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		korbitSymbol, kw.order.Quantity, kw.order.Price), "info")

	// Korbit API 직접 호출
	orderID, err := kw.createKorbitOrder(korbitSymbol)
	if err != nil {
		kw.mu.Lock()
		kw.status.ErrorCount++
		kw.status.LastError = err.Error()
		kw.mu.Unlock()

		kw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error")
		kw.manager.SendSystemLog("KorbitWorker", "executeSellOrder",
			fmt.Sprintf("매도 주문 실패: %v", err), "error", "", kw.order.Name, err.Error())
		return
	}

	// 성공 로그
	kw.sendLog(fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		kw.order.Price, kw.order.Quantity, orderID), "success", kw.order.Price, kw.order.Quantity)
}

// createKorbitOrder Korbit API를 직접 호출하여 주문 생성
func (kw *KorbitWorker) createKorbitOrder(korbitSymbol string) (string, error) {
	// 1. 요청 변수 구성 (Korbit API 문서 기준)
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	params := url.Values{}
	params.Set("symbol", korbitSymbol) // btc_krw
	params.Set("side", "sell")         // 매도
	params.Set("price", fmt.Sprintf("%.0f", kw.order.Price))
	params.Set("qty", fmt.Sprintf("%.8f", kw.order.Quantity))
	params.Set("orderType", "limit") // 지정가
	params.Set("timeInForce", "gtc") // Good Till Cancel
	params.Set("timestamp", timestamp)

	kw.sendLog(fmt.Sprintf("Korbit 주문 요청: %s", params.Encode()), "info")

	// 2. HMAC-SHA256 서명 생성
	signature := kw.createKorbitSignature(params.Encode())
	params.Set("signature", signature)

	kw.sendLog(fmt.Sprintf("Korbit 서명: %s", signature), "info")

	// 3. HTTP 요청 생성
	req, err := http.NewRequestWithContext(kw.ctx, "POST", kw.url, strings.NewReader(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 4. 헤더 설정 (Korbit API 문서 기준)
	req.Header.Set("X-KAPI-KEY", kw.accessKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 5. HTTP 요청 전송
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("요청 전송 실패: %v", err)
	}
	defer resp.Body.Close()

	// 6. 응답 파싱
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 7. 응답 검증 (200이면 성공, 아니면 실패)
	if resp.StatusCode == 200 {
		// 성공
		return "success", nil
	} else {
		// 실패
		return "", fmt.Errorf("매도 주문 실패 (status=%d): %v", resp.StatusCode, response)
	}
}

// createKorbitSignature Korbit HMAC-SHA256 서명 생성 (API 문서 기준)
func (kw *KorbitWorker) createKorbitSignature(queryString string) string {
	// HMAC-SHA256 서명 생성
	h := hmac.New(sha256.New, []byte(kw.secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// convertToKorbitSymbol 심볼을 Korbit 형식으로 변환
func convertToKorbitSymbol(symbol string) string {
	// BTC/KRW -> btc_krw
	// USDT/KRW -> usdt_krw
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return strings.ToLower(parts[0]) + "_" + strings.ToLower(parts[1])
	}
	return strings.ToLower(symbol)
}

// GetPlatformName 플랫폼 이름 반환
func (kw *KorbitWorker) GetPlatformName() string {
	return "korbit"
}
