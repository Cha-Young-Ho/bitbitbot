package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"log"
	"time"
)

// UpbitWorker Upbit 플랫폼용 워커
type UpbitWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewUpbitWorker 새로운 Upbit 워커를 생성합니다
func NewUpbitWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *UpbitWorker {
	return &UpbitWorker{
		BaseWorker: NewBaseWorker(order, manager),
		accessKey:  accessKey,
		secretKey:  secretKey,
	}
}

// getCurrentPrice Upbit에서 현재 가격을 조회합니다
func (uw *UpbitWorker) getCurrentPrice() (float64, error) {
	// Upbit API를 사용하여 현재가 조회
	// TODO: 실제 Upbit API 구현
	return 0, fmt.Errorf("Upbit API가 아직 구현되지 않았습니다")
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

	log.Printf("Upbit 워커 시작: %s", uw.order.Name)
	return nil
}

// run 워커의 메인 루프
func (uw *UpbitWorker) run() {
	ticker := time.NewTicker(time.Duration(uw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	uw.sendLog("Upbit 워커가 시작되었습니다", "info")
	fmt.Printf("[Upbit] 워커 시작 - 주문명: %s, 심볼: %s, 목표가: %.2f\n",
		uw.order.Name, uw.order.Symbol, uw.order.Price)

	for {
		select {
		case <-uw.ctx.Done():
			uw.sendLog("Upbit 워커가 중지되었습니다", "info")
			fmt.Printf("[Upbit] 워커 중지 - 주문명: %s\n", uw.order.Name)
			return
		case <-ticker.C:
			uw.printStatus()
		}
	}
}

// printStatus 현재 상태를 출력합니다
func (uw *UpbitWorker) printStatus() {
	uw.mu.Lock()
	uw.status.LastCheck = time.Now()
	uw.status.CheckCount++
	uw.mu.Unlock()

	fmt.Printf("[Upbit] 상태 출력 - 주문명: %s, 심볼: %s, 목표가: %.2f, 수량: %.8f, 체크횟수: %d\n",
		uw.order.Name, uw.order.Symbol, uw.order.Price, uw.order.Quantity, uw.status.CheckCount)

	uw.sendStatusLog(fmt.Sprintf("Upbit 상태 확인 - 체크횟수: %d, 목표가: %.2f, 현재가: %.2f",
		uw.status.CheckCount, uw.order.Price, uw.status.LastPrice))
}

// executeSellOrder Upbit에서 매도 주문을 실행합니다
func (uw *UpbitWorker) executeSellOrder(price float64) {
	uw.sendLog(fmt.Sprintf("Upbit 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, uw.order.Quantity), "order", price, uw.order.Quantity)
	fmt.Printf("[Upbit] 매도 주문 실행 - 주문명: %s, 가격: %.2f, 수량: %.8f\n",
		uw.order.Name, price, uw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (uw *UpbitWorker) GetPlatformName() string {
	return "Upbit"
}
