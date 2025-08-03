package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"log"
	"time"
)

// CoinoneWorker Coinone 플랫폼용 워커
type CoinoneWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewCoinoneWorker 새로운 Coinone 워커를 생성합니다
func NewCoinoneWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *CoinoneWorker {
	return &CoinoneWorker{
		BaseWorker: NewBaseWorker(order, manager),
		accessKey:  accessKey,
		secretKey:  secretKey,
	}
}

// Start 워커를 시작합니다
func (cw *CoinoneWorker) Start(ctx context.Context) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", cw.order.Name)
	}

	cw.ctx, cw.cancel = context.WithCancel(ctx)
	cw.isRunning = true
	cw.status.IsRunning = true

	// 워커 고루틴 시작
	go cw.run()

	log.Printf("Coinone 워커 시작: %s", cw.order.Name)
	return nil
}

// run 워커의 메인 루프
func (cw *CoinoneWorker) run() {
	ticker := time.NewTicker(time.Duration(cw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	cw.sendLog("Coinone 워커가 시작되었습니다", "info")
	fmt.Printf("[Coinone] 워커 시작 - 주문명: %s, 심볼: %s, 목표가: %.2f\n",
		cw.order.Name, cw.order.Symbol, cw.order.Price)

	for {
		select {
		case <-cw.ctx.Done():
			cw.sendLog("Coinone 워커가 중지되었습니다", "info")
			fmt.Printf("[Coinone] 워커 중지 - 주문명: %s\n", cw.order.Name)
			return
		case <-ticker.C:
			cw.printStatus()
		}
	}
}

// printStatus 현재 상태를 출력합니다
func (cw *CoinoneWorker) printStatus() {
	cw.mu.Lock()
	cw.status.LastCheck = time.Now()
	cw.status.CheckCount++
	cw.mu.Unlock()

	fmt.Printf("[Coinone] 상태 출력 - 주문명: %s, 심볼: %s, 목표가: %.2f, 수량: %.8f, 체크횟수: %d\n",
		cw.order.Name, cw.order.Symbol, cw.order.Price, cw.order.Quantity, cw.status.CheckCount)

	cw.sendStatusLog(fmt.Sprintf("Coinone 상태 확인 - 체크횟수: %d, 목표가: %.2f, 현재가: %.2f",
		cw.status.CheckCount, cw.order.Price, cw.status.LastPrice))
}

// getCurrentPrice Coinone에서 현재 가격을 조회합니다
func (cw *CoinoneWorker) getCurrentPrice() (float64, error) {
	// Coinone API를 사용하여 현재가 조회
	// TODO: 실제 Coinone API 구현
	return 0, fmt.Errorf("Coinone API가 아직 구현되지 않았습니다")
}

// executeSellOrder Coinone에서 매도 주문을 실행합니다
func (cw *CoinoneWorker) executeSellOrder(price float64) {
	cw.sendLog(fmt.Sprintf("Coinone 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, cw.order.Quantity), "order", price, cw.order.Quantity)
	fmt.Printf("[Coinone] 매도 주문 실행 - 주문명: %s, 가격: %.2f, 수량: %.8f\n",
		cw.order.Name, price, cw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (cw *CoinoneWorker) GetPlatformName() string {
	return "Coinone"
}
