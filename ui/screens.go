package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"gui-app/config"
	"gui-app/models"
	"gui-app/services"
	"gui-app/utils"
)

// App UI 애플리케이션 구조체
type App struct {
	fyneApp     fyne.App
	config      *config.AppConfig
	dataService *services.DataService
	mainWindow  fyne.Window
}

// NewApp 새로운 UI 앱을 생성합니다
func NewApp(fyneApp fyne.App, cfg *config.AppConfig, dataService *services.DataService) *App {
	app := &App{
		fyneApp:     fyneApp,
		config:      cfg,
		dataService: dataService,
	}

	// 모던 테마 적용
	fyneApp.Settings().SetTheme(NewModernTheme())

	return app
}

// ShowLoginScreen 로그인 화면을 표시합니다
func (a *App) ShowLoginScreen() {
	appName, version := a.config.GetAppInfo()

	loginWindow := a.fyneApp.NewWindow("🔐 로그인 - " + appName)
	width, height := a.config.GetWindowSize()
	loginWindow.Resize(fyne.NewSize(width*0.4, height*0.6))
	loginWindow.CenterOnScreen()

	// 헤더 섹션
	titleLabel := widget.NewLabel(appName)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	subtitleLabel := widget.NewLabel("거래소 API 키 관리 플랫폼")
	subtitleLabel.Alignment = fyne.TextAlignCenter

	versionLabel := widget.NewLabel(fmt.Sprintf("Version %s", version))
	versionLabel.Alignment = fyne.TextAlignCenter

	// 로그인 폼
	keyEntry := widget.NewPasswordEntry()
	keyEntry.SetPlaceHolder("마스터 키를 입력하세요")

	loginBtn := NewPrimaryButton("🚀 시작하기", func() {
		a.handleLogin(keyEntry.Text, loginWindow)
	})

	// 특별한 기능 버튼 추가
	specialBtn := widget.NewButton("🔧 특별한 기능", func() {
		a.handleSpecialFunction()
	})
	specialBtn.Importance = widget.LowImportance

	// 키 엔터 이벤트
	keyEntry.OnSubmitted = func(text string) {
		a.handleLogin(text, loginWindow)
	}

	// 보안 안내
	securityInfo := NewInfoLabel(
		"🔒 모든 데이터는 사용자 컴퓨터에 암호화되어 저장됩니다.\n처음 사용 시 원하는 마스터 키를 설정하세요.",
		fyne.TextAlignCenter,
	)

	// 레이아웃 구성
	content := container.NewVBox(
		container.NewPadded(
			container.NewVBox(
				titleLabel,
				subtitleLabel,
				versionLabel,
			),
		),
		widget.NewSeparator(),
		container.NewPadded(
			container.NewVBox(
				widget.NewLabel("🔑 마스터 키"),
				keyEntry,
				loginBtn,
				specialBtn,
			),
		),
		widget.NewSeparator(),
		container.NewPadded(securityInfo),
	)

	loginWindow.SetContent(content)
	loginWindow.Show()
}

// handleLogin 로그인 처리
func (a *App) handleLogin(key string, loginWindow fyne.Window) {
	if key == "" {
		dialog.ShowError(fmt.Errorf("마스터 키를 입력해주세요"), loginWindow)
		return
	}

	// 사용자 키 설정
	if err := a.dataService.SetUserKey(key); err != nil {
		dialog.ShowError(fmt.Errorf("키 검증 실패: %v", err), loginWindow)
		return
	}

	// 데이터 로드
	if err := a.dataService.LoadData(); err != nil {
		// 더 자세한 에러 정보 제공
		var errorMsg string
		if strings.Contains(err.Error(), "복호화 실패") {
			errorMsg = fmt.Sprintf("로그인 실패: 잘못된 마스터 키입니다.\n\n이전에 다른 키를 사용하셨다면 해당 키를 입력해주세요.\n처음 사용하시는 경우 원하는 키를 입력하세요.\n\n상세 오류: %v", err)
		} else if strings.Contains(err.Error(), "파일 읽기 실패") {
			errorMsg = fmt.Sprintf("로그인 실패: 데이터 파일 읽기 오류\n\n%v", err)
		} else {
			errorMsg = fmt.Sprintf("로그인 실패: %v", err)
		}
		dialog.ShowError(fmt.Errorf(errorMsg), loginWindow)
		return
	}

	// 성공 시 메인 화면으로 이동
	loginWindow.Close()
	a.ShowMainScreen()
}

