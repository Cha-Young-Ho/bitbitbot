package ui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"gui-app/models"
)

// ShowAddExchangeDialog 거래소 추가 대화상자를 표시합니다 (단순화)
func (a *App) ShowAddExchangeDialog() {
	fmt.Println("=== ShowAddExchangeDialog 시작 ===")

	// 지원하는 거래소 목록 가져오기
	exchanges := a.dataService.GetSupportedExchanges()
	fmt.Printf("지원하는 거래소 수: %d\n", len(exchanges))

	// 거래소 타입 선택 옵션 생성
	var options []string
	var exchangeMap = make(map[string]models.ExchangeInfo)

	for _, exchange := range exchanges {
		displayText := fmt.Sprintf("%s %s", exchange.Logo, exchange.DisplayName)
		options = append(options, displayText)
		exchangeMap[displayText] = exchange
		fmt.Printf("거래소 추가: %s -> %s\n", displayText, exchange.Type)
	}

	// 거래소 타입 선택
	typeSelect := widget.NewSelect(options, nil)
	typeSelect.SetSelected(options[0]) // 첫 번째 옵션을 기본 선택
	fmt.Printf("기본 선택: %s\n", options[0])

	// 입력 필드들
	aliasEntry := widget.NewEntry()
	aliasEntry.SetPlaceHolder("API 키 별칭 (예: 내 업비트 메인)")

	apiKeyEntry := widget.NewEntry()
	apiKeyEntry.SetPlaceHolder("API Key를 입력하세요")

	secretKeyEntry := widget.NewPasswordEntry()
	secretKeyEntry.SetPlaceHolder("Secret Key를 입력하세요")

	fmt.Println("입력 필드 생성 완료")

	// 현재 선택된 거래소 정보를 표시하는 라벨
	infoLabel := widget.NewLabel("")
	infoLabel.Wrapping = fyne.TextWrapWord

	// 거래소 타입 변경 시 정보 업데이트
	updateInfo := func(selectedOption string) {
		fmt.Printf("거래소 타입 변경: %s\n", selectedOption)
		if exchange, exists := exchangeMap[selectedOption]; exists {
			info := fmt.Sprintf("📊 %s\n🌐 %s\n💰 거래 수수료: %.2f%%",
				exchange.DisplayName,
				exchange.BaseURL,
				exchange.TradingFee*100)
			infoLabel.SetText(info)
			fmt.Println("거래소 정보 업데이트 완료")
		} else {
			fmt.Printf("거래소 정보를 찾을 수 없음: %s\n", selectedOption)
		}
	}

	// 초기 정보 설정
	fmt.Println("초기 정보 설정 중...")
	updateInfo(typeSelect.Selected)

	// 거래소 타입 변경 이벤트
	typeSelect.OnChanged = updateInfo

	// 폼 구성
	form := container.NewVBox(
		// 헤더
		widget.NewLabel("🔑 API 키 등록"),
		widget.NewSeparator(),

		// 거래소 선택
		widget.NewLabel("거래소 선택:"),
		typeSelect,
		infoLabel,
		widget.NewSeparator(),

		// 입력 필드들
		widget.NewLabel("별칭:"),
		aliasEntry,
		widget.NewLabel("API Key:"),
		apiKeyEntry,
		widget.NewLabel("Secret Key:"),
		secretKeyEntry,
	)
	fmt.Println("폼 구성 완료")

	// 등록 버튼
	var currentDialog dialog.Dialog // 대화상자 참조를 저장

	addBtn := NewPrimaryButton("🚀 등록", func() {
		fmt.Println("=== 등록 버튼 클릭 ===")

		// 입력 검증
		fmt.Printf("별칭: '%s'\n", aliasEntry.Text)
		if aliasEntry.Text == "" {
			fmt.Println("별칭이 비어있음 - 에러 표시")
			dialog.ShowError(fmt.Errorf("별칭을 입력해주세요"), a.mainWindow)
			return
		}

		fmt.Printf("API Key: '%s'\n", apiKeyEntry.Text)
		if apiKeyEntry.Text == "" {
			fmt.Println("API Key가 비어있음 - 에러 표시")
			dialog.ShowError(fmt.Errorf("API Key를 입력해주세요"), a.mainWindow)
			return
		}

		fmt.Printf("Secret Key 길이: %d\n", len(secretKeyEntry.Text))
		if secretKeyEntry.Text == "" {
			fmt.Println("Secret Key가 비어있음 - 에러 표시")
			dialog.ShowError(fmt.Errorf("Secret Key를 입력해주세요"), a.mainWindow)
			return
		}

		fmt.Printf("선택된 거래소: '%s'\n", typeSelect.Selected)
		selectedExchange, exists := exchangeMap[typeSelect.Selected]
		if !exists {
			fmt.Printf("거래소 정보를 찾을 수 없음: %s\n", typeSelect.Selected)
			dialog.ShowError(fmt.Errorf("거래소 정보를 찾을 수 없습니다"), a.mainWindow)
			return
		}
		fmt.Printf("거래소 타입: %s\n", selectedExchange.Type)

		// 거래소 등록 (메인 스레드에서 실행)
		fmt.Println("DataService.AddExchange 호출 시작...")
		if err := a.dataService.AddExchange(aliasEntry.Text, selectedExchange.Type, apiKeyEntry.Text, secretKeyEntry.Text); err != nil {
			fmt.Printf("AddExchange 실패: %v\n", err)
			dialog.ShowError(err, a.mainWindow)
			return
		}
		fmt.Println("DataService.AddExchange 성공!")

		// 성공 시 대화상자 닫기
		fmt.Println("대화상자 닫기 시도...")
		if currentDialog != nil {
			currentDialog.Hide()
			fmt.Println("대화상자 닫기 완료")
		} else {
			fmt.Println("currentDialog가 nil임")
		}

		fmt.Println("성공 메시지 표시...")
		dialog.ShowInformation("성공", fmt.Sprintf("API 키 '%s'이(가) 성공적으로 등록되었습니다!", aliasEntry.Text), a.mainWindow)

		fmt.Println("메인 화면 새로고침...")
		a.RefreshMainScreen()
		fmt.Println("=== 등록 완료 ===")
	})

	cancelBtn := NewSecondaryButton("취소", func() {
		fmt.Println("취소 버튼 클릭")
		// 대화상자 닫기는 자동으로 처리됨
	})

	buttons := container.NewHBox(cancelBtn, addBtn)
	content := container.NewBorder(nil, buttons, nil, nil, form)
	fmt.Println("대화상자 콘텐츠 구성 완료")

	// 대화상자 표시
	fmt.Println("대화상자 생성 중...")
	currentDialog = dialog.NewCustom("API 키 등록", "닫기", content, a.mainWindow)
	currentDialog.Resize(fyne.NewSize(500, 600))
	currentDialog.Show()
	fmt.Println("=== ShowAddExchangeDialog 완료 ===")
}

