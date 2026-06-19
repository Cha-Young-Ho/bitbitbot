package platform

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CoinoneWorker 코인원 거래소 워커
type CoinoneWorker struct {
	mu                 sync.RWMutex
	config             *WorkerConfig
	storage            *MemoryStorage
	running            bool
	stopCh             chan struct{}
	accessKey          string
	secretKey          string
	url                string
	lastSuccessOrderID string
	pricePrecision     int
	quantityPrecision  int
}

// NewCoinoneWorker 새로운 코인원 워커를 생성합니다
func NewCoinoneWorker(config *WorkerConfig, storage *MemoryStorage) *CoinoneWorker {
	return &CoinoneWorker{
		config:          config,
		storage:         storage,
		running:         false,
		stopCh:          make(chan struct{}),
		accessKey:       config.AccessKey,
		secretKey:       config.SecretKey,
		url:             "https://api.coinone.co.kr/v2.1/order",
		pricePrecision:  8,
		quantityPrecision: 8,
	}
}

// Start 워커를 시작합니다
func (cw *CoinoneWorker) Start(ctx context.Context) {
	cw.mu.Lock()
	cw.running = true
	cw.mu.Unlock()

	cw.storage.AddLog("info", "코인원 워커가 시작되었습니다.", cw.config.Exchange, cw.config.Symbol)

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(cw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		// 실행 상태 확인
		cw.mu.RLock()
		if !cw.running {
			cw.mu.RUnlock()
			cw.storage.AddLog("info", "코인원 워커가 중지되었습니다.", cw.config.Exchange, cw.config.Symbol)
			return
		}
		cw.mu.RUnlock()

		select {
		case <-ctx.Done():
			cw.mu.Lock()
			cw.running = false
			cw.mu.Unlock()
			cw.storage.AddLog("info", "코인원 워커가 중지되었습니다.", cw.config.Exchange, cw.config.Symbol)
			return
		case <-cw.stopCh:
			cw.mu.Lock()
			cw.running = false
			cw.mu.Unlock()
			cw.storage.AddLog("info", "코인원 워커가 중지되었습니다.", cw.config.Exchange, cw.config.Symbol)
			return
		case <-ticker.C:
			// 실행 상태 재확인 후 요청 처리
			cw.mu.RLock()
			if cw.running {
				cw.mu.RUnlock()
				cw.executeSellOrder()
			} else {
				cw.mu.RUnlock()
				return
			}
		}
	}
}

// Stop 워커를 중지합니다
func (cw *CoinoneWorker) Stop() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.running {
		cw.running = false
		// 채널이 이미 닫혀있지 않은 경우에만 닫기
		select {
		case <-cw.stopCh:
			// 이미 닫혀있음
		default:
			close(cw.stopCh)
		}
	}
}

// IsRunning 워커 실행 상태 확인
func (cw *CoinoneWorker) IsRunning() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.running
}

// executeSellOrder 코인원에서 매도 주문 실행
func (cw *CoinoneWorker) executeSellOrder() {
	// 실행 상태 재확인
	cw.mu.RLock()
	if !cw.running {
		cw.mu.RUnlock()
		return
	}
	cw.mu.RUnlock()

	// 심볼 변환 (BTC/KRW -> BTC)
	coinoneSymbol := cw.convertToCoinoneSymbol(cw.config.Symbol)

	// 가격/수량 정밀도에 맞게 truncate
	// 수량은 정수만, 가격은 기본 8자리 사용
	formattedQty := truncateToPrecision(cw.config.SellAmount, 0)
	formattedPrice := fmt.Sprintf("%.8f", cw.config.SellPrice)

	// Coinone API 2.1 직접 호출
	orderID, err := cw.createCoinoneOrderV21(coinoneSymbol, formattedPrice, formattedQty)
	if err != nil {
		cw.storage.AddLog("error", fmt.Sprintf("매도 주문 실패: %v", err), cw.config.Exchange, cw.config.Symbol)
		return
	}

	// 성공 로그
	cw.storage.AddLog("success", fmt.Sprintf("지정가 매도 주문 생성 완료 (가격: %s, 수량: %s, 주문ID: %s)",
		formattedPrice, formattedQty, orderID), cw.config.Exchange, cw.config.Symbol)
}