// ShowMainScreen 메인 화면을 표시합니다 (간소화)
func (a *App) ShowMainScreen() {
	appName, _ := a.config.GetAppInfo()

	a.mainWindow = a.fyneApp.NewWindow("📊 " + appName)
	width, height := a.config.GetWindowSize()
	a.mainWindow.Resize(fyne.NewSize(width, height))
	a.mainWindow.CenterOnScreen()

	// 헤더 생성
	header := a.createHeader()

	// 메인 콘텐츠 생성
	mainContent := a.createMainContent()

	// 전체 레이아웃
	content := container.NewVBox(
		container.NewPadded(header),
		widget.NewSeparator(),
		container.NewPadded(mainContent),
	)

	a.mainWindow.SetContent(content)
	a.mainWindow.Show()
}

// createHeader 헤더를 생성합니다
func (a *App) createHeader() *fyne.Container {
	appName, version := a.config.GetAppInfo()

	titleLabel := widget.NewLabel("📊 " + appName)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	versionLabel := widget.NewLabel("v" + version)

	timeLabel := widget.NewLabel(utils.FormatDateTime(time.Now()))

	// 상단 액션 버튼들
	addExchangeBtn := NewActionButton("API 키 추가", "🔑", widget.HighImportance, func() {
		a.ShowAddExchangeDialog()
	})

	refreshBtn := NewActionButton("새로고침", "🔄", widget.MediumImportance, func() {
		a.RefreshMainScreen()
	})

	logoutBtn := NewActionButton("로그아웃", "🚪", widget.LowImportance, func() {
		a.handleLogout()
	})

	leftSide := container.NewHBox(titleLabel, versionLabel)
	rightSide := container.NewHBox(timeLabel, addExchangeBtn, refreshBtn, logoutBtn)

	return container.NewBorder(nil, nil, leftSide, rightSide)
}

// createMainContent 메인 콘텐츠를 생성합니다
func (a *App) createMainContent() *fyne.Container {
	// 좌측: 거래소 목록
	exchangeSection := a.createExchangeSection()

	// 우측: 주문 목록
	orderSection := a.createOrderSection()

	// 좌우 분할 (구분선이 더 명확하게)
	leftPanel := NewSectionCard("🔑 등록된 API 키", exchangeSection)
	rightPanel := NewSectionCard("📋 매도 주문", orderSection)

	// 수직 구분선을 위한 컨테이너
	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.5) // 50:50 비율

	return container.NewBorder(nil, nil, nil, nil, split)
}

// createExchangeSection 거래소 섹션을 생성합니다
func (a *App) createExchangeSection() fyne.CanvasObject {
	exchanges := a.dataService.GetActiveExchanges()

	if len(exchanges) == 0 {
		return NewEmptyState(
			"등록된 API 키가 없습니다.\nAPI 키를 추가하여 시작하세요.",
			"API 키 추가",
			func() { a.ShowAddExchangeDialog() },
		)
	}

	var exchangeCards []fyne.CanvasObject

	for _, exchange := range exchanges {
		orders := a.dataService.GetSellOrdersByExchange(exchange.ID)
		// 거래소 정보 가져오기
		exchangeInfo := models.GetExchangeInfo(exchange.Type)
		if exchangeInfo == nil {
			continue
		}

		card := NewSimpleExchangeCard(
			exchange,
			*exchangeInfo,
			len(orders),
			func(exchangeID string) { a.ShowAddOrderDialog(exchangeID) },
			func(exchangeID string) { a.ShowManageOrdersDialog(exchangeID) },
		)

		exchangeCards = append(exchangeCards, card)
	}

	content := container.NewVBox(exchangeCards...)
	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(0, 400))

	return scroll
}

