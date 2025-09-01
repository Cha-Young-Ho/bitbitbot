package platform

import (
	"context"
	"fmt"
	"strings"
)

// WorkerFactory 워커 팩토리
type WorkerFactory struct{}

// NewWorkerFactory 새로운 워커 팩토리 생성
func NewWorkerFactory() *WorkerFactory {
	return &WorkerFactory{}
}

// CreateWorker 거래소별 워커 생성
func (wf *WorkerFactory) CreateWorker(config *WorkerConfig, storage *MemoryStorage) (WorkerInterface, error) {
	// 거래소 이름을 소문자로 변환
	exchange := strings.ToLower(config.Exchange)
	fmt.Printf("WorkerFactory: 거래소 '%s' -> '%s'로 변환하여 워커 생성 시작\n", config.Exchange, exchange)

	switch exchange {
	case "binance":
		fmt.Println("WorkerFactory: 바이낸스 워커 생성")
		return NewBinanceWorker(config, storage), nil
	case "bitget":
		fmt.Println("WorkerFactory: 비트겟 워커 생성")
		return NewBitgetWorker(config, storage), nil
	case "bybit":
		fmt.Println("WorkerFactory: 바이비트 워커 생성")
		return NewBybitWorker(config, storage), nil
	case "kucoin":
		fmt.Println("WorkerFactory: 쿠코인 워커 생성")
		return NewKuCoinWorker(config, storage), nil
	case "upbit":
		fmt.Println("WorkerFactory: 업비트 워커 생성")
		return NewUpbitWorker(config, storage), nil
	case "bithumb":
		fmt.Println("WorkerFactory: 빗썸 워커 생성")
		return NewBithumbWorker(config, storage), nil
	case "coinbase":
		fmt.Println("WorkerFactory: 코인베이스 워커 생성")
		return NewCoinbaseWorker(config, storage), nil
	case "huobi":
		fmt.Println("WorkerFactory: 후오비 워커 생성")
		return NewHuobiWorker(config, storage), nil
	case "mexc":
		fmt.Println("WorkerFactory: MEXC 워커 생성")
		return NewMexcWorker(config, storage), nil
	case "coinone":
		fmt.Println("WorkerFactory: 코인원 워커 생성")
		return NewCoinoneWorker(config, storage), nil
	case "korbit":
		fmt.Println("WorkerFactory: 코빗 워커 생성")
		return NewKorbitWorker(config, storage), nil
	case "gate":
		fmt.Println("WorkerFactory: Gate.io 워커 생성")
		return NewGateWorker(config, storage), nil
	case "okx":
		fmt.Println("WorkerFactory: OKX 워커 생성")
		return NewOKXWorker(config, storage), nil
	default:
		// 기존 Worker 사용 (다른 거래소들은 기존 로직 사용)
		fmt.Printf("WorkerFactory: 기본 워커 생성 (거래소: %s)\n", config.Exchange)
		return NewWorker(config, storage), nil
	}
}

// WorkerInterface 워커 인터페이스
type WorkerInterface interface {
	Start(ctx context.Context)
	Stop()
	IsRunning() bool
	GetPlatformName() string
}