// createCoinoneOrderV21 코인원 API 2.1을 사용하여 주문 생성
func (cw *CoinoneWorker) createCoinoneOrderV21(coinoneSymbol, formattedPrice, formattedQty string) (string, error) {
	// 1. 요청 바디 구성 (코인원 API 2.1 문서 기준)
	nonce := uuid.New().String() // UUID v4 형식

	requestBody := map[string]interface{}{
		"access_token":    cw.accessKey,
		"nonce":           nonce,
		"side":            "SELL", // 매도
		"quote_currency":  "KRW",
		"target_currency": coinoneSymbol,
		"type":            "LIMIT",                                  // 지정가
		"price":           formattedPrice,
		"qty":             formattedQty,
		"post_only":       false, // Maker/Taker 모두 허용 (매수벽에 걸려도 체결 가능)
	}

	// 2. JSON 문자열로 변환
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON 변환 실패: %v", err)
	}

	// 3. Base64 인코딩 (페이로드)
	payload := base64.StdEncoding.EncodeToString(jsonBody)

	// 4. HMAC-SHA512 서명 생성
	signature := cw.createCoinoneSignature(payload)

	// 5. HTTP 요청 생성
	req, err := http.NewRequest("POST", cw.url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 6. 헤더 설정 (코인원 API 2.1 문서 기준)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-COINONE-PAYLOAD", payload)
	req.Header.Set("X-COINONE-SIGNATURE", signature)

	// 7. HTTP 요청 전송
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("요청 전송 실패: %v", err)
	}
	defer resp.Body.Close()

	// 8. 응답 파싱
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("응답 파싱 실패: %v", err)
	}

	// 응답 로그 출력 (디버깅용)
	cw.storage.AddLog("info", fmt.Sprintf("코인원 API 응답: %+v", response), cw.config.Exchange, cw.config.Symbol)

	// 9. 응답 검증
	if resp.StatusCode != 200 {
		errorMsg := "알 수 없는 오류"
		if response["errorCode"] != nil {
			errorMsg = fmt.Sprintf("에러코드: %v", response["errorCode"])
		}
		if response["errorMsg"] != nil {
			errorMsg += fmt.Sprintf(", 메시지: %v", response["errorMsg"])
		}
		return "", fmt.Errorf("API 오류 (status=%d): %s", resp.StatusCode, errorMsg)
	}

	// 10. 결과 확인 (코인원 API는 result 필드로 성공/실패 구분)
	if result, ok := response["result"].(string); ok && result == "error" {
		errorMsg := "알 수 없는 오류"
		if response["error_code"] != nil {
			errorMsg = fmt.Sprintf("에러코드: %v", response["error_code"])
		}
		if response["error_msg"] != nil {
			errorMsg += fmt.Sprintf(", 메시지: %v", response["error_msg"])
		}
		return "", fmt.Errorf("코인원 API 오류: %s", errorMsg)
	}

	// 11. 주문 ID 추출 (API 2.1에서는 order_id 필드 사용)
	orderID, ok := response["order_id"].(string)
	if !ok || orderID == "" {
		return "", fmt.Errorf("주문 ID 없음: %v", response)
	}

	return orderID, nil
}

// createCoinoneSignature 코인원 HMAC-SHA512 서명 생성
func (cw *CoinoneWorker) createCoinoneSignature(payload string) string {
	// HMAC-SHA512 서명 생성
	h := hmac.New(sha512.New, []byte(cw.secretKey))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// convertToCoinoneSymbol 심볼을 코인원 형식으로 변환
func (cw *CoinoneWorker) convertToCoinoneSymbol(symbol string) string {
	// BTC/KRW -> BTC
	// USDT/KRW -> USDT
	parts := strings.Split(symbol, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return symbol
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (cw *CoinoneWorker) GetPlatformName() string {
	return "Coinone"
}
