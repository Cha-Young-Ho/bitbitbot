package platform

import (
	"bitbit-app/local_file"
	"context"
	"encoding/json" // Added for json.Unmarshal
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

// BybitWorker Bybit 플랫폼용 워커
type BybitWorker struct {
	*BaseWorker
	accessKey string
	secretKey string
	exchange  ccxt.IExchange
}

// NewBybitWorker 새로운 Bybit 워커를 생성합니다
func NewBybitWorker(order local_file.SellOrder, manager *WorkerManager, accessKey, secretKey, passwordPhrase string) *BybitWorker {
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

	exchange := ccxt.CreateExchange("bybit", exchangeConfig)

	// BaseWorker 생성
	baseWorker := NewBaseWorker(order, manager, accessKey, secretKey, passwordPhrase)

	return &BybitWorker{
		BaseWorker: baseWorker,
		accessKey:  accessKey,
		secretKey:  secretKey,
		exchange:   exchange,
	}
}

// Start 워커를 시작합니다
func (bw *BybitWorker) Start(ctx context.Context) error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.isRunning {
		return fmt.Errorf("워커가 이미 실행 중입니다: %s", bw.order.Name)
	}

	bw.ctx, bw.cancel = context.WithCancel(ctx)
	bw.isRunning = true
	bw.status.IsRunning = true

	// 워커 고루틴 시작 (Bybit 자체 run 사용)
	go bw.run()
	return nil
}

// run 워커의 메인 루프 (Bybit 전용)
func (bw *BybitWorker) run() {
	ticker := time.NewTicker(time.Duration(bw.order.Term) * time.Second)
	defer ticker.Stop()

	// 시작 로그 제거
	// Bybit 워커 시작 로그 제거

	for {
		select {
		case <-bw.ctx.Done():
			bw.sendLog("Bybit 워커가 중지되었습니다", "info")
			// Bybit 워커 중지 로그 제거
			return
		case <-ticker.C:
			bw.executeSellOrder()
		}
	}
}

// executeSellOrder 지정가 매도 주문을 실행합니다
func (bw *BybitWorker) executeSellOrder() {
	// BaseWorker의 상태 업데이트
	bw.mu.Lock()
	bw.status.LastCheck = time.Now()
	bw.status.CheckCount++
	bw.mu.Unlock()

	// 거래소가 nil인 경우 에러 처리
	if bw.exchange == nil {
		bw.mu.Lock()
		bw.status.ErrorCount++
		bw.status.LastError = "거래소가 초기화되지 않았습니다"
		bw.mu.Unlock()

		bw.sendLog("거래소가 초기화되지 않았습니다", "error")
		return
	}

	// Bybit 심볼 형식으로 변환 (예: BTC/USDT -> BTC/USDT)
	bybitSymbol := bw.convertToBybitSymbol(bw.order.Symbol)

	// CCXT를 사용한 지정가 매도 주문
	_, err := bw.exchange.CreateLimitSellOrder(
		bybitSymbol,       // 심볼 (예: BTC/USDT)
		bw.order.Quantity, // 수량
		bw.order.Price,    // 가격
	)

	if err != nil {
		bw.mu.Lock()
		bw.status.ErrorCount++
		bw.status.LastError = err.Error()
		bw.mu.Unlock()

		// CCXT 에러 응답을 파싱하여 retMsg만 추출
		errorMessage := bw.parseCCXTError(err.Error())
		
		// 일관된 로그 포맷으로 에러 출력 (retMsg만 표시)
		bw.sendLog(fmt.Sprintf("주문 실패\n이유: %s\n심볼: %s\n가격: %.8f", 
			errorMessage, bw.order.Symbol, bw.order.Price), "error", bw.order.Price, bw.order.Quantity)
		
		bw.manager.SendSystemLog("BybitWorker", "executeSellOrder",
			fmt.Sprintf("매도 주문 실패: %v", err), "error", "", bw.order.Name, err.Error())
		return
	}

	// 성공 시 간단한 로그만 출력
	bw.sendLog("주문 성공", "success", bw.order.Price, bw.order.Quantity)
}

// parseCCXTError CCXT 에러 응답을 파싱하여 retMsg만 추출
func (bw *BybitWorker) parseCCXTError(errorStr string) string {
	// CCXT 에러 응답이 JSON 형태인지 확인
	if strings.Contains(errorStr, "{") && strings.Contains(errorStr, "}") {
		// JSON 파싱 시도
		var errorResponse map[string]interface{}
		if err := json.Unmarshal([]byte(errorStr), &errorResponse); err == nil {
			// retMsg가 있으면 사용 (Bybit 특화)
			if retMsg, ok := errorResponse["retMsg"].(string); ok && retMsg != "" {
				return retMsg
			}
			// message가 있으면 사용
			if message, ok := errorResponse["message"].(string); ok && message != "" {
				return message
			}
			// error가 있으면 사용
			if errorMsg, ok := errorResponse["error"].(string); ok && errorMsg != "" {
				return errorMsg
			}
		}
	}
	
	// JSON 파싱이 실패하거나 구조가 다른 경우 원본 에러 메시지 반환
	return errorStr
}

// formatLogMessage 로그 메시지 포맷팅
func (bw *BybitWorker) formatLogMessage(messageType, message string, price, quantity float64) string {
	timestamp := time.Now().Format("15:04:05")
	
	switch messageType {
	case "order":
		return fmt.Sprintf("[%s] %s | 가격: %.8f | 수량: %.8f", timestamp, message, price, quantity)
	case "success":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	case "error":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	case "info":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	case "warning":
		return fmt.Sprintf("[%s] %s", timestamp, message)
	default:
		return fmt.Sprintf("[%s] %s", timestamp, message)
	}
}

// GetPlatformName 플랫폼 이름을 반환합니다
func (bw *BybitWorker) GetPlatformName() string {
	return "Bybit"
}

// convertToBybitSymbol Bybit 심볼 형식으로 변환합니다
func (bw *BybitWorker) convertToBybitSymbol(symbol string) string {
	// Bybit는 CCXT 표준 형식 사용 (예: BTC/USDT)
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		bw.sendLog(fmt.Sprintf("잘못된 심볼 형식: %s (올바른 형식: BTC/USDT)", symbol), "warning")
		return symbol
	}

	base := strings.TrimSpace(strings.ToUpper(parts[0]))  // BTC
	quote := strings.TrimSpace(strings.ToUpper(parts[1])) // USDT

	// Bybit 마켓 형식으로 변환
	bybitSymbol := base + "/" + quote // "BTC/USDT"

	return bybitSymbol
}
