package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"log"
	"time"
)

// BybitWorker Bybit 플랫폼용 워커
type BybitWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewBybitWorker 새로운 Bybit 워커를 생성합니다
func NewBybitWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *BybitWorker {
	return &BybitWorker{
		BaseWorker: NewBaseWorker(order, manager),
		accessKey:  accessKey,
		secretKey:  secretKey,
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

	// 워커 고루틴 시작
	go bw.run()

	log.Printf("Bybit 워커 시작: %s", bw.order.Name)
	return nil
}

// run 워커의 메인 루프
func (bw *BybitWorker) run() {
	ticker := time.NewTicker(time.Duration(bw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	bw.sendLog("Bybit 워커가 시작되었습니다", "info")
	fmt.Printf("[Bybit] 워커 시작 - 주문명: %s, 심볼: %s, 목표가: %.2f\n",
		bw.order.Name, bw.order.Symbol, bw.order.Price)

	for {
		select {
		case <-bw.ctx.Done():
			bw.sendLog("Bybit 워커가 중지되었습니다", "info")
			fmt.Printf("[Bybit] 워커 중지 - 주문명: %s\n", bw.order.Name)
			return
		case <-ticker.C:
			bw.printStatus()
		}
	}
}

// printStatus 현재 상태를 출력합니다
func (bw *BybitWorker) printStatus() {
	bw.mu.Lock()
	bw.status.LastCheck = time.Now()
	bw.status.CheckCount++
	bw.mu.Unlock()

	fmt.Printf("[Bybit] 상태 출력 - 주문명: %s, 심볼: %s, 목표가: %.2f, 수량: %.8f, 체크횟수: %d\n",
		bw.order.Name, bw.order.Symbol, bw.order.Price, bw.order.Quantity, bw.status.CheckCount)

	bw.sendStatusLog(fmt.Sprintf("Bybit 상태 확인 - 체크횟수: %d, 목표가: %.2f, 현재가: %.2f",
		bw.status.CheckCount, bw.order.Price, bw.status.LastPrice))
}

// getCurrentPrice Bybit에서 현재 가격을 조회합니다
func (bw *BybitWorker) getCurrentPrice() (float64, error) {
	// Bybit API를 사용하여 현재가 조회
	// TODO: 실제 Bybit API 구현
	return 0, fmt.Errorf("Bybit API가 아직 구현되지 않았습니다")
}

// executeSellOrder Bybit에서 매도 주문을 실행합니다
func (bw *BybitWorker) executeSellOrder(price float64) {
	bw.sendLog(fmt.Sprintf("Bybit 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, bw.order.Quantity), "order", price, bw.order.Quantity)
	fmt.Printf("[Bybit] 매도 주문 실행 - 주문명: %s, 가격: %.2f, 수량: %.8f\n",
		bw.order.Name, price, bw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BybitWorker) GetPlatformName() string {
	return "Bybit"
}
