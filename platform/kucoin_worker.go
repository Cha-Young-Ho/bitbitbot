package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// KucoinWorker Kucoin 플랫폼용 워커
type KucoinWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewKucoinWorker 새로운 Kucoin 워커를 생성합니다
func NewKucoinWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *KucoinWorker {
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

	exchange := ccxt.CreateExchange("kucoin", exchangeConfig)

	// BaseWorker 생성
	baseWorker := NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase)

	return &KucoinWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
	}
}

// Start 워커를 시작합니다
func (kw *KucoinWorker) Start(ctx context.Context) error {
	kw.mu.Lock()
	defer kw.mu.Unlock()

	if kw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", kw.order.Name)
	}

	kw.ctx, kw.cancel = context.WithCancel(ctx)
	kw.isRunning = true
	kw.status.IsRunning = true

	// 워커 고루틴 시작
	go kw.run()
	return nil
}

// run 워커의 메인 루프
func (kw *KucoinWorker) run() {
	// Term(초)이 소수일 수 있으므로 밀리초로 변환하여 절삭 방지, 최소 1ms 보장
	intervalMs := int64(kw.order.Term * 1000)
	if intervalMs < 1 {
		intervalMs = 1
	}
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-kw.ctx.Done():
			kw.sendLog("Kucoin 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			// 매 tick마다 반드시 실행: 비동기 고루틴으로 처리 (이전 요청 진행 중이어도 새 요청 즉시 시작)
			go kw.executeSellOrder(kw.order.Price)
		}
	}
}

// executeSellOrder Kucoin에서 매도 주문을 실행합니다
func (kw *KucoinWorker) executeSellOrder(price float64) {
	// Kucoin 지정가 매도 주문 실행
	kw.sendLog(fmt.Sprintf("Kucoin 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, kw.order.Quantity), "order", price, kw.order.Quantity)

	// 거래소가 nil인 경우 에러 처리
	if kw.exchange == nil {
		kw.manager.SendSystemLog("KucoinWorker", "executeSellOrder", "거래소가 초기화되지 않았습니다", "error", "", kw.order.Name, "")
		kw.sendLog("거래소가 초기화되지 않았습니다", "error", price, kw.order.Quantity)
		return
	}

	// Kucoin 심볼 형식으로 변환
	kucoinSymbol := kw.convertToKucoinSymbol(kw.order.Symbol)

	// CCXT를 사용하여 지정가 매도 주문 생성
	orderID, err := kw.exchange.CreateLimitSellOrder(
		kucoinSymbol,
		kw.order.Quantity,
		price,
		nil, // 옵션 없이 기본값 사용
	)
	if err != nil {
		kw.manager.SendSystemLog("KucoinWorker", "executeSellOrder", fmt.Sprintf("주문 실패: %v", err), "error", "", kw.order.Name, err.Error())
		kw.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error", price, kw.order.Quantity)
		return
	}

	// 성공 시 응답 내용 로그
	successMsg := fmt.Sprintf("주문 성공 (주문ID: %s)", orderID)
	kw.manager.SendSystemLog("KucoinWorker", "executeSellOrder", successMsg, "info", "", kw.order.Name, "")
	kw.sendLog("Kucoin 지정가 매도 주문이 접수되었습니다", "success", price, kw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (kw *KucoinWorker) GetPlatformName() string {
	return "Kucoin"
}

// convertToKucoinSymbol Kucoin 심볼 형식으로 변환합니다
func (kw *KucoinWorker) convertToKucoinSymbol(symbol string) string {
	// 사용자 입력: "BTC/USDT" -> Kucoin 형식: "BTC-USDT"
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		kw.sendLog(fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), "warning")
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// Kucoin 마켓 형식으로 변환
	kucoinSymbol := base + "-" + quote // "BTC-USDT"

	kw.sendLog(fmt.Sprintf("심볼 변환: %s -> %s", symbol, kucoinSymbol), "info")

	return kucoinSymbol
}
