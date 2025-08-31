package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"time"
)

// MexcWorker Mexc 플랫폼용 워커
type MexcWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
}

// NewMexcWorker 새로운 Mexc 워커를 생성합니다
func NewMexcWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *MexcWorker {
	return &MexcWorker{
		BaseWorker: NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase),
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
	return nil
}

// run 워커의 메인 루프
func (mw *MexcWorker) run() {
	// Term(초)이 소수일 수 있으므로 밀리초로 변환하여 절삭 방지, 최소 1ms 보장
	intervalMs := int64(mw.order.Term * 1000)
	if intervalMs < 1 {
		intervalMs = 1
	}
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-mw.ctx.Done():
			mw.sendLog("MEXC 워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			// 매 tick마다 반드시 실행: 비동기 고루틴으로 처리 (이전 요청 진행 중이어도 새 요청 즉시 시작)
			go mw.executeSellOrder(mw.order.Price)
		}
	}
}

// executeSellOrder MEXC에서 매도 주문을 실행합니다
func (mw *MexcWorker) executeSellOrder(price float64) {
	// MEXC 거래소는 아직 구현되지 않았습니다
	mw.sendLog("MEXC 거래소는 아직 구현되지 않았습니다", "warning", price, mw.order.Quantity)
	mw.manager.SendSystemLog("MexcWorker", "executeSellOrder", "MEXC 거래소는 아직 구현되지 않았습니다", "warning", "", mw.order.Name, "")
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (mw *MexcWorker) GetPlatformName() string {
	return "Mexc"
}
