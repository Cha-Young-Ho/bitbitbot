package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// GateWorker Gate 플랫폼용 워커
type GateWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewGateWorker 새로운 Gate 워커를 생성합니다
func NewGateWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *GateWorker {
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

	exchange := ccxt.CreateExchange("gate", exchangeConfig)

	// BaseWorker 생성
	baseWorker := NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase)

	return &GateWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
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
			gw.sendLog("Gate 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			// 매 tick마다 반드시 실행: 비동기 고루틴으로 처리 (이전 요청 진행 중이어도 새 요청 즉시 시작)
			go gw.executeSellOrder(gw.order.Price)
		}
	}
}

// executeSellOrder Gate에서 매도 주문을 실행합니다
func (gw *GateWorker) executeSellOrder(price float64) {
	// Gate 지정가 매도 주문 실행
	gw.sendLog(fmt.Sprintf("Gate 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, gw.order.Quantity), "order", price, gw.order.Quantity)

	// 거래소가 nil인 경우 에러 처리
	if gw.exchange == nil {
		gw.manager.SendSystemLog("GateWorker", "executeSellOrder", "거래소가 초기화되지 않았습니다", "error", "", gw.order.Name, "")
		gw.sendLog("거래소가 초기화되지 않았습니다", "error", price, gw.order.Quantity)
		return
	}

	// Gate 심볼 형식으로 변환
	gateSymbol := gw.convertToGateSymbol(gw.order.Symbol)

	// CCXT를 사용하여 지정가 매도 주문 생성
	orderID, err := gw.exchange.CreateLimitSellOrder(
		gateSymbol,
		gw.order.Quantity,
		price,
		nil, // 옵션 없이 기본값 사용
	)
	if err != nil {
		gw.manager.SendSystemLog("GateWorker", "executeSellOrder", fmt.Sprintf("주문 실패: %v", err), "error", "", gw.order.Name, err.Error())
		gw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error", price, gw.order.Quantity)
		return
	}

	// 성공 시 응답 내용 로그
	successMsg := fmt.Sprintf("주문 성공 (주문ID: %s)", orderID)
	gw.manager.SendSystemLog("GateWorker", "executeSellOrder", successMsg, "info", "", gw.order.Name, "")
	gw.sendLog("Gate 지정가 매도 주문이 접수되었습니다", "success", price, gw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (gw *GateWorker) GetPlatformName() string {
	return "Gate"
}

// convertToGateSymbol Gate 심볼 형식으로 변환합니다
func (gw *GateWorker) convertToGateSymbol(symbol string) string {
	// 사용자 입력: "BTC/USDT" -> Gate 형식: "BTC_USDT"
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		gw.sendLog(fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), "warning")
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// Gate 마켓 형식으로 변환
	gateSymbol := base + "_" + quote // "BTC_USDT"

	gw.sendLog(fmt.Sprintf("심볼 변환: %s -> %s", symbol, gateSymbol), "info")

	return gateSymbol
}
