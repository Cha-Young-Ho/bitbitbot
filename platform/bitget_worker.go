package platform

import (
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// BitgetWorker 비트겟 거래소 워커
type BitgetWorker struct {
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewBitgetWorker 새로운 비트겟 워커를 생성합니다
func NewBitgetWorker(config *WorkerConfig, storage *MemoryStorage) *BitgetWorker {
	// CCXT 거래소 인스턴스 생성
	exchangeConfig := map[string]interface{}{
		"apiKey":          config.AccessKey,
		"secret":          config.SecretKey,
		"timeout":         30000, // 30초
		"sandbox":         false, // 실제 거래
		"enableRateLimit": true,
	}

	// Password Phrase가 있으면 추가
	if config.PasswordPhrase != "" {
		exchangeConfig["password"] = config.PasswordPhrase
	}

	exchange := ccxt.CreateExchange("bitget", exchangeConfig)

	return &BitgetWorker{
		config:    config,
		storage:   storage,
		running:   false,
		stopCh:    make(chan struct{}),
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		exchange:  exchange,
	}
}

// Start 워커를 시작합니다
func (bw *BitgetWorker) Start(ctx context.Context) {
	bw.running = true
	bw.storage.AddLog("info", "비트겟 워커가 시작되었습니다.", bw.config.Exchange, bw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(bw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			bw.running = false
			bw.storage.AddLog("info", "비트겟 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		case <-bw.stopCh:
			bw.running = false
			bw.storage.AddLog("info", "비트겟 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		case <-ticker.C:
			bw.executeSellOrder()
		}
	}
}

// Stop 워커를 중지합니다
func (bw *BitgetWorker) Stop() {
	if bw.running {
		close(bw.stopCh)
		bw.running = false
	}
}

// IsRunning 워커 실행 상태 확인
func (bw *BitgetWorker) IsRunning() bool {
	return bw.running
}

// executeSellOrder 비트겟에서 매도 주문 실행
func (bw *BitgetWorker) executeSellOrder() {
	// 거래소가 nil인 경우 에러 처리
	if bw.exchange == nil {
		bw.storage.AddLog("error", "거래소가 초기화되지 않았습니다", bw.config.Exchange, bw.config.Symbol)
		return
	}

	// 비트겟 심볼 형식으로 변환 (예: BTC/USDT -> BTCUSDT)
	bitgetSymbol := bw.convertToBitgetSymbol(bw.config.Symbol)

	// 주문 시도 로그
	bw.storage.AddLog("info", fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		bitgetSymbol, bw.config.SellAmount, bw.config.SellPrice), bw.config.Exchange, bw.config.Symbol)

	// CCXT를 사용한 지정가 매도 주문
	orderID, err := bw.exchange.CreateLimitSellOrder(
		bitgetSymbol,         // 심볼 (예: BTCUSDT)
		bw.config.SellAmount, // 수량
		bw.config.SellPrice,  // 가격
	)

	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("매도 주문 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	// 성공 로그
	bw.storage.AddLog("success", fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		bw.config.SellPrice, bw.config.SellAmount, orderID), bw.config.Exchange, bw.config.Symbol)
}

// convertToBitgetSymbol 비트겟 심볼 형식으로 변환
func (bw *BitgetWorker) convertToBitgetSymbol(symbol string) string {
	// 사용자 입력: "BTC/USDT" -> 비트겟 형식: "BTCUSDT"
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		bw.storage.AddLog("warning", fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), bw.config.Exchange, bw.config.Symbol)
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// 비트겟 마켓 형식으로 변환
	bitgetSymbol := base + quote // "BTCUSDT"

	bw.storage.AddLog("info", fmt.Sprintf("심볼 변환: %s -> %s", symbol, bitgetSymbol), bw.config.Exchange, bw.config.Symbol)

	return bitgetSymbol
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BitgetWorker) GetPlatformName() string {
	return "Bitget"
}
