package platform

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	jose "gopkg.in/go-jose/go-jose.v2"
	"gopkg.in/go-jose/go-jose.v2/jwt"
)

type CoinbaseWorker struct {
	config  *WorkerConfig
	storage *MemoryStorage
	mu      sync.RWMutex
	stopCh  chan struct{}
	running bool

	apiKeyID   string // CDP API Key ID (kid: organizations/.../apiKeys/<kid>)
	apiKeyPEM  string // CDP API Key EC Private Key (PEM)
	httpClient *http.Client
}

// Advanced Trade (brokerage) 주문 요청/응답 구조체
type marketIOCConfig struct {
	BaseSize  string `json:"base_size,omitempty"`
	QuoteSize string `json:"quote_size,omitempty"`
}

type limitGTCConfig struct {
	BaseSize   string `json:"base_size,omitempty"`
	LimitPrice string `json:"limit_price"`
	PostOnly   bool   `json:"post_only,omitempty"`
}

type orderConfiguration struct {
	MarketIOC     *marketIOCConfig `json:"market_market_ioc,omitempty"`
	LimitLimitGTC *limitGTCConfig  `json:"limit_limit_gtc,omitempty"`
}

type createOrderRequest struct {
	ClientOrderID      string             `json:"client_order_id"`
	ProductID          string             `json:"product_id"`
	Side               string             `json:"side"` // "SELL" or "BUY"
	OrderConfiguration orderConfiguration `json:"order_configuration"`
}

type createOrderSuccess struct {
	OrderID string `json:"order_id"`
}

type createOrderResponse struct {
	Success         bool                `json:"success"`
	SuccessResponse *createOrderSuccess `json:"success_response,omitempty"`
	ErrorResponse   map[string]any      `json:"error_response,omitempty"`
}

// NewCoinbaseWorker — Coinbase Advanced Trade 워커 생성 (CDP JWT)
func NewCoinbaseWorker(config *WorkerConfig, storage *MemoryStorage) *CoinbaseWorker {
	// AccessKey = kid, SecretKey = EC PRIVATE KEY (PEM)
	apiKeyID := config.AccessKey
	apiKeyPEM := config.SecretKey

	storage.AddLog("info", "Coinbase Advanced Trade 워커가 성공적으로 초기화되었습니다", config.Exchange, config.Symbol)

	return &CoinbaseWorker{
		config:     config,
		storage:    storage,
		stopCh:     make(chan struct{}),
		apiKeyID:   apiKeyID,
		apiKeyPEM:  apiKeyPEM,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Start — 주기적으로 매도 주문 실행
func (cbw *CoinbaseWorker) Start(ctx context.Context) {
	cbw.mu.Lock()
	cbw.running = true
	cbw.mu.Unlock()

	// 티커 생성 (밀리초 단위로 변환)
	intervalMs := int64(cbw.config.RequestInterval * 1000)
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval < time.Millisecond {
		interval = time.Millisecond // 최소 1ms
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	cbw.storage.AddLog("info", "Coinbase Advanced Trade 워커 시작", cbw.config.Exchange, cbw.config.Symbol)

	for {
		select {
		case <-ctx.Done():
			cbw.Stop()
			return
		case <-cbw.stopCh:
			return
		case <-ticker.C:
			cbw.executeSellOrder(ctx)
		}
	}
}

// Stop — 워커 중지
func (cbw *CoinbaseWorker) Stop() {
	cbw.mu.Lock()
	defer cbw.mu.Unlock()
	if cbw.running {
		cbw.running = false
		close(cbw.stopCh)
		cbw.storage.AddLog("info", "Coinbase Advanced Trade 워커 중지됨", cbw.config.Exchange, cbw.config.Symbol)
	}
}

// IsRunning — 워커 실행 상태 확인
func (cbw *CoinbaseWorker) IsRunning() bool {
	cbw.mu.RLock()
	defer cbw.mu.RUnlock()
	return cbw.running
}

// executeSellOrder — Advanced Trade (brokerage) API를 사용한 매도 주문 실행 (CDP JWT)
func (cbw *CoinbaseWorker) executeSellOrder(ctx context.Context) {
	cbw.mu.RLock()
	if !cbw.running {
		cbw.mu.RUnlock()
		return
	}
	cbw.mu.RUnlock()

	if cbw.httpClient == nil {
		cbw.storage.AddLog("error", "HTTP 클라이언트가 초기화되지 않았습니다", cbw.config.Exchange, cbw.config.Symbol)
		return
	}

	// 심볼 변환 (XRP/USDT -> XRP-USDT)
	productID := strings.ToUpper(strings.ReplaceAll(cbw.config.Symbol, "/", "-"))

	// 매도 주문 생성 (limit GTC)
	order := createOrderRequest{
		ClientOrderID: fmt.Sprintf("cdp-%d", time.Now().UnixNano()),
		ProductID:     productID,
		Side:          "SELL",
		OrderConfiguration: orderConfiguration{
			LimitLimitGTC: &limitGTCConfig{
				BaseSize:   fmt.Sprintf("%.8f", cbw.config.SellAmount),
				LimitPrice: fmt.Sprintf("%.8f", cbw.config.SellPrice),
				PostOnly:   false,
			},
		},
	}

	// 주문 실행
	orderResp, err := cbw.createOrderJWT(ctx, order)
	if err != nil {
		cbw.storage.AddLog("error", fmt.Sprintf("매도 주문 실패: %v", err), cbw.config.Exchange, cbw.config.Symbol)
		return
	}

	if orderResp != nil {
		if orderResp.Success && orderResp.SuccessResponse != nil {
			cbw.storage.AddLog("success", fmt.Sprintf("%s, 매도수량: %.8f, 가격: %.2f, 심볼: %s 매도 주문에 성공했습니다.",
				cbw.GetPlatformName(), cbw.config.SellAmount, cbw.config.SellPrice, cbw.config.Symbol), cbw.config.Exchange, cbw.config.Symbol)
		} else {
			errorMsg := "알 수 없는 오류"
			if orderResp.ErrorResponse != nil {
				errorMsg = fmt.Sprintf("%+v", orderResp.ErrorResponse)
			}
			cbw.storage.AddLog("error", fmt.Sprintf("%s, 매도수량: %.8f, 가격: %.2f, 심볼: %s 매도 주문에 실패했습니다. 거래소 응답 메세지: %s",
				cbw.GetPlatformName(), cbw.config.SellAmount, cbw.config.SellPrice, cbw.config.Symbol, errorMsg), cbw.config.Exchange, cbw.config.Symbol)
		}
	}
}

// nonce 및 JWT 빌더 구현 (CDP 예제 기반)
var maxRand = big.NewInt(math.MaxInt64)

type nonceSource struct{}

func (n nonceSource) Nonce() (string, error) {
	r, err := rand.Int(rand.Reader, maxRand)
	if err != nil {
		return "", err
	}
	return r.String(), nil
}

type apiKeyClaims struct {
	*jwt.Claims
	URI string `json:"uri"`
}

// buildJWT — Advanced Trade용 CDP JWT 생성
func (cbw *CoinbaseWorker) buildJWT(uri string) (string, error) {
	key, err := parseECPrivateKeyFromString(cbw.apiKeyPEM)
	if err != nil {
		return "", fmt.Errorf("jwt: %w", err)
	}

	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.ES256, Key: key},
		(&jose.SignerOptions{NonceSource: nonceSource{}}).WithType("JWT").WithHeader("kid", cbw.apiKeyID),
	)
	if err != nil {
		return "", fmt.Errorf("jwt: %w", err)
	}

	cl := &apiKeyClaims{
		Claims: &jwt.Claims{
			Subject:   cbw.apiKeyID,
			Issuer:    "cdp",
			NotBefore: jwt.NewNumericDate(time.Now()),
			Expiry:    jwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
		},
		URI: uri,
	}

	jwtString, err := jwt.Signed(sig).Claims(cl).CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("jwt: %w", err)
	}
	return jwtString, nil
}