// createOrderSection 주문 섹션을 생성합니다
func (a *App) createOrderSection() fyne.CanvasObject {
	orders := a.dataService.GetSellOrders()

	if len(orders) == 0 {
		return NewEmptyState(
			"등록된 매도 주문이 없습니다.\nAPI 키에서 주문을 생성하세요.",
			"",
			nil,
		)
	}

	var orderCards []fyne.CanvasObject

	for _, order := range orders {
		card := NewOrderCard(
			order,
			func(order models.SellOrder) { a.ShowEditOrderDialog(order) },
			func(orderID string) { a.handleDeleteOrder(orderID) },
		)

		orderCards = append(orderCards, card)
	}

	content := container.NewVBox(orderCards...)
	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(0, 400))

	return scroll
}

// RefreshMainScreen 메인 화면을 새로고침합니다
func (a *App) RefreshMainScreen() {
	fmt.Println("=== RefreshMainScreen 시작 ===")
	if a.mainWindow != nil {
		fmt.Println("기존 메인 윈도우 닫기...")
		a.mainWindow.Close()
		fmt.Println("기존 메인 윈도우 닫기 완료")
	} else {
		fmt.Println("메인 윈도우가 nil임")
	}
	fmt.Println("새로운 메인 화면 표시...")
	a.ShowMainScreen()
	fmt.Println("=== RefreshMainScreen 완료 ===")
}

// handleLogout 로그아웃을 처리합니다
func (a *App) handleLogout() {
	dialog.ShowConfirm(
		"로그아웃",
		"정말 로그아웃하시겠습니까?",
		func(confirm bool) {
			if confirm {
				if a.mainWindow != nil {
					a.mainWindow.Close()
				}
				a.ShowLoginScreen()
			}
		},
		a.mainWindow,
	)
}

// handleDeleteOrder 주문 삭제를 처리합니다
func (a *App) handleDeleteOrder(orderID string) {
	dialog.ShowConfirm(
		"주문 삭제",
		"이 주문을 삭제하시겠습니까?",
		func(confirm bool) {
			if confirm {
				if err := a.dataService.DeleteSellOrder(orderID); err != nil {
					dialog.ShowError(err, a.mainWindow)
				} else {
					a.RefreshMainScreen()
				}
			}
		},
		a.mainWindow,
	)
}

