package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"time"
)

// BithumbWorker 빗썸 플랫폼용 워커
type BithumbWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewBithumbWorker 새로운 빗썸 워커를 생성합니다
func NewBithumbWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *BithumbWorker {
	return &BithumbWorker{
		BaseWorker: NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase),
		accessKey:  accessKey,
		secretKey:  secretKey,
	}
}

// Start 워커를 시작합니다
func (bw *BithumbWorker) Start(ctx context.Context) error {
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
	return nil
}

// run 워커의 메인 루프
func (bw *BithumbWorker) run() {
	// Term(초)이 소수일 수 있으므로 밀리초로 변환하여 절삭 방지, 최소 1ms 보장
	intervalMs := int64(bw.order.Term * 1000)
	if intervalMs < 1 {
		intervalMs = 1
	}
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-bw.ctx.Done():
			bw.sendLog("빗썸 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			// 매 tick마다 반드시 실행: 비동기 고루틴으로 처리 (이전 요청 진행 중이어도 새 요청 즉시 시작)
			go bw.executeSellOrder(bw.order.Price)
		}
	}
}

// executeSellOrder 빗썸에서 매도 주문을 실행합니다
func (bw *BithumbWorker) executeSellOrder(price float64) {
	// 빗썸 거래소는 아직 구현되지 않았습니다
	bw.sendLog("빗썸 거래소는 아직 구현되지 않았습니다", "warning", price, bw.order.Quantity)
	bw.manager.SendSystemLog("BithumbWorker", "executeSellOrder", "빗썸 거래소는 아직 구현되지 않았습니다", "warning", "", bw.order.Name, "")
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BithumbWorker) GetPlatformName() string {
	return "Bithumb"
}
