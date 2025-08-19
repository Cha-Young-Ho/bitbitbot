package platform

import (
	"bitbit-app/local_file"
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// OKXWorker OKX 플랫폼용 워커
type OKXWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewOKXWorker 새로운 OKX 워커를 생성합니다
func NewOKXWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *OKXWorker {
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

	exchange := ccxt.CreateExchange("okx", exchangeConfig)

	// BaseWorker 생성
	baseWorker := NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase)

	return &OKXWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
	}
}

// Start 워커를 시작합니다
func (ow *OKXWorker) Start(ctx context.Context) error {
	ow.mu.Lock()
	defer ow.mu.Unlock()

	if ow.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", ow.order.Name)
	}

	ow.ctx, ow.cancel = context.WithCancel(ctx)
	ow.isRunning = true
	ow.status.IsRunning = true

	// 워커 고루틴 시작 (OKX 자체 run 사용)
	go ow.run()
	return nil
}

// run 워커의 메인 루프 (OKX 전용)
func (ow *OKXWorker) run() {
	ticker := time.NewTicker(time.Duration(ow.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그 제거
	// OKX 워커 시작 로그 제거

	for {
		select {
		case <-ow.ctx.Done():
			ow.sendLog("OKX 워커가 중지되었습니다", "info")
			// OKX 워커 중지 로그 제거
			return
		case <-ticker.C:
			ow.executeSellOrder()
		}
	}
}

// executeSellOrder 지정가 매도 주문을 실행합니다
func (ow *OKXWorker) executeSellOrder() {
	// BaseWorker의 상태 업데이트
	ow.mu.Lock()
	ow.status.LastCheck = time.Now()
	ow.status.CheckCount++
	ow.mu.Unlock()

	// 거래소가 nil인 경우 에러 처리
	if ow.exchange == nil {
		ow.mu.Lock()
		ow.status.ErrorCount++
		ow.status.LastError = "거래소가 초기화되지 않았습니다"
		ow.mu.Unlock()

		ow.sendLog("거래소가 초기화되지 않았습니다", "error")
		return
	}

	// OKX 심볼 형식으로 변환 (예: BTC/USDT -> BTC-USDT)
	okxSymbol := ow.convertToOKXSymbol(ow.order.Symbol)

	// 디버깅을 위한 로그 추가
	ow.sendLog(fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		okxSymbol, ow.order.Quantity, ow.order.Price), "info")

	// CCXT를 사용한 지정가 매도 주문
	orderID, err := ow.exchange.CreateLimitSellOrder(
		okxSymbol,         // 심볼 (예: BTC-USDT)
		ow.order.Quantity, // 수량
		ow.order.Price,    // 가격
	)

	if err != nil {
		ow.mu.Lock()
		ow.status.ErrorCount++
		ow.status.LastError = err.Error()
		ow.mu.Unlock()

		ow.sendLog(fmt.Sprintf("매도 주문 실패: %v", err), "error")
		// OKX 매도 주문 실패 로그 제거
		return
	}

	// 성공 로그
	ow.sendLog(fmt.Sprintf("매도 주문 성공 - 주문ID: %s, 심볼: %s, 수량: %.8f, 가격: %.2f",
		orderID, okxSymbol, ow.order.Quantity, ow.order.Price), "order", ow.order.Price, ow.order.Quantity)
	// OKX 매도 주문 성공 로그 제거

	// 워커 중지 (주문 완료)
	ow.Stop()
}

// convertToOKXSymbol 심볼을 OKX 형식으로 변환합니다
func (ow *OKXWorker) convertToOKXSymbol(symbol string) string {
	// BTC/USDT -> BTC-USDT
	// ETH/KRW -> ETH-KRW
	return strings.Replace(symbol, "/", "-", -1)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (ow *OKXWorker) GetPlatformName() string {
	return "OKX"
}