// handleSpecialFunction 특별한 기능을 처리합니다 (사용자 정의 코드 실행)
func (a *App) handleSpecialFunction() {
	fmt.Println("=== 특별한 기능 실행 시작 ===")

	// ========================================
	// 여기에 나만의 코드를 작성하세요!
	// ========================================

	// 예시 1: 간단한 메시지 출력
	fmt.Println("🎯 나만의 특별한 기능이 실행되었습니다!")

	// 예시 2: 시스템 정보 출력
	fmt.Printf("📊 현재 시간: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	// 예시 3: HTTP 요청 테스트
	fmt.Println("🌐 HTTP 요청 테스트 중...")
	go func() {
		if err := a.testHTTPRequest(); err != nil {
			fmt.Printf("❌ HTTP 요청 실패: %v\n", err)
		}
	}()

	// 예시 4: 데이터 서비스 접근 (주의: 로그인 전이므로 제한적)
	if a.dataService != nil {
		exchanges := a.dataService.GetSupportedExchanges()
		fmt.Printf("💱 지원하는 거래소 수: %d개\n", len(exchanges))
		for i, exchange := range exchanges {
			fmt.Printf("   %d. %s %s\n", i+1, exchange.Logo, exchange.DisplayName)
		}
	}

	// 예시 5: 사용자 확인 다이얼로그
	dialog.ShowConfirm(
		"특별한 기능",
		"특별한 기능이 실행되었습니다!\n터미널에서 로그를 확인하세요.\n\nHTTP 요청도 실행하시겠습니까?",
		func(response bool) {
			if response {
				fmt.Println("✅ 사용자가 HTTP 요청 실행을 선택했습니다.")
				// 여기에 추가 로직을 작성할 수 있습니다.
				a.executeCustomLogic()
			} else {
				fmt.Println("❌ 사용자가 취소를 선택했습니다.")
			}
		},
		a.fyneApp.NewWindow("특별한 기능"),
	)

	fmt.Println("=== 특별한 기능 실행 완료 ===")
}

// testHTTPRequest JWT 토큰으로 WebSocket 연결 테스트를 실행합니다
func (a *App) testHTTPRequest() error {
	// panic 복구를 위한 defer 추가
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("🚨 WebSocket 연결 중 복구된 panic: %v\n", r)
		}
	}()

	fmt.Println("📡 WebSocket 연결 테스트 시작...")
	accessKey := "10cLxYAMPGuNOPu3kjBMcjz53Z50EwAdmil9xzL1"
	secretKey := "0r4yQdTm5QAxgejmiAYT7KWSPilH4r5HpKexOzWk"

	// JWT 토큰 생성 (Node.js 코드와 동일)
	fmt.Println("🔐 JWT 토큰 생성 중...")
	nonce1 := uuid.New().String()
	payload := map[string]interface{}{
		"access_key": accessKey,
		"nonce":      nonce1, // uuidv4()와 동일
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(payload))
	jwtToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return fmt.Errorf("JWT 토큰 생성 실패: %v", err)
	}

	fmt.Printf("✅ JWT 토큰 생성 성공!\n")
	fmt.Printf("🔍 JWT 토큰: %s\n", jwtToken)

	// WebSocket 연결 설정
	fmt.Println("🌐 WebSocket 연결 준비 중...")
	wsURL := "wss://api.upbit.com/websocket/v1/private"

	// Authorization 헤더 설정
	headers := http.Header{}
	headers.Add("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	fmt.Printf("🔑 Authorization 헤더: Bearer %s...\n", jwtToken[:50])
	fmt.Printf("📡 연결 URL: %s\n", wsURL)

	// WebSocket 연결
	fmt.Println("🔌 WebSocket 연결 시도 중...")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		fmt.Printf("❌ WebSocket 연결 실패: %v\n", err)

		if resp != nil {
			fmt.Printf("📊 응답 상태: %d %s\n", resp.StatusCode, resp.Status)

			// 응답 헤더 출력
			fmt.Println("📋 응답 헤더:")
			for key, values := range resp.Header {
				for _, value := range values {
					fmt.Printf("   %s: %s\n", key, value)
				}
			}

			// 응답 본문 읽기 (에러 상세 정보)
			if resp.Body != nil {
				defer resp.Body.Close()
				if body, readErr := io.ReadAll(resp.Body); readErr == nil {
					if len(body) > 0 {
						fmt.Printf("📄 응답 본문 (크기: %d bytes):\n%s\n", len(body), string(body))
					} else {
						fmt.Println("📄 응답 본문이 비어있습니다")
					}
				} else {
					fmt.Printf("❌ 응답 본문 읽기 실패: %v\n", readErr)
				}
			} else {
				fmt.Println("📄 응답 본문이 없습니다")
			}
		} else {
			fmt.Println("❌ HTTP 응답 정보가 없습니다 (네트워크 연결 문제일 수 있음)")
		}
		return err
	}

	// defer로 연결 정리 보장
	defer func() {
		fmt.Println("🔌 WebSocket 연결 종료 중...")
		if closeErr := conn.Close(); closeErr != nil {
			fmt.Printf("⚠️  연결 종료 중 에러: %v\n", closeErr)
		} else {
			fmt.Println("✅ WebSocket 연결이 안전하게 종료되었습니다")
		}
	}()

	fmt.Println("✅ WebSocket 연결 성공!")
	fmt.Printf("📊 응답 상태: %d %s\n", resp.StatusCode, resp.Status)

	// 연결 후 주문 정보 요청 (Node.js 코드와 동일)
	fmt.Println("📤 주문 정보 요청 메시지 전송 중...")

	// UUID로 ticket 생성
	ticketUUID := uuid.New().String()
	fmt.Printf("🎫 생성된 ticket UUID: %s\n", ticketUUID)

	// JSON 배열 객체로 요청 메시지 생성 (업비트 공식 포맷)
	requestArray := []map[string]interface{}{
		{"ticket": nonce1},
		{"type": "myOrder"},
	}
	fmt.Printf(
		"token : %s, uuid : %s",
		jwtToken,
		ticketUUID,
	)
	// JSON으로 마샬링
	requestBytes, err := json.Marshal(requestArray)
	if err != nil {
		return fmt.Errorf("JSON 마샬링 실패: %v", err)
	}

	requestMessage := string(requestBytes)
	fmt.Printf("📋 생성된 JSON 배열:\n%s\n", requestMessage)

	if err := conn.WriteMessage(websocket.TextMessage, requestBytes); err != nil {
		return fmt.Errorf("메시지 전송 실패: %v", err)
	}

	fmt.Printf("✅ JSON 배열 메시지 전송 완료:\n%s\n", requestMessage)

	// Ping/Pong으로 연결 유지 설정
	fmt.Println("🏓 Ping/Pong 연결 유지 활성화...")
	conn.SetPingHandler(func(appData string) error {
		fmt.Printf("📡 Ping 수신: %s\n", appData)
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	conn.SetPongHandler(func(appData string) error {
		fmt.Printf("🏓 Pong 수신: %s\n", appData)
		return nil
	})

	// 즉시 첫 Ping 전송
	fmt.Println("📤 초기 Ping 전송 중...")
	if err := conn.WriteControl(websocket.PingMessage, []byte("initial"), time.Now().Add(5*time.Second)); err != nil {
		fmt.Printf("❌ 초기 Ping 전송 실패: %v\n", err)
	} else {
		fmt.Println("✅ 초기 Ping 전송 완료!")
	}

	// 15초마다 Ping 전송하는 고루틴 (30초 → 15초로 단축)
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	go func() {
		for {
			select {
			case <-pingTicker.C:
				fmt.Println("📤 15초 주기 Ping 전송 중...")
				if err := conn.WriteControl(websocket.PingMessage, []byte("keepalive"), time.Now().Add(5*time.Second)); err != nil {
					fmt.Printf("❌ Ping 전송 실패 (연결 끊어짐): %v\n", err)
					return
				}
				fmt.Println("✅ Ping 전송 완료!")
			}
		}
	}()

	// 무한 메시지 수신 대기 (ping/pong으로 연결 유지)
	fmt.Println("👂 무한 메시지 수신 대기 중... (Ctrl+C로 종료)")
	fmt.Println("🔄 15초마다 자동으로 Ping을 전송하여 연결을 유지합니다")
	fmt.Println("💡 업비트 서버가 연결을 끊지 않도록 적극적인 연결 유지 중...")

	messageCount := 0
	errorCount := 0
	maxErrors := 5          // 에러 허용 횟수 증가
	connectionLost := false // 연결 상태 추적

	for !connectionLost {
		// 에러가 너무 많이 발생하면 종료
		if errorCount >= maxErrors {
			fmt.Printf("❌ 최대 에러 수(%d) 초과 - 연결 종료\n", maxErrors)
			connectionLost = true
			break
		}

		// 10초 타임아웃으로 메시지 읽기 (ping/pong 간격보다 짧게)
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		messageType, message, err := conn.ReadMessage()

		if err != nil {
			errorCount++

			// 연결 종료 에러들을 체크
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				fmt.Printf("🔒 WebSocket 연결이 종료되었습니다: %v (총 %d개 메시지 수신)\n", err, messageCount)
				connectionLost = true
				break
			}

			if websocket.IsUnexpectedCloseError(err) {
				fmt.Printf("❌ 예상치 못한 연결 종료: %v\n", err)
				connectionLost = true
				break
			}

			// 타임아웃 에러는 무시하고 계속 대기 (ping/pong으로 연결 유지 중)
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				errorCount-- // 타임아웃은 에러 카운트에서 제외
				// 1분마다 한 번씩만 상태 메시지 출력
				if messageCount%6 == 0 { // 10초 * 6 = 1분
					fmt.Printf("💓 연결 유지 중... (현재까지 %d개 메시지 수신, 🏓 Ping/Pong 활성)\n", messageCount)
				}
				continue
			}

			// 다른 에러들 - 연결 문제일 가능성
			fmt.Printf("⚠️  메시지 읽기 에러 #%d: %v\n", errorCount, err)

			// 연속 에러 시 연결 상태 확인
			if errorCount >= 3 {
				fmt.Println("🔍 연결 상태 확인 중...")
				// 간단한 Ping 테스트
				if pingErr := conn.WriteControl(websocket.PingMessage, []byte("test"), time.Now().Add(3*time.Second)); pingErr != nil {
					fmt.Printf("❌ 연결이 끊어진 것 같습니다: %v\n", pingErr)
					connectionLost = true
					break
				}
			}

			// 잠시 대기 후 재시도
			time.Sleep(2 * time.Second)
			continue
		}

		// 성공적으로 메시지를 받았으면 에러 카운트 리셋
		errorCount = 0
		messageCount++
		fmt.Printf("\n🎉 메시지 %d 수신 (타입: %d, 크기: %d bytes) - %s:\n",
			messageCount, messageType, len(message), time.Now().Format("15:04:05"))

		// 메시지 타입별 처리
		switch messageType {
		case websocket.TextMessage:
			messageStr := string(message)
			fmt.Printf("📝 텍스트 메시지:\n%s\n", messageStr)

			// JSON인지 확인하고 포맷팅
			if strings.HasPrefix(strings.TrimSpace(messageStr), "{") {
				fmt.Println("🎯 JSON 형태의 데이터입니다!")
				// JSON 파싱 시도해서 타입 확인
				if strings.Contains(messageStr, `"type":"myAsset"`) {
					fmt.Println("💰 자산 정보 (myAsset) 데이터입니다!")
				} else if strings.Contains(messageStr, `"type":"myOrder"`) {
					fmt.Println("📋 주문 정보 (myOrder) 데이터입니다!")
				}
			}

		case websocket.BinaryMessage:
			fmt.Printf("📦 바이너리 메시지 (크기: %d bytes):\n", len(message))

			// 바이너리 데이터를 16진수로 출력 (처음 100바이트만)
			hexLimit := min(100, len(message))
			fmt.Printf("🔍 16진수 표현: %x\n", message[:hexLimit])
			if len(message) > hexLimit {
				fmt.Printf("   ... (총 %d bytes, %d bytes만 표시)\n", len(message), hexLimit)
			}

			// 바이너리 데이터를 문자열로 변환해서 확인
			if len(message) > 0 {
				messageStr := string(message)
				fmt.Printf("📄 문자열 변환:\n%s\n", messageStr)

				// JSON인지 확인
				if strings.HasPrefix(strings.TrimSpace(messageStr), "{") || strings.HasPrefix(strings.TrimSpace(messageStr), "[") {
					fmt.Println("🎯 JSON 형태의 데이터로 보입니다!")
					// JSON 파싱 시도해서 타입 확인
					if strings.Contains(messageStr, `"type":"myAsset"`) {
						fmt.Println("💰 자산 정보 (myAsset) 데이터입니다!")
					} else if strings.Contains(messageStr, `"type":"myOrder"`) {
						fmt.Println("📋 주문 정보 (myOrder) 데이터입니다!")
					}
				}
			}

		case websocket.PingMessage:
			fmt.Println("📡 서버로부터 Ping 메시지 수신")

		case websocket.PongMessage:
			fmt.Println("🏓 서버로부터 Pong 메시지 수신")

		default:
			fmt.Printf("❓ 알 수 없는 메시지 타입: %d\n", messageType)
		}

		fmt.Println("─────────────────────────────────────────")
	}

	fmt.Printf("🏁 WebSocket 세션 종료 (총 %d개 메시지 수신)\n", messageCount)
	return nil
}