// parseECPrivateKeyFromString handles multiple encodings:
// - PEM SEC1 (BEGIN EC PRIVATE KEY)
// - PEM PKCS#8 (BEGIN PRIVATE KEY)
// - Raw base64 DER (SEC1 or PKCS#8)
// Also converts literal \n sequences to real newlines when needed
func parseECPrivateKeyFromString(keyStr string) (*ecdsa.PrivateKey, error) {
	if keyStr == "" {
		return nil, fmt.Errorf("empty key")
	}

	// Normalize: convert literal \n to newline if present
	if strings.Contains(keyStr, "\\n") && !strings.Contains(keyStr, "\n") {
		keyStr = strings.ReplaceAll(keyStr, "\\n", "\n")
	}

	// Try PEM decode first
	if block, _ := pem.Decode([]byte(keyStr)); block != nil {
		// SEC1 EC PRIVATE KEY
		if strings.Contains(block.Type, "EC PRIVATE KEY") {
			if ecKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
				return ecKey, nil
			}
		}
		// PKCS#8 PRIVATE KEY
		if strings.Contains(block.Type, "PRIVATE KEY") {
			if pk, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
				if ecKey, ok := pk.(*ecdsa.PrivateKey); ok {
					return ecKey, nil
				}
				return nil, fmt.Errorf("unsupported key type in PKCS#8")
			}
		}
		return nil, fmt.Errorf("unsupported PEM key type: %s", block.Type)
	}

	// If not PEM, try base64 DER
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keyStr))
	if err == nil {
		if ecKey, err2 := x509.ParseECPrivateKey(der); err2 == nil {
			return ecKey, nil
		}
		if pk, err2 := x509.ParsePKCS8PrivateKey(der); err2 == nil {
			if ecKey, ok := pk.(*ecdsa.PrivateKey); ok {
				return ecKey, nil
			}
			return nil, fmt.Errorf("unsupported key type in DER PKCS#8")
		}
	}

	return nil, fmt.Errorf("private key format not recognized. Ensure EC key PEM with headers")
}

// createOrderJWT — Advanced Trade 주문 생성 (CDP JWT Bearer)
func (cbw *CoinbaseWorker) createOrderJWT(ctx context.Context, order createOrderRequest) (*createOrderResponse, error) {
	orderData, err := json.Marshal(order)
	if err != nil {
		return nil, fmt.Errorf("주문 데이터 JSON 변환 실패: %v", err)
	}

	const host = "api.coinbase.com"
	const path = "/api/v3/brokerage/orders"
	uriClaim := fmt.Sprintf("%s %s%s", "POST", host, path)

	jwtToken, err := cbw.buildJWT(uriClaim)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://"+host+path, bytes.NewReader(orderData))
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := cbw.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API 요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var parsed createOrderResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("응답 JSON 파싱 실패: %v (%s)", err, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return &parsed, fmt.Errorf("API 오류 (상태코드: %d): %s", resp.StatusCode, string(body))
	}

	return &parsed, nil
}

// GetPlatformName — 플랫폼명 반환
func (cbw *CoinbaseWorker) GetPlatformName() string {
	return "Coinbase Advanced Trade"
}
