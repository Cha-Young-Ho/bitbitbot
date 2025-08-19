package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// BybitWorker Bybit 플랫폼용 워커
type BybitWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewBybitWorker 새로운 Bybit 워커를 생성합니다
func NewBybitWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *BybitWorker {
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

	exchange := ccxt.CreateExchange("bybit", exchangeConfig)

	// BaseWorker 생성
	baseWorker := NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase)

	return &BybitWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
	}
}

// Start 워커를 시작합니다
func (bw *BybitWorker) Start(ctx context.Context) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", bw.order.Name)
	}

	bw.ctx, bw.cancel = context.WithCancel(ctx)
	bw.isRunning = true
	bw.status.IsRunning = true

	// 워커 고루틴 시작 (Bybit 자체 run 사용)
	go bw.run()
	return nil
}

// run 워커의 메인 루프 (Bybit 전용)
func (bw *BybitWorker) run() {
	ticker := time.NewTicker(time.Duration(bw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그 제거
	// Bybit 워커 시작 로그 제거

	for {
		select {
		case <-bw.ctx.Done():
			bw.sendLog("Bybit 워커가 중지되었습니다", "info")
			// Bybit 워커 중지 로그 제거
			return
		case <-ticker.C:
			bw.executeSellOrder()
		}
	}
}

// executeSellOrder 지정가 매도 주문을 실행합니다
func (bw *BybitWorker) executeSellOrder() {
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

	// Bybit 심볼 형식으로 변환 (예: BTC/USDT -> BTC/USDT)
	bybitSymbol := bw.convertToBybitSymbol(bw.order.Symbol)

	// 디버깅을 위한 로그 추가
	bw.sendLog(fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		bybitSymbol, bw.order.Quantity, bw.order.Price), "info")

	// CCXT를 사용한 지정가 매도 주문
	orderID, err := bw.exchange.CreateLimitSellOrder(
		bybitSymbol,       // 심볼 (예: BTC/USDT)
		bw.order.Quantity, // 수량
		bw.order.Price,    // 가격
	)

	if err != nil {
		bw.mu.Lock()
		bw.status.ErrorCount++
		bw.status.LastError = err.Error()
		bw.mu.Unlock()

		bw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error")
		bw.manager.SendSystemLog("BybitWorker", "executeSellOrder",
			fmt.Sprintf("매도 주문 실패: %v", err), "error", "", bw.order.Name, err.Error())
		return
	}

	// 성공 로그
	bw.sendLog(fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		bw.order.Price, bw.order.Quantity, orderID), "success", bw.order.Price, bw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BybitWorker) GetPlatformName() string {
	return "Bybit"
}

// convertToBybitSymbol Bybit 심볼 형식으로 변환합니다
func (bw *BybitWorker) convertToBybitSymbol(symbol string) string {
	// Bybit는 CCXT 표준 형식 사용 (예: BTC/USDT)
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		bw.sendLog(fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), "warning")
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// Bybit 마켓 형식으로 변환
	bybitSymbol := base + "/" + quote // "BTC/USDT"

	bw.sendLog(fmt.Sprintf("심볼 변환: %s -> %s", symbol, bybitSymbol), "info")

	return bybitSymbol
}