// ShowAddOrderDialog 주문 추가 대화상자를 표시합니다
func (a *App) ShowAddOrderDialog(exchangeID string) {
	// 거래소 정보 가져오기
	exchange, err := a.dataService.GetExchangeByID(exchangeID)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}

	// 입력 필드들
	amountEntry := widget.NewEntry()
	amountEntry.SetPlaceHolder("0.00000000")

	priceEntry := widget.NewEntry()
	priceEntry.SetPlaceHolder("0.00")

	// 폼 구성
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "📊 거래소", Widget: widget.NewLabel(exchange.Name)},
			{Text: "₿ 매도 수량 (BTC)", Widget: amountEntry},
			{Text: "💰 매도 가격 (KRW)", Widget: priceEntry},
		},
		OnSubmit: func() {
			// 입력 검증
			amount, err := strconv.ParseFloat(amountEntry.Text, 64)
			if err != nil || amount <= 0 {
				dialog.ShowError(fmt.Errorf("올바른 수량을 입력해주세요"), a.mainWindow)
				return
			}

			price, err := strconv.ParseFloat(priceEntry.Text, 64)
			if err != nil || price <= 0 {
				dialog.ShowError(fmt.Errorf("올바른 가격을 입력해주세요"), a.mainWindow)
				return
			}

			// 주문 추가
			if err := a.dataService.AddSellOrder(exchangeID, amount, price); err != nil {
				dialog.ShowError(err, a.mainWindow)
				return
			}

			dialog.ShowInformation("성공", "매도 주문이 추가되었습니다", a.mainWindow)
			a.RefreshMainScreen()
		},
		OnCancel: func() {
			// 취소 처리는 자동으로 됨
		},
		SubmitText: "주문 추가",
		CancelText: "취소",
	}

	// 대화상자 표시
	dialog.ShowForm("매도 주문 추가", "추가", "취소", form.Items, func(submitted bool) {
		if submitted {
			form.OnSubmit()
		}
	}, a.mainWindow)
}

