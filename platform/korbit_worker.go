package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"log"
	"time"
)

// KorbitWorker Korbit 플랫폼용 워커
type KorbitWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewKorbitWorker 새로운 Korbit 워커를 생성합니다
func NewKorbitWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *KorbitWorker {
	return &KorbitWorker{
		BaseWorker: NewBaseWorker(order, manager),
		accessKey:  accessKey,
		secretKey:  secretKey,
	}
}

// Start 워커를 시작합니다
func (kw *KorbitWorker) Start(ctx context.Context) error {
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

	log.Printf("Korbit 워커 시작: %s", kw.order.Name)
	return nil
}

// run 워커의 메인 루프
func (kw *KorbitWorker) run() {
	ticker := time.NewTicker(time.Duration(kw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	kw.sendLog("Korbit 워커가 시작되었습니다", "info")
	fmt.Printf("[Korbit] 워커 시작 - 주문명: %s, 심볼: %s, 목표가: %.2f\n",
		kw.order.Name, kw.order.Symbol, kw.order.Price)

	for {
		select {
		case <-kw.ctx.Done():
			kw.sendLog("Korbit 워커가 중지되었습니다", "info")
			fmt.Printf("[Korbit] 워커 중지 - 주문명: %s\n", kw.order.Name)
			return
		case <-ticker.C:
			kw.printStatus()
		}
	}
}

// printStatus 현재 상태를 출력합니다
func (kw *KorbitWorker) printStatus() {
	kw.mu.Lock()
	kw.status.LastCheck = time.Now()
	kw.status.CheckCount++
	kw.mu.Unlock()

	fmt.Printf("[Korbit] 상태 출력 - 주문명: %s, 심볼: %s, 목표가: %.2f, 수량: %.8f, 체크횟수: %d\n",
		kw.order.Name, kw.order.Symbol, kw.order.Price, kw.order.Quantity, kw.status.CheckCount)

	kw.sendStatusLog(fmt.Sprintf("Korbit 상태 확인 - 체크횟수: %d, 목표가: %.2f, 현재가: %.2f",
		kw.status.CheckCount, kw.order.Price, kw.status.LastPrice))
}

// getCurrentPrice Korbit에서 현재 가격을 조회합니다
func (kw *KorbitWorker) getCurrentPrice() (float64, error) {
	// Korbit API를 사용하여 현재가 조회
	// TODO: 실제 Korbit API 구현
	return 0, fmt.Errorf("Korbit API가 아직 구현되지 않았습니다")
}

// executeSellOrder Korbit에서 매도 주문을 실행합니다
func (kw *KorbitWorker) executeSellOrder(price float64) {
	kw.sendLog(fmt.Sprintf("Korbit 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, kw.order.Quantity), "order", price, kw.order.Quantity)
	fmt.Printf("[Korbit] 매도 주문 실행 - 주문명: %s, 가격: %.2f, 수량: %.8f\n",
		kw.order.Name, price, kw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (kw *KorbitWorker) GetPlatformName() string {
	return "Korbit"
}
