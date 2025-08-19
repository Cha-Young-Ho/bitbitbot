package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// CoinbaseWorker Coinbase 플랫폼용 워커
type CoinbaseWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewCoinbaseWorker 새로운 Coinbase 워커를 생성합니다
func NewCoinbaseWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *CoinbaseWorker {
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

	exchange := ccxt.CreateExchange("coinbase", exchangeConfig)

	// BaseWorker 생성
	baseWorker := NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase)

	return &CoinbaseWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
	}
}

// Start 워커를 시작합니다
func (cw *CoinbaseWorker) Start(ctx context.Context) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", cw.order.Name)
	}

	cw.ctx, cw.cancel = context.WithCancel(ctx)
	cw.isRunning = true
	cw.status.IsRunning = true

	// 워커 고루틴 시작 (Coinbase 자체 run 사용)
	go cw.run()
	return nil
}

// run 워커의 메인 루프 (Coinbase 전용)
func (cw *CoinbaseWorker) run() {
	ticker := time.NewTicker(time.Duration(cw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그 제거
	// Coinbase 워커 시작 로그 제거

	for {
		select {
		case <-cw.ctx.Done():
			cw.sendLog("Coinbase 워커가 중지되었습니다", "info")
			// Coinbase 워커 중지 로그 제거
			return
		case <-ticker.C:
			cw.executeSellOrder()
		}
	}
}

// executeSellOrder 지정가 매도 주문을 실행합니다
func (cw *CoinbaseWorker) executeSellOrder() {
	// BaseWorker의 상태 업데이트
	cw.mu.Lock()
	cw.status.LastCheck = time.Now()
	cw.status.CheckCount++
	cw.mu.Unlock()

	// 거래소가 nil인 경우 에러 처리
	if cw.exchange == nil {
		cw.mu.Lock()
		cw.status.ErrorCount++
		cw.status.LastError = "거래소가 초기화되지 않았습니다"
		cw.mu.Unlock()

		cw.sendLog("거래소가 초기화되지 않았습니다", "error")
		return
	}

	// Coinbase 심볼 형식으로 변환 (예: BTC/USDT -> BTC/USDT)
	coinbaseSymbol := cw.convertToCoinbaseSymbol(cw.order.Symbol)

	// 디버깅을 위한 로그 추가
	cw.sendLog(fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		coinbaseSymbol, cw.order.Quantity, cw.order.Price), "info")

	// CCXT를 사용한 지정가 매도 주문
	orderID, err := cw.exchange.CreateLimitSellOrder(
		coinbaseSymbol,    // 심볼 (예: BTC/USDT)
		cw.order.Quantity, // 수량
		cw.order.Price,    // 가격
	)

	if err != nil {
		cw.mu.Lock()
		cw.status.ErrorCount++
		cw.status.LastError = err.Error()
		cw.mu.Unlock()

		cw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error")
		cw.manager.SendSystemLog("CoinbaseWorker", "executeSellOrder",
			fmt.Sprintf("매도 주문 실패: %v", err), "error", "", cw.order.Name, err.Error())
		return
	}

	// 성공 로그
	cw.sendLog(fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		cw.order.Price, cw.order.Quantity, orderID), "success", cw.order.Price, cw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (cw *CoinbaseWorker) GetPlatformName() string {
	return "Coinbase"
}

// convertToCoinbaseSymbol Coinbase 심볼 형식으로 변환합니다
func (cw *CoinbaseWorker) convertToCoinbaseSymbol(symbol string) string {
	// Coinbase는 CCXT 표준 형식 사용 (예: BTC/USDT)
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		cw.sendLog(fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), "warning")
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// Coinbase 마켓 형식으로 변환
	coinbaseSymbol := base + "/" + quote // "BTC/USDT"

	cw.sendLog(fmt.Sprintf("심볼 변환: %s -> %s", symbol, coinbaseSymbol), "info")

	return coinbaseSymbol
}