// ShowEditOrderDialog 주문 수정 대화상자를 표시합니다
func (a *App) ShowEditOrderDialog(order models.SellOrder) {
	// 입력 필드들
	amountEntry := widget.NewEntry()
	amountEntry.SetText(fmt.Sprintf("%.8f", order.Amount))

	priceEntry := widget.NewEntry()
	priceEntry.SetText(fmt.Sprintf("%.2f", order.Price))

	// 폼 구성
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "📊 거래소", Widget: widget.NewLabel(order.ExchangeName)},
			{Text: "₿ 매도 수량 (BTC)", Widget: amountEntry},
			{Text: "💰 매도 가격 (KRW)", Widget: priceEntry},
		},
		OnSubmit: func() {
			// 입력 검증
			amount, err := strconv.ParseFloat(amountEntry.Text, 64)
			if err != nil || amount <= 0 {
				dialog.ShowError(fmt.Errorf("올바른 수량을 입력해주세요"), a.mainWindow)
				return
			}

			price, err := strconv.ParseFloat(priceEntry.Text, 64)
			if err != nil || price <= 0 {
				dialog.ShowError(fmt.Errorf("올바른 가격을 입력해주세요"), a.mainWindow)
				return
			}

			// 주문 수정
			if err := a.dataService.UpdateSellOrder(order.ID, amount, price); err != nil {
				dialog.ShowError(err, a.mainWindow)
				return
			}

			dialog.ShowInformation("성공", "매도 주문이 수정되었습니다", a.mainWindow)
			a.RefreshMainScreen()
		},
		OnCancel: func() {
			// 취소 처리는 자동으로 됨
		},
		SubmitText: "주문 수정",
		CancelText: "취소",
	}

	// 대화상자 표시
	dialog.ShowForm("매도 주문 수정", "수정", "취소", form.Items, func(submitted bool) {
		if submitted {
			form.OnSubmit()
		}
	}, a.mainWindow)
}

// ShowManageOrdersDialog 주문 관리 대화상자를 표시합니다
func (a *App) ShowManageOrdersDialog(exchangeID string) {
	// 거래소 정보 가져오기
	exchange, err := a.dataService.GetExchangeByID(exchangeID)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}

	// 주문 목록 가져오기
	orders := a.dataService.GetSellOrdersByExchange(exchangeID)

	// 주문 목록 표시
	var orderCards []fyne.CanvasObject

	if len(orders) == 0 {
		emptyState := NewEmptyState(
			"등록된 주문이 없습니다.",
			"주문 추가",
			func() {
				a.ShowAddOrderDialog(exchangeID)
			},
		)
		orderCards = append(orderCards, emptyState)
	} else {
		for _, order := range orders {
			card := NewOrderCard(
				order,
				func(order models.SellOrder) { a.ShowEditOrderDialog(order) },
				func(orderID string) { a.handleDeleteOrder(orderID) },
			)
			orderCards = append(orderCards, card)
		}
	}

	// 헤더 정보
	header := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("🔑 %s 주문 관리", exchange.Name)),
		widget.NewSeparator(),
	)

	// 액션 버튼들
	addOrderBtn := NewActionButton("주문 추가", "➕", widget.HighImportance, func() {
		a.ShowAddOrderDialog(exchangeID)
	})

	actions := container.NewHBox(addOrderBtn)

	// 콘텐츠 구성
	ordersList := container.NewVBox(orderCards...)
	scroll := container.NewScroll(ordersList)
	scroll.SetMinSize(fyne.NewSize(500, 400))

	content := container.NewVBox(
		header,
		actions,
		widget.NewSeparator(),
		scroll,
	)

	// 대화상자 표시
	d := dialog.NewCustom("주문 관리", "닫기", content, a.mainWindow)
	d.Resize(fyne.NewSize(600, 500))
	d.Show()
}
