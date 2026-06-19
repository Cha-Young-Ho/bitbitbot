package platform

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BinanceWorker 바이낸스 거래소 워커
type BinanceWorker struct {
	config             *WorkerConfig
	storage            *MemoryStorage
	running            bool
	stopCh             chan struct{}
	accessKey          string
	secretKey          string
	url                string
	mu                 sync.RWMutex
	lastSuccessOrderID string
	symbolInfoCache    map[string]*BinanceSymbolInfo
	symbolInfoMu       sync.RWMutex
}

// BinanceSymbolInfo 심볼 정밀도/규칙 정보
type BinanceSymbolInfo struct {
	Symbol           string
	PricePrecision   int
	AmountPrecision  int
	MinQty           string
	MinPrice         string
}

// NewBinanceWorker 새로운 바이낸스 워커를 생성합니다
func NewBinanceWorker(config *WorkerConfig, storage *MemoryStorage) *BinanceWorker {
	return &BinanceWorker{
		config:          config,
		storage:         storage,
		running:         false,
		stopCh:          make(chan struct{}),
		accessKey:       config.AccessKey,
		secretKey:       config.SecretKey,
		url:             "https://api.binance.com/api/v3/order",
		symbolInfoCache: make(map[string]*BinanceSymbolInfo),
	}
}

