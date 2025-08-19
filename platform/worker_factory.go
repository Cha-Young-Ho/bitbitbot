package platform

import (
	"bitbit-app/local_file"
	"fmt"
	"strings"
)

// WorkerFactory 워커를 생성하는 팩토리
type WorkerFactory struct {
	manager *WorkerManager
}

// NewWorkerFactory 새로운 워커 팩토리를 생성합니다
func NewWorkerFactory(manager *WorkerManager) *WorkerFactory {
	return &WorkerFactory{
		manager: manager,
	}
}

// CreateWorker 플랫폼에 따라 적절한 워커를 생성합니다
func (wf *WorkerFactory) CreateWorker(order local_file.SellOrder, accessKey, secretKey, passwordPhrase string) (Worker, error) {
	platform := strings.ToLower(order.Platform)

	switch platform {
	case "upbit":
		return NewUpbitWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "bithumb":
		return NewBithumbWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "binance":
		return NewBinanceWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "bybit":
		return NewBybitWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "bitget":
		return NewBitgetWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "huobi":
		return NewHuobiWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "mexc":
		return NewMexcWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "kucoin":
		return NewKuCoinWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "coinbase":
		return NewCoinbaseWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "coinone":
		return NewCoinoneWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "korbit":
		return NewKorbitWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	case "okx":
		return NewOKXWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	default:
		// 아직 구현되지 않은 플랫폼은 기본 워커로 대체
		fmt.Printf("지원되지 않는 플랫폼: %s, 기본 워커로 대체\n", platform)
		return NewBaseWorker(order, wf.manager, accessKey, secretKey, passwordPhrase), nil
	}
}

// GetSupportedPlatforms 지원되는 플랫폼 목록을 반환합니다
func (wf *WorkerFactory) GetSupportedPlatforms() []string {
	return []string{
		"Upbit",
		"Bithumb",
		"Binance",
		"Bybit",
		"Bitget",
		"Huobi",
		"Mexc",
		"KuCoin",
		"Coinbase",
		"Coinone",
		"Korbit",
		"OKX",
	}
}
