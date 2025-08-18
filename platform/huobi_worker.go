package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"time"
)

// HuobiWorker Huobi 플랫폼용 워커
type HuobiWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewHuobiWorker 새로운 Huobi 워커를 생성합니다
func NewHuobiWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *HuobiWorker {
	return &HuobiWorker{
		BaseWorker: NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase),
		accessKey:  accessKey,
		secretKey:  secretKey,
	}
}

// Start 워커를 시작합니다
func (hw *HuobiWorker) Start(ctx context.Context) error {
	hw.mu.Lock()
	defer hw.mu.Unlock()

	if hw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", hw.order.Name)
	}

	hw.ctx, hw.cancel = context.WithCancel(ctx)
	hw.isRunning = true
	hw.status.IsRunning = true

	// 워커 고루틴 시작
	go hw.run()
	return nil
}

// run 워커의 메인 루프
func (hw *HuobiWorker) run() {
	ticker := time.NewTicker(time.Duration(hw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	hw.sendLog("Huobi 워커가 시작되었습니다", "info")
	fmt.Printf("[Huobi] 워커 시작 - 주문명: %s, 심볼: %s, 목표가: %.2f\n",
		hw.order.Name, hw.order.Symbol, hw.order.Price)

	for {
		select {
		case <-hw.ctx.Done():
			hw.sendLog("Huobi 워커가 중지되었습니다", "info")
			fmt.Printf("[Huobi] 워커 중지 - 주문명: %s\n", hw.order.Name)
			return
		case <-ticker.C:
			hw.printStatus()
		}
	}
}

// printStatus 현재 상태를 출력합니다
func (hw *HuobiWorker) printStatus() {
	hw.mu.Lock()
	hw.status.LastCheck = time.Now()
	hw.status.CheckCount++
	hw.mu.Unlock()

	fmt.Printf("[Huobi] 상태 출력 - 주문명: %s, 심볼: %s, 목표가: %.2f, 수량: %.8f, 체크횟수: %d\n",
		hw.order.Name, hw.order.Symbol, hw.order.Price, hw.order.Quantity, hw.status.CheckCount)

	hw.sendStatusLog(fmt.Sprintf("Huobi 상태 확인 - 체크횟수: %d, 목표가: %.2f, 현재가: %.2f",
		hw.status.CheckCount, hw.order.Price, hw.status.LastPrice))
}

// getCurrentPrice Huobi에서 현재 가격을 조회합니다
func (hw *HuobiWorker) getCurrentPrice() (float64, error) {
	// Huobi API를 사용하여 현재가 조회
	// TODO: 실제 Huobi API 구현
	return 0, fmt.Errorf("Huobi API가 아직 구현되지 않았습니다")
}

// executeSellOrder Huobi에서 매도 주문을 실행합니다
func (hw *HuobiWorker) executeSellOrder(price float64) {
	hw.sendLog(fmt.Sprintf("Huobi 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, hw.order.Quantity), "order", price, hw.order.Quantity)
	fmt.Printf("[Huobi] 매도 주문 실행 - 주문명: %s, 가격: %.2f, 수량: %.8f\n",
		hw.order.Name, price, hw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (hw *HuobiWorker) GetPlatformName() string {
	return "Huobi"
}