// Start 워커를 시작합니다
func (bw *BinanceWorker) Start(ctx context.Context) {
	bw.mu.Lock()
	bw.running = true
	bw.mu.Unlock()

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(bw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		bw.mu.RLock()
		if !bw.running {
			bw.mu.RUnlock()
			bw.storage.AddLog("info", "바이낸스 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		}
		bw.mu.RUnlock()

		select {
		case <-ctx.Done():
			bw.mu.Lock()
			bw.running = false
			bw.mu.Unlock()
			bw.storage.AddLog("info", "바이낸스 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		case <-bw.stopCh:
			bw.mu.Lock()
			bw.running = false
			bw.mu.Unlock()
			bw.storage.AddLog("info", "바이낸스 워커가 중지되었습니다.", bw.config.Exchange, bw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			bw.mu.RLock()
			if bw.running {
				bw.mu.RUnlock()
				bw.executeSellOrder()
			} else {
				bw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (bw *BinanceWorker) Stop() {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.running {
		bw.running = false
		close(bw.stopCh)
	}
}

// IsRunning 워커 실행 상태 확인
func (bw *BinanceWorker) IsRunning() bool {
	bw.mu.RLock()
	defer bw.mu.RUnlock()
	return bw.running
}

// executeSellOrder 바이낸스에서 매도 주문 실행
func (bw *BinanceWorker) executeSellOrder() {
	// 실행 상태 재확인
	bw.mu.RLock()
	if !bw.running {
		bw.mu.RUnlock()
		return
	}
	bw.mu.RUnlock()

	timestamp := time.Now().UnixMilli()

	// 심볼 정밀도 정보 조회 (최초 1회만 API 호출, 이후 캐시 사용)
	symbolInfo, err := bw.getSymbolInfo(bw.config.Symbol)
	if err != nil {
		bw.storage.AddLog("warning", fmt.Sprintf("바이낸스 심볼 정밀도 조회 실패, 기본 정밀도로 진행합니다: %v", err), bw.config.Exchange, bw.config.Symbol)
	}

	// 수량은 정수만, 가격은 원래 정밀도 사용
	pricePrecision := 8
	if symbolInfo != nil && symbolInfo.PricePrecision > 0 {
		pricePrecision = symbolInfo.PricePrecision
	}
	formattedQty := truncateToPrecision(bw.config.SellAmount, 0)
	formattedPrice := truncateToPrecision(bw.config.SellPrice, pricePrecision)

	params := url.Values{}
	params.Set("symbol", strings.ReplaceAll(bw.config.Symbol, "/", ""))
	params.Set("side", "SELL")
	params.Set("type", "LIMIT")
	params.Set("timeInForce", "GTC")
	params.Set("quantity", formattedQty)
	params.Set("price", formattedPrice)
	params.Set("timestamp", strconv.FormatInt(timestamp, 10))

	// 서명 생성
	signature := bw.generateBinanceSignature(params.Encode())
	params.Set("signature", signature)

	req, err := http.NewRequest("POST", bw.url, strings.NewReader(params.Encode()))
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 생성 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	req.Header.Set("X-MBX-APIKEY", bw.accessKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("HTTP 요청 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		bw.storage.AddLog("error", fmt.Sprintf("응답 파싱 실패: %v", err), bw.config.Exchange, bw.config.Symbol)
		return
	}

	if resp.StatusCode == 200 {
		orderID, ok := result["orderId"].(float64)
		if ok {
			orderIDStr := fmt.Sprintf("%.0f", orderID)
			bw.mu.Lock()
			if bw.lastSuccessOrderID != orderIDStr {
				bw.lastSuccessOrderID = orderIDStr
				bw.mu.Unlock()
				bw.storage.AddLog("success", fmt.Sprintf("%s, 매도수량: %s, 가격: %s, 심볼: %s 매도 주문에 성공했습니다.",
					bw.GetPlatformName(), formattedQty, formattedPrice, bw.config.Symbol), bw.config.Exchange, bw.config.Symbol)
			} else {
				bw.mu.Unlock()
			}
		} else {
			bw.mu.Lock()
			if bw.lastSuccessOrderID != "success" {
				bw.lastSuccessOrderID = "success"
				bw.mu.Unlock()
				bw.storage.AddLog("success", fmt.Sprintf("%s, 매도수량: %s, 가격: %s, 심볼: %s 매도 주문에 성공했습니다.",
					bw.GetPlatformName(), formattedQty, formattedPrice, bw.config.Symbol), bw.config.Exchange, bw.config.Symbol)
			} else {
				bw.mu.Unlock()
			}
		}
	} else {
		errorMsg := "알 수 없는 오류"
		if result["msg"] != nil {
			errorMsg = fmt.Sprintf("%v", result["msg"])
		}
		bw.storage.AddLog("error", fmt.Sprintf("%s, 매도수량: %s, 가격: %s, 심볼: %s 매도 주문에 실패했습니다. 거래소 응답 메세지: %s",
			bw.GetPlatformName(), formattedQty, formattedPrice, bw.config.Symbol, errorMsg), bw.config.Exchange, bw.config.Symbol)
	}
}

// generateBinanceSignature 바이낸스 HMAC-SHA256 서명 생성
func (bw *BinanceWorker) generateBinanceSignature(queryString string) string {
	h := hmac.New(sha256.New, []byte(bw.secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// getSymbolInfo 바이낸스 심볼 정밀도 정보를 조회하고 캐시합니다.
func (bw *BinanceWorker) getSymbolInfo(symbol string) (*BinanceSymbolInfo, error) {
	// 캐시 확인
	bw.symbolInfoMu.RLock()
	if info, ok := bw.symbolInfoCache[symbol]; ok {
		bw.symbolInfoMu.RUnlock()
		return info, nil
	}
	bw.symbolInfoMu.RUnlock()

	type binanceFilter struct {
		FilterType string `json:"filterType"`
		StepSize   string `json:"stepSize,omitempty"`
		TickSize   string `json:"tickSize,omitempty"`
		MinQty     string `json:"minQty,omitempty"`
		MinPrice   string `json:"minPrice,omitempty"`
	}

	type binanceSymbol struct {
		Symbol  string          `json:"symbol"`
		Filters []binanceFilter `json:"filters"`
	}

	type exchangeInfoResp struct {
		Symbols []binanceSymbol `json:"symbols"`
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.binance.com/api/v3/exchangeInfo", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info exchangeInfoResp
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	exSymbol := strings.ReplaceAll(symbol, "/", "")

	for _, s := range info.Symbols {
		if s.Symbol != exSymbol {
			continue
		}

		result := &BinanceSymbolInfo{Symbol: s.Symbol}

		for _, f := range s.Filters {
			switch f.FilterType {
			case "LOT_SIZE":
				result.MinQty = f.MinQty
				// stepSize로 수량 정밀도 계산
				if f.StepSize != "" {
					result.AmountPrecision = countDecimalPlaces(f.StepSize)
				}
			case "PRICE_FILTER":
				result.MinPrice = f.MinPrice
				// tickSize로 가격 정밀도 계산
				if f.TickSize != "" {
					result.PricePrecision = countDecimalPlaces(f.TickSize)
				}
			}
		}

		// 기본값 보정
		if result.PricePrecision <= 0 {
			result.PricePrecision = 8
		}
		if result.AmountPrecision <= 0 {
			result.AmountPrecision = 8
		}

		// 캐시에 저장
		bw.symbolInfoMu.Lock()
		bw.symbolInfoCache[symbol] = result
		bw.symbolInfoMu.Unlock()

		return result, nil
	}

	return nil, fmt.Errorf("바이낸스 심볼 정보를 찾을 수 없습니다: %s", symbol)
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BinanceWorker) GetPlatformName() string {
	return "Binance"
}
