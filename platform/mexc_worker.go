package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// MexcWorker Mexc 플랫폼용 워커
type MexcWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewMexcWorker 새로운 Mexc 워커를 생성합니다
func NewMexcWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *MexcWorker {
	// CCXT 거래소 인스턴스 생성
	exchange := ccxt.CreateExchange("mexc", map[string]interface{}{
		"apiKey":          accessKey,
		"secret":          secretKey,
		"timeout":         30000, // 30초
		"sandbox":         false, // 실제 거래
		"enableRateLimit": true,
	})

	// BaseWorker 생성 (exchange는 nil로 전달)
	baseWorker := NewBaseWorker(order, manager)

	return &MexcWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
	}
}

// Start 워커를 시작합니다
func (mw *MexcWorker) Start(ctx context.Context) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if mw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", mw.order.Name)
	}

	mw.ctx, mw.cancel = context.WithCancel(ctx)
	mw.isRunning = true
	mw.status.IsRunning = true

	// 워커 고루틴 시작 (Mexc 자체 run 사용)
	go mw.run()

	log.Printf("Mexc 워커 시작: %s", mw.order.Name)
	return nil
}

// run 워커의 메인 루프 (Mexc 전용)
func (mw *MexcWorker) run() {
	ticker := time.NewTicker(time.Duration(mw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	mw.sendLog("Mexc 워커가 시작되었습니다", "info")
	fmt.Printf("[Mexc] 워커 시작 - 주문명: %s, 심볼: %s, 지정가: %.2f, 주기: %.1f초\n",
		mw.order.Name, mw.order.Symbol, mw.order.Price, mw.order.Term)

	for {
		select {
		case <-mw.ctx.Done():
			mw.sendLog("Mexc 워커가 중지되었습니다", "info")
			fmt.Printf("[Mexc] 워커 중지 - 주문명: %s\n", mw.order.Name)
			return
		case <-ticker.C:
			mw.executeSellOrder()
		}
	}
}

// executeSellOrder 지정가 매도 주문을 실행합니다
func (mw *MexcWorker) executeSellOrder() {
	// BaseWorker의 상태 업데이트
	mw.mu.Lock()
	mw.status.LastCheck = time.Now()
	mw.status.CheckCount++
	mw.mu.Unlock()

	// 거래소가 nil인 경우 에러 처리
	if mw.exchange == nil {
		mw.mu.Lock()
		mw.status.ErrorCount++
		mw.status.LastError = "거래소가 초기화되지 않았습니다"
		mw.mu.Unlock()

		mw.sendLog("거래소가 초기화되지 않았습니다", "error")
		return
	}

	// Mexc 심볼 형식으로 변환 (예: BTC/USDT -> BTC_USDT)
	mexcSymbol := mw.convertToMexcSymbol(mw.order.Symbol)

	// 디버깅을 위한 로그 추가
	mw.sendLog(fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		mexcSymbol, mw.order.Quantity, mw.order.Price), "info")

	// CCXT를 사용한 지정가 매도 주문
	orderID, err := mw.exchange.CreateLimitSellOrder(
		mexcSymbol,        // 심볼 (예: BTC_USDT)
		mw.order.Quantity, // 수량
		mw.order.Price,    // 가격
	)

	if err != nil {
		mw.mu.Lock()
		mw.status.ErrorCount++
		mw.status.LastError = err.Error()
		mw.mu.Unlock()

		mw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error")
		mw.manager.SendSystemLog("MexcWorker", "executeSellOrder",
			fmt.Sprintf("매도 주문 실패: %v", err), "error", "", mw.order.Name, err.Error())
		return
	}

	// 성공 로그
	mw.sendLog(fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		mw.order.Price, mw.order.Quantity, orderID), "success", mw.order.Price, mw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (mw *MexcWorker) GetPlatformName() string {
	return "Mexc"
}

// convertToMexcSymbol Mexc 심볼 형식으로 변환합니다
func (mw *MexcWorker) convertToMexcSymbol(symbol string) string {
	// 사용자 입력: "BTC/USDT" -> Mexc 형식: "BTC_USDT"
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		mw.sendLog(fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), "warning")
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// Mexc 마켓 형식으로 변환
	mexcSymbol := base + "_" + quote // "BTC_USDT"

	mw.sendLog(fmt.Sprintf("심볼 변환: %s -> %s", symbol, mexcSymbol), "info")

	return mexcSymbol
}
