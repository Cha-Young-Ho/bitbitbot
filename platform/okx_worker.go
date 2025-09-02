package platform

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// OKXWorker OKX 거래소 워커
type OKXWorker struct {
	mu        sync.RWMutex
	config    *WorkerConfig
	storage   *MemoryStorage
	running   bool
	stopCh    chan struct{}
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewOKXWorker 새로운 OKX 워커를 생성합니다
func NewOKXWorker(config *WorkerConfig, storage *MemoryStorage) *OKXWorker {
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

	exchange := ccxt.CreateExchange("okx", exchangeConfig)

	return &OKXWorker{
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
func (ow *OKXWorker) Start(ctx context.Context) {
	ow.mu.Lock()
	ow.running = true
	ow.mu.Unlock()
	
	ow.storage.AddLog("info", "OKX 워커가 시작되었습니다.", ow.config.Exchange, ow.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(ow.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		ow.mu.RLock()
		if !ow.running {
			ow.mu.RUnlock()
			ow.storage.AddLog("info", "OKX 워커가 중지되었습니다.", ow.config.Exchange, ow.config.Symbol)
			return
		}
		ow.mu.RUnlock()

		select {
		case <-ctx.Done():
			ow.mu.Lock()
			ow.running = false
			ow.mu.Unlock()
			ow.storage.AddLog("info", "OKX 워커가 중지되었습니다.", ow.config.Exchange, ow.config.Symbol)
			return
		case <-ow.stopCh:
			ow.mu.Lock()
			ow.running = false
			ow.mu.Unlock()
			ow.storage.AddLog("info", "OKX 워커가 중지되었습니다.", ow.config.Exchange, ow.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			ow.mu.RLock()
			if ow.running {
				ow.mu.RUnlock()
				ow.executeSellOrder()
			} else {
				ow.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (ow *OKXWorker) Stop() {
	ow.mu.Lock()
	defer ow.mu.Unlock()
	
	if ow.running {
		ow.running = false
		close(ow.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (ow *OKXWorker) IsRunning() bool {
	ow.mu.RLock()
	defer ow.mu.RUnlock()
	return ow.running
}

// executeSellOrder OKX에서 매도 주문 실행
func (ow *OKXWorker) executeSellOrder() {
	// 실행 상태 재확인
	ow.mu.RLock()
	if !ow.running {
		ow.mu.RUnlock()
		return
	}
	ow.mu.RUnlock()
	// 거래소가 nil인 경우 에러 처리
	if ow.exchange == nil {
		ow.storage.AddLog("error", "거래소가 초기화되지 않았습니다", ow.config.Exchange, ow.config.Symbol)
		return
	}

	// OKX 심볼 형식으로 변환 (예: BTC/USDT -> BTC-USDT)
	okxSymbol := ow.convertToOKXSymbol(ow.config.Symbol)

	// 주문 시도 로그
	ow.storage.AddLog("info", fmt.Sprintf("주문 시도 - 심볼: %s, 수량: %.8f, 가격: %.2f",
		okxSymbol, ow.config.SellAmount, ow.config.SellPrice), ow.config.Exchange, ow.config.Symbol)

	// CCXT를 사용한 지정가 매도 주문
	orderID, err := ow.exchange.CreateLimitSellOrder(
		okxSymbol,            // 심볼 (예: BTC-USDT)
		ow.config.SellAmount, // 수량
		ow.config.SellPrice,  // 가격
	)

	if err != nil {
		ow.storage.AddLog("error", fmt.Sprintf("매도 주문 실패: %v", err), ow.config.Exchange, ow.config.Symbol)
		return
	}

	// 성공 로그
	ow.storage.AddLog("success", fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %.2f, 수량: %.8f, 주문ID: %s)",
		ow.config.SellPrice, ow.config.SellAmount, orderID), ow.config.Exchange, ow.config.Symbol)
}

// convertToOKXSymbol 심볼을 OKX 형식으로 변환
func (ow *OKXWorker) convertToOKXSymbol(symbol string) string {
	// BTC/USDT -> BTC-USDT
	// ETH/KRW -> ETH-KRW
	okxSymbol := strings.Replace(symbol, "/", "-", -1)

	ow.storage.AddLog("info", fmt.Sprintf("심볼 변환: %s -> %s", symbol, okxSymbol), ow.config.Exchange, ow.config.Symbol)

	return okxSymbol
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (ow *OKXWorker) GetPlatformName() string {
	return "OKX"
}
