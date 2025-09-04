package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// BinanceWorker Binance 플랫폼용 워커
type BinanceWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewBinanceWorker 새로운 Binance 워커를 생성합니다
func NewBinanceWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *BinanceWorker {
	// CCXT 거래소 인스턴스 생성
	exchangeConfig := map[string]interface{}{
		"apiKey":          accessKey,
		"secret":          secretKey,
		"timeout":         30000, // 30초
		"sandbox":         false, // 실제 거래
		"enableRateLimit": true,
	}

	// Password Phrase가 있으면 추가
	if passwordPhrase != "" {
		exchangeConfig["password"] = passwordPhrase
	}

	exchange := ccxt.CreateExchange("binance", exchangeConfig)

	// BaseWorker 생성
	baseWorker := NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase)

	return &BinanceWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
	}
}

// Start 워커를 시작합니다
func (bw *BinanceWorker) Start(ctx context.Context) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", bw.order.Name)
	}

	bw.ctx, bw.cancel = context.WithCancel(ctx)
	bw.isRunning = true
	bw.status.IsRunning = true

	// 워커 고루틴 시작 (바이낸스 자체 run 사용)
	go bw.run()
	return nil
}

// run 워커의 메인 루프 (바이낸스 전용)
func (bw *BinanceWorker) run() {
	ticker := time.NewTicker(time.Duration(bw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그 제거
	// Binance 워커 시작 로그 제거

	for {
		select {
		case <-bw.ctx.Done():
			bw.sendLog("Binance 워커가 중지되었습니다", "info")
			// Binance 워커 중지 로그 제거
			return
		case <-ticker.C:
			bw.executeSellOrder()
		}
	}
}

// executeSellOrder 지정가 매도 주문을 실행합니다
func (bw *BinanceWorker) executeSellOrder() {
	// BaseWorker의 상태 업데이트
	bw.mu.Lock()
	bw.status.LastCheck = time.Now()
	bw.status.CheckCount++
	bw.mu.Unlock()

	// 거래소가 nil인 경우 에러 처리
	if bw.exchange == nil {
		bw.mu.Lock()
		bw.status.ErrorCount++
		bw.status.LastError = "거래소가 초기화되지 않았습니다"
		bw.mu.Unlock()

		bw.sendLog("거래소가 초기화되지 않았습니다", "error")
		return
	}

	// 바이낸스 심볼 형식으로 변환 (예: BTC/USDT -> BTCUSDT)
	binanceSymbol := bw.convertToBinanceSymbol(bw.order.Symbol)

	// 디버깅을 위한 로그 추가
	bw.sendLog(fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		binanceSymbol, bw.order.Quantity, bw.order.Price), "info")

	// CCXT를 사용한 지정가 매도 주문
	orderID, err := bw.exchange.CreateLimitSellOrder(
		binanceSymbol,     // 심볼 (예: BTCUSDT)
		bw.order.Quantity, // 수량
		bw.order.Price,    // 가격
	)

	if err != nil {
		bw.mu.Lock()
		bw.status.ErrorCount++
		bw.status.LastError = err.Error()
		bw.mu.Unlock()

		// 에러 응답을 콘솔에 출력
		bw.printOrderResult("Binance", false, "", err.Error())

		bw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error")
		bw.manager.SendSystemLog("BinanceWorker", "executeSellOrder",
			fmt.Sprintf("매도 주문 실패: %v", err), "error", "", bw.order.Name, err.Error())
		return
	}

	// 성공 응답을 콘솔에 출력
	orderIDStr := ""
	if orderID.Id != nil {
		orderIDStr = *orderID.Id
	}
	bw.printOrderResult("Binance", true, orderIDStr, "")

	// 성공 시 간단한 로그만 출력
	bw.sendLog("주문 성공", "success", bw.order.Price, bw.order.Quantity)
}

// printOrderResult 주문 결과를 콘솔에 출력
func (bw *BinanceWorker) printOrderResult(exchangeName string, success bool, orderID, errorMsg string) {
	fmt.Printf("\n=== %s 주문 결과 ===\n", exchangeName)
	fmt.Printf("성공 여부: %t\n", success)
	if success {
		fmt.Printf("주문 ID: %s\n", orderID)
		fmt.Printf("가격: %.8f\n", bw.order.Price)
		fmt.Printf("수량: %.8f\n", bw.order.Quantity)
		fmt.Printf("총 금액: %.8f\n", bw.order.Price*bw.order.Quantity)
	} else {
		fmt.Printf("에러 메시지: %s\n", errorMsg)
	}
	fmt.Printf("=== %s 주문 결과 끝 ===\n\n", exchangeName)
}

// formatLogMessage 로그 메시지 포맷팅
func (bw *BinanceWorker) formatLogMessage(messageType, message string, price, quantity float64) string {
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
func (bw *BinanceWorker) GetPlatformName() string {
	return "Binance"
}

// convertToBinanceSymbol 바이낸스 심볼 형식으로 변환합니다
func (bw *BinanceWorker) convertToBinanceSymbol(symbol string) string {
	// 사용자 입력: "BTC/USDT" -> 바이낸스 형식: "BTCUSDT"
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		bw.sendLog(fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), "warning")
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// 바이낸스 마켓 형식으로 변환
	binanceSymbol := base + quote // "BTCUSDT"

	bw.sendLog(fmt.Sprintf("심볼 변환: %s -> %s", symbol, binanceSymbol), "info")

	return binanceSymbol
}
