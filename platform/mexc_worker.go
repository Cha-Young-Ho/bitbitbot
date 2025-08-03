package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"log"
	"time"
)

// MexcWorker Mexc 플랫폼용 워커
type MexcWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewMexcWorker 새로운 Mexc 워커를 생성합니다
func NewMexcWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey string) *MexcWorker {
	return &MexcWorker{
		BaseWorker: NewBaseWorker(order, manager),
		accessKey:  accessKey,
		secretKey:  secretKey,
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

	// 워커 고루틴 시작
	go mw.run()

	log.Printf("Mexc 워커 시작: %s", mw.order.Name)
	return nil
}

// run 워커의 메인 루프
func (mw *MexcWorker) run() {
	ticker := time.NewTicker(time.Duration(mw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그
	mw.sendLog("Mexc 워커가 시작되었습니다", "info")
	fmt.Printf("[Mexc] 워커 시작 - 주문명: %s, 심볼: %s, 목표가: %.2f\n",
		mw.order.Name, mw.order.Symbol, mw.order.Price)

	for {
		select {
		case <-mw.ctx.Done():
			mw.sendLog("Mexc 워커가 중지되었습니다", "info")
			fmt.Printf("[Mexc] 워커 중지 - 주문명: %s\n", mw.order.Name)
			return
		case <-ticker.C:
			mw.printStatus()
		}
	}
}

// printStatus 현재 상태를 출력합니다
func (mw *MexcWorker) printStatus() {
	mw.mu.Lock()
	mw.status.LastCheck = time.Now()
	mw.status.CheckCount++
	mw.mu.Unlock()

	fmt.Printf("[Mexc] 상태 출력 - 주문명: %s, 심볼: %s, 목표가: %.2f, 수량: %.8f, 체크횟수: %d\n",
		mw.order.Name, mw.order.Symbol, mw.order.Price, mw.order.Quantity, mw.status.CheckCount)

	mw.sendStatusLog(fmt.Sprintf("Mexc 상태 확인 - 체크횟수: %d, 목표가: %.2f, 현재가: %.2f",
		mw.status.CheckCount, mw.order.Price, mw.status.LastPrice))
}

// getCurrentPrice Mexc에서 현재 가격을 조회합니다
func (mw *MexcWorker) getCurrentPrice() (float64, error) {
	// Mexc API를 사용하여 현재가 조회
	// TODO: 실제 Mexc API 구현
	return 0, fmt.Errorf("Mexc API가 아직 구현되지 않았습니다")
}

// executeSellOrder Mexc에서 매도 주문을 실행합니다
func (mw *MexcWorker) executeSellOrder(price float64) {
	mw.sendLog(fmt.Sprintf("Mexc 매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, mw.order.Quantity), "order", price, mw.order.Quantity)
	fmt.Printf("[Mexc] 매도 주문 실행 - 주문명: %s, 가격: %.2f, 수량: %.8f\n",
		mw.order.Name, price, mw.order.Quantity)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (mw *MexcWorker) GetPlatformName() string {
	return "Mexc"
}