// min 함수 (Go 1.21 이전 버전 호환성)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// executeCustomLogic 사용자 정의 로직을 실행합니다
func (a *App) executeCustomLogic() {
	fmt.Println("=== 사용자 정의 로직 실행 ===")

	// ========================================
	// 여기에 더 복잡한 나만의 코드를 작성하세요!
	// ========================================

	// 예시 1: 파일 시스템 작업
	fmt.Println("📁 홈 디렉토리 확인 중...")
	if homeDir, err := os.UserHomeDir(); err == nil {
		fmt.Printf("🏠 홈 디렉토리: %s\n", homeDir)
	}

	// 예시 2: 고급 HTTP 요청들
	fmt.Println("🌐 고급 네트워크 작업 실행 중...")
	go func() {
		a.performAdvancedHTTPRequests()
	}()

	// 예시 3: 데이터 처리
	fmt.Println("📊 데이터 처리 중...")
	numbers := []int{1, 2, 3, 4, 5}
	sum := 0
	for _, num := range numbers {
		sum += num
	}
	fmt.Printf("📈 합계: %d\n", sum)

	// 예시 4: 알림 표시
	a.fyneApp.SendNotification(&fyne.Notification{
		Title:   "특별한 기능",
		Content: "사용자 정의 로직이 성공적으로 실행되었습니다!",
	})

	fmt.Println("=== 사용자 정의 로직 완료 ===")
}

