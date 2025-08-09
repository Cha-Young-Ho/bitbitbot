package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// BaseWorker 기본 워커 구현체
type BaseWorker struct {
	order     local_file.SellOrder
	manager   *WorkerManager
	status    WorkerStatus
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
}

// NewBaseWorker 새로운 기본 워커를 생성합니다
func NewBaseWorker(order local_file.SellOrder, manager *WorkerManager) *BaseWorker {
	return &BaseWorker{
		order:   order,
		manager: manager,
		status: WorkerStatus{
			IsRunning:  false,
			LastCheck:  time.Time{},
			LastPrice:  0,
			CheckCount: 0,
			ErrorCount: 0,
			LastError:  "",
		},
	}
}

// Start 워커를 시작합니다
func (bw *BaseWorker) Start(ctx context.Context) error {
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

	log.Printf("워커 시작: %s (%s)", bw.order.Name, bw.order.Platform)
	return nil
}

// Stop 워커를 중지합니다
func (bw *BaseWorker) Stop() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if !bw.isRunning {
		return nil
	}

	if bw.cancel != nil {
		bw.cancel()
	}

	bw.isRunning = false
	bw.status.IsRunning = false

	log.Printf("워커 중지: %s (%s)", bw.order.Name, bw.order.Platform)
	return nil
}

// GetStatus 워커의 현재 상태를 반환합니다
func (bw *BaseWorker) GetStatus() WorkerStatus {
	bw.mu.RLock()
	defer bw.mu.RUnlock()

	return bw.status
}

// GetOrderInfo 주문 정보를 반환합니다
func (bw *BaseWorker) GetOrderInfo() local_file.SellOrder {
	return bw.order
}

// run 워커의 메인 루프
func (bw *BaseWorker) run() {
	// Term(초)이 소수일 수 있으므로, 밀리초 단위로 변환해 절삭 문제 방지
	ticker := time.NewTicker(time.Duration(bw.order.Term*1000) * time.Millisecond)
	defer ticker.Stop()

	// 시작 로그
	bw.sendLog("워커가 시작되었습니다", "info")

	for {
		select {
		case <-bw.ctx.Done():
			bw.sendLog("워커가 중지되었습니다", "info")
			return
		case <-ticker.C:
			bw.checkPrice()
		}
	}
}

// checkPrice 가격을 확인하고 조건에 맞으면 매도 주문을 실행합니다
func (bw *BaseWorker) checkPrice() {
	bw.mu.Lock()
	bw.status.LastCheck = time.Now()
	bw.status.CheckCount++
	bw.mu.Unlock()

	// 현재 가격 조회
	currentPrice, err := bw.getCurrentPrice()
	if err != nil {
		bw.mu.Lock()
		bw.status.ErrorCount++
		bw.status.LastError = err.Error()
		bw.mu.Unlock()

		bw.sendLog(fmt.Sprintf("가격 조회 실패: %v", err), "error")
		return
	}

	bw.mu.Lock()
	bw.status.LastPrice = currentPrice
	bw.mu.Unlock()

	// 가격 로그
	bw.sendLog(fmt.Sprintf("현재 가격: %.2f", currentPrice), "price", currentPrice)

	// 목표가 도달 확인
	if currentPrice >= bw.order.Price {
		bw.executeSellOrder(currentPrice)
	}
}

// getCurrentPrice 현재 가격을 조회합니다 (플랫폼별 구현 필요)
func (bw *BaseWorker) getCurrentPrice() (float64, error) {
	// 기본 구현 - 플랫폼별로 오버라이드 필요
	return 0, fmt.Errorf("getCurrentPrice가 구현되지 않았습니다")
}

// executeSellOrder 매도 주문을 실행합니다 (플랫폼별 구현 필요)
func (bw *BaseWorker) executeSellOrder(price float64) {
	// 기본 구현 - 플랫폼별로 오버라이드 필요
	bw.sendLog(fmt.Sprintf("매도 주문 실행 (가격: %.2f, 수량: %.8f)", price, bw.order.Quantity), "order", price, bw.order.Quantity)
}

// sendLog 로그를 매니저에 전송합니다
func (bw *BaseWorker) sendLog(message string, logType string, args ...float64) {
	workerLog := WorkerLog{
		OrderName:   bw.order.Name,
		Platform:    bw.order.Platform,
		Symbol:      bw.order.Symbol,
		Message:     message,
		LogType:     logType,
		Timestamp:   time.Now(),
		TargetPrice: bw.order.Price,
		Quantity:    bw.order.Quantity,
		CheckCount:  bw.status.CheckCount,
		ErrorCount:  bw.status.ErrorCount,
		LastPrice:   bw.status.LastPrice,
		OrderStatus: "running",
	}

	if len(args) > 0 {
		workerLog.Price = args[0]
	}
	if len(args) > 1 {
		workerLog.Quantity = args[1]
	}

	bw.manager.SendLog(workerLog)
}

// sendStatusLog 상태 정보를 포함한 로그를 전송합니다
func (bw *BaseWorker) sendStatusLog(message string) {
	workerLog := WorkerLog{
		OrderName:   bw.order.Name,
		Platform:    bw.order.Platform,
		Symbol:      bw.order.Symbol,
		Message:     message,
		LogType:     "status",
		Timestamp:   time.Now(),
		TargetPrice: bw.order.Price,
		Quantity:    bw.order.Quantity,
		CheckCount:  bw.status.CheckCount,
		ErrorCount:  bw.status.ErrorCount,
		LastPrice:   bw.status.LastPrice,
		OrderStatus: "running",
	}

	log.Printf("sendStatusLog 호출: %s - %s", bw.order.Name, message)
	bw.manager.SendLog(workerLog)
}

// IsRunning 워커가 실행 중인지 확인합니다
func (bw *BaseWorker) IsRunning() bool {
	bw.mu.RLock()
	defer bw.mu.RUnlock()

	return bw.isRunning
}
