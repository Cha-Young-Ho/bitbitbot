package platform

import (
	"bitbit-app/local_file"
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UpbitWorker Upbit 플랫폼용 워커
type UpbitWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	url       string
	accounts  string
}

// NewUpbitWorker 새로운 Upbit 워커를 생성합니다
func NewUpbitWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *UpbitWorker {
	return &UpbitWorker{
		BaseWorker: NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase),
		accessKey:  accessKey,
		secretKey:  secretKey,
		url:        "https://api.upbit.com/v1/orders",
		accounts:   "https://api.upbit.com/v1/accounts",
	}
}

// Start 워커를 시작합니다
func (uw *UpbitWorker) Start(ctx context.Context) error {
	uw.mu.Lock()
	defer uw.mu.Unlock()

	if uw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", uw.order.Name)
	}

	uw.ctx, uw.cancel = context.WithCancel(ctx)
	uw.isRunning = true
	uw.status.IsRunning = true

	// 워커 고루틴 시작
	go uw.run()
	return nil
}

// run 워커의 메인 루프
func (uw *UpbitWorker) run() {
	// Term(초)이 소수일 수 있으므로 밀리초로 변환하여 절삭 방지, 최소 1ms 보장

	// Upbit 워커 시작 로그 제거
	intervalMs := int64(uw.order.Term * 1000)
	if intervalMs < 1 {
		intervalMs = 1
	}
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	// 시작 로그 제거

	for {
		select {
		case <-uw.ctx.Done():
			uw.sendLog("Upbit 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			// Upbit 워커 실행 로그 제거
			// 매 tick마다 반드시 실행: 비동기 고루틴으로 처리 (이전 요청 진행 중이어도 새 요청 즉시 시작)
			go uw.executeSellOrder(uw.order.Price)
		}
	}
}

// executeSellOrder Upbit에서 매도 주문을 실행합니다
func (uw *UpbitWorker) executeSellOrder(price float64) {
	// Upbit 지정가 매도: POST /v1/orders
	// params: market, side=ask, volume, price, ord_type=limit
	uw.sendLog(fmt.Sprintf("Upbit 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, uw.order.Quantity), "order", price, uw.order.Quantity)

	params := url.Values{}
	params.Set("market", toUpbitMarket(uw.order.Symbol)) // 예: KRW-BTC 형식으로 변환
	params.Set("side", "ask")                            // 매도
	params.Set("volume", fmt.Sprintf("%.8f", uw.order.Quantity))
	params.Set("price", fmt.Sprintf("%.8f", price))
	params.Set("ord_type", "limit")

	// JWT 생성 (query_hash 포함) - 업비트 스펙에 맞춰 인코딩되지 않은 쿼리 문자열로 해시 생성
	jwtToken, err := uw.createUserJWTToken(params)
	if err != nil {
		uw.manager.SendSystemLog("UpbitWorker", "executeSellOrder", "JWT 생성 실패", "error", "", uw.order.Name, err.Error())
		return
	}
	headers, _ := uw.createHeader(jwtToken)

	// JSON 바디 구성
	body := map[string]string{
		"market":   params.Get("market"),
		"side":     params.Get("side"),
		"volume":   params.Get("volume"),
		"price":    params.Get("price"),
		"ord_type": params.Get("ord_type"),
	}
	var bufBytes []byte
	bufBytes, err = json.Marshal(body)
	if err != nil {
		uw.manager.SendSystemLog("UpbitWorker", "executeSellOrder", "바디 변환 실패", "error", "", uw.order.Name, err.Error())
		return
	}

	req, err := http.NewRequestWithContext(uw.ctx, http.MethodPost, uw.url, bytes.NewReader(bufBytes))
	if err != nil {
		uw.manager.SendSystemLog("UpbitWorker", "executeSellOrder", "요청 생성 실패", "error", "", uw.order.Name, err.Error())
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		uw.manager.SendSystemLog("UpbitWorker", "executeSellOrder", "요청 실패", "error", "", uw.order.Name, err.Error())
		return
	}
	defer resp.Body.Close()
	var respBody map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		// 상태코드만 로그
		uw.manager.SendSystemLog("UpbitWorker", "executeSellOrder", fmt.Sprintf("응답 파싱 실패 (status=%d)", resp.StatusCode), "error", "", uw.order.Name, err.Error())
		return
	}

	// 응답을 콘솔에 출력 (로그 포맷터 사용)
	uw.printExchangeResponse("Upbit", resp.StatusCode, respBody)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 실패 사유를 시스템 로그에 남김
		errMsg := fmt.Sprintf("주문 실패 (status=%d): %v", resp.StatusCode, respBody)
		// 시스템 로그
		uw.manager.SendSystemLog("UpbitWorker", "executeSellOrder", errMsg, "error", "", uw.order.Name, "")
		// 워커 로그에도 실패 메시지 추가
		uw.sendLog(errMsg, "error", price, uw.order.Quantity)
		return
	}

	// 성공 시 간단한 로그만 출력
	uw.sendLog("주문 성공", "success")
}

// printExchangeResponse 거래소 응답을 콘솔에 출력
func (uw *UpbitWorker) printExchangeResponse(exchangeName string, statusCode int, parsedResult map[string]interface{}) {
	fmt.Printf("\n=== %s API 응답 ===\n", exchangeName)
	fmt.Printf("상태 코드: %d\n", statusCode)
	fmt.Printf("파싱된 결과: %+v\n", parsedResult)
	fmt.Printf("=== %s 응답 끝 ===\n\n", exchangeName)
}

// formatLogMessage 로그 메시지 포맷팅
func (uw *UpbitWorker) formatLogMessage(messageType, message string, price, quantity float64) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	switch messageType {
	case "order":
		return fmt.Sprintf("[%s] %s | 가격: %.8f | 수량: %.8f", timestamp, message, price, quantity)
	case "success":
		return fmt.Sprintf("[%s] ✅ %s | 가격: %.8f | 수량: %.8f", timestamp, message, price, quantity)
	case "error":
		return fmt.Sprintf("[%s] ❌ %s | 가격: %.8f | 수량: %.8f", timestamp, message, price, quantity)
	case "info":
		return fmt.Sprintf("[%s] ℹ️ %s", timestamp, message)
	case "warning":
		return fmt.Sprintf("[%s] ⚠️ %s", timestamp, message)
	default:
		return fmt.Sprintf("[%s] %s", timestamp, message)
	}
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (uw *UpbitWorker) GetPlatformName() string {
	return "Upbit"
}

func (uw *UpbitWorker) createUserJWTToken(params url.Values) (string, error) {
	claims := jwt.MapClaims{
		"access_key": uw.accessKey,
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
	return token.SignedString([]byte(uw.secretKey))
}

func (uw *UpbitWorker) createHeader(jwtToken string) (map[string]string, error) {
	headers := make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("Bearer %s", jwtToken)
	headers["Content-Type"] = "application/json; charset=utf-8"
	return headers, nil
}

// UpbitAccount 전체 계좌 조회 응답 모델
type UpbitAccount struct {
	Currency            string `json:"currency"`
	Balance             string `json:"balance"`
	Locked              string `json:"locked"`
	AvgBuyPrice         string `json:"avg_buy_price"`
	AvgBuyPriceModified bool   `json:"avg_buy_price_modified"`
	UnitCurrency        string `json:"unit_currency"`
}

// toUpbitMarket 사용자 입력("BTC/KRW")을 업비트 마켓 포맷("KRW-BTC")으로 변환
func toUpbitMarket(symbol string) string {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return symbol // 포맷이 다르면 원본 반환
	}
	base := strings.TrimSpace(strings.ToUpper(parts[0]))
	quote := strings.TrimSpace(strings.ToUpper(parts[1]))
	return quote + "-" + base
}