// performAdvancedHTTPRequests 고급 HTTP 요청들을 실행합니다
func (a *App) performAdvancedHTTPRequests() {
	fmt.Println("🚀 고급 HTTP 요청들 시작...")

	// 요청 1: JSON 응답 받기
	fmt.Println("1️⃣ JSON 데이터 요청 중...")
	resp1, err := http.Get("https://httpbin.org/json")
	if err != nil {
		fmt.Printf("❌ JSON 요청 실패: %v\n", err)
	} else {
		defer resp1.Body.Close()
		body1, _ := io.ReadAll(resp1.Body)
		fmt.Printf("✅ JSON 응답 받음 (크기: %d bytes)\n", len(body1))

		// JSON 일부 출력
		bodyStr := string(body1)
		if len(bodyStr) > 150 {
			bodyStr = bodyStr[:150] + "..."
		}
		fmt.Printf("📊 JSON 내용: %s\n", bodyStr)
	}

	// 요청 2: User-Agent 헤더와 함께 요청
	fmt.Println("2️⃣ 커스텀 헤더로 요청 중...")
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://httpbin.org/user-agent", nil)
	if err != nil {
		fmt.Printf("❌ 요청 생성 실패: %v\n", err)
	} else {
		req.Header.Set("User-Agent", "BitcoinTrader/1.0")
		resp2, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ 커스텀 헤더 요청 실패: %v\n", err)
		} else {
			defer resp2.Body.Close()
			body2, _ := io.ReadAll(resp2.Body)
			fmt.Printf("✅ 커스텀 헤더 응답: %s\n", string(body2))
		}
	}

	// 요청 3: POST 요청 (JSON 데이터 전송)
	fmt.Println("3️⃣ POST 요청으로 데이터 전송 중...")
	jsonData := `{"name": "BitcoinTrader", "type": "special_function", "timestamp": "` + time.Now().Format("2006-01-02T15:04:05Z") + `"}`
	resp3, err := http.Post("https://httpbin.org/post", "application/json", strings.NewReader(jsonData))
	if err != nil {
		fmt.Printf("❌ POST 요청 실패: %v\n", err)
	} else {
		defer resp3.Body.Close()
		fmt.Printf("✅ POST 요청 성공! 상태: %d\n", resp3.StatusCode)

		body3, _ := io.ReadAll(resp3.Body)
		bodyStr := string(body3)
		if len(bodyStr) > 300 {
			bodyStr = bodyStr[:300] + "..."
		}
		fmt.Printf("📝 POST 응답: %s\n", bodyStr)
	}

	// 요청 4: 실제 암호화폐 가격 API (선택사항)
	fmt.Println("4️⃣ 비트코인 가격 확인 중...")
	resp4, err := http.Get("https://api.coindesk.com/v1/bpi/currentprice.json")
	if err != nil {
		fmt.Printf("❌ 비트코인 가격 요청 실패: %v\n", err)
	} else {
		defer resp4.Body.Close()
		body4, _ := io.ReadAll(resp4.Body)
		fmt.Printf("💰 비트코인 가격 데이터 받음 (크기: %d bytes)\n", len(body4))

		// 간단한 파싱 (실제로는 JSON 파싱을 해야 함)
		bodyStr := string(body4)
		if strings.Contains(bodyStr, "USD") {
			fmt.Println("💵 USD 가격 정보가 포함되어 있습니다!")
		}
	}

	fmt.Println("🎯 모든 HTTP 요청 완료!")
}

// Run 애플리케이션을 실행합니다
func (a *App) Run() {
	a.ShowLoginScreen()
	a.fyneApp.Run()
}
