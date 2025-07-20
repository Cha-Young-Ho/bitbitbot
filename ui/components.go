package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"gui-app/models"
	"gui-app/utils"
)

// SimpleExchangeCard 간단한 거래소 카드 컴포넌트
type SimpleExchangeCard struct {
	widget.BaseWidget
	exchange     models.Exchange
	exchangeInfo models.ExchangeInfo
	orderCount   int
	onSellOrder  func(string)
	onManage     func(string)
}

// NewSimpleExchangeCard 새로운 간단한 거래소 카드를 생성합니다
func NewSimpleExchangeCard(exchange models.Exchange, exchangeInfo models.ExchangeInfo, orderCount int, onSellOrder, onManage func(string)) *SimpleExchangeCard {
	card := &SimpleExchangeCard{
		exchange:     exchange,
		exchangeInfo: exchangeInfo,
		orderCount:   orderCount,
		onSellOrder:  onSellOrder,
		onManage:     onManage,
	}
	card.ExtendBaseWidget(card)
	return card
}

// CreateRenderer 간단한 거래소 카드 렌더러를 생성합니다
func (e *SimpleExchangeCard) CreateRenderer() fyne.WidgetRenderer {
	// 거래소 정보
	nameLabel := widget.NewLabel(fmt.Sprintf("%s %s", e.exchangeInfo.Logo, e.exchangeInfo.DisplayName))
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	aliasLabel := widget.NewLabel(fmt.Sprintf("📝 별칭: %s", e.exchange.Name))

	apiKeyLabel := widget.NewLabel(fmt.Sprintf("🔑 API Key: %s", utils.MaskAPIKey(e.exchange.APIKey)))

	orderLabel := widget.NewLabel(fmt.Sprintf("📋 주문: %d개", e.orderCount))

	statusLabel := widget.NewLabel("🟢 활성")
	if !e.exchange.IsActive {
		statusLabel.SetText("🔴 비활성")
	}

	// 버튼들
	sellBtn := widget.NewButton("💰 매도 주문", func() {
		if e.onSellOrder != nil {
			e.onSellOrder(e.exchange.ID)
		}
	})
	sellBtn.Importance = widget.HighImportance

	manageBtn := widget.NewButton("⚙️ 관리", func() {
		if e.onManage != nil {
			e.onManage(e.exchange.ID)
		}
	})

	buttonContainer := container.NewHBox(sellBtn, manageBtn)

	content := container.NewVBox(
		nameLabel,
		aliasLabel,
		apiKeyLabel,
		orderLabel,
		statusLabel,
		widget.NewSeparator(),
		buttonContainer,
	)

	// 테두리가 있는 카드로 감싸기
	card := widget.NewCard("", "", content)

	return widget.NewSimpleRenderer(card)
}

// StatsCard 통계 카드 컴포넌트
type StatsCard struct {
	widget.BaseWidget
	title string
	value string
	icon  string
	color color.RGBA
}

// NewStatsCard 새로운 통계 카드를 생성합니다
func NewStatsCard(title, value, icon string, bgColor color.RGBA) *StatsCard {
	card := &StatsCard{
		title: title,
		value: value,
		icon:  icon,
		color: bgColor,
	}
	card.ExtendBaseWidget(card)
	return card
}

// CreateRenderer 카드 렌더러를 생성합니다
func (s *StatsCard) CreateRenderer() fyne.WidgetRenderer {
	iconLabel := widget.NewLabel(s.icon)
	iconLabel.TextStyle = fyne.TextStyle{Bold: true}
	iconLabel.Alignment = fyne.TextAlignCenter

	titleLabel := widget.NewLabel(s.title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: false}
	titleLabel.Alignment = fyne.TextAlignCenter

	valueLabel := widget.NewLabel(s.value)
	valueLabel.TextStyle = fyne.TextStyle{Bold: true}
	valueLabel.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		iconLabel,
		titleLabel,
		valueLabel,
	)

	// 테두리가 있는 카드로 감싸기
	card := widget.NewCard("", "", content)

	return widget.NewSimpleRenderer(card)
}

// UpdateValue 값을 업데이트합니다
func (s *StatsCard) UpdateValue(value string) {
	s.value = value
	s.Refresh()
}

// ExchangeCard 거래소 카드 컴포넌트
type ExchangeCard struct {
	widget.BaseWidget
	exchange    models.Exchange
	orderCount  int
	onSellOrder func(string)
	onManage    func(string)
}

// NewExchangeCard 새로운 거래소 카드를 생성합니다
func NewExchangeCard(exchange models.Exchange, orderCount int, onSellOrder, onManage func(string)) *ExchangeCard {
	card := &ExchangeCard{
		exchange:    exchange,
		orderCount:  orderCount,
		onSellOrder: onSellOrder,
		onManage:    onManage,
	}
	card.ExtendBaseWidget(card)
	return card
}

// CreateRenderer 거래소 카드 렌더러를 생성합니다
func (e *ExchangeCard) CreateRenderer() fyne.WidgetRenderer {
	nameLabel := widget.NewLabel(fmt.Sprintf("🏢 %s", e.exchange.Name))
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	apiKeyLabel := widget.NewLabel(fmt.Sprintf("🔑 %s", utils.MaskAPIKey(e.exchange.APIKey)))

	orderLabel := widget.NewLabel(fmt.Sprintf("📋 주문: %d개", e.orderCount))

	statusLabel := widget.NewLabel("🟢 활성")
	if !e.exchange.IsActive {
		statusLabel.SetText("🔴 비활성")
	}

	sellBtn := widget.NewButton("💰 매도 주문", func() {
		if e.onSellOrder != nil {
			e.onSellOrder(e.exchange.ID)
		}
	})
	sellBtn.Importance = widget.HighImportance

	manageBtn := widget.NewButton("⚙️ 관리", func() {
		if e.onManage != nil {
			e.onManage(e.exchange.ID)
		}
	})

	buttonContainer := container.NewHBox(sellBtn, manageBtn)

	content := container.NewVBox(
		nameLabel,
		apiKeyLabel,
		orderLabel,
		statusLabel,
		widget.NewSeparator(),
		buttonContainer,
	)

	// 테두리가 있는 카드로 감싸기
	card := widget.NewCard("", "", content)

	return widget.NewSimpleRenderer(card)
}

// OrderCard 주문 카드 컴포넌트
type OrderCard struct {
	widget.BaseWidget
	order    models.SellOrder
	onEdit   func(models.SellOrder)
	onDelete func(string)
}

// NewOrderCard 새로운 주문 카드를 생성합니다
func NewOrderCard(order models.SellOrder, onEdit func(models.SellOrder), onDelete func(string)) *OrderCard {
	card := &OrderCard{
		order:    order,
		onEdit:   onEdit,
		onDelete: onDelete,
	}
	card.ExtendBaseWidget(card)
	return card
}

// CreateRenderer 주문 카드 렌더러를 생성합니다
func (o *OrderCard) CreateRenderer() fyne.WidgetRenderer {
	exchangeLabel := widget.NewLabel(fmt.Sprintf("🏢 %s", o.order.ExchangeName))
	exchangeLabel.TextStyle = fyne.TextStyle{Bold: true}

	amountLabel := widget.NewLabel(fmt.Sprintf("₿ %s BTC", utils.FormatBTC(o.order.Amount)))
	priceLabel := widget.NewLabel(utils.FormatKRW(o.order.Price))
	dateLabel := widget.NewLabel(fmt.Sprintf("📅 %s", utils.FormatDateTime(o.order.CreatedAt)))

	statusLabel := widget.NewLabel("🟢 활성")
	if o.order.Status != "active" {
		statusLabel.SetText("🔴 비활성")
	}

	editBtn := widget.NewButton("✏️ 수정", func() {
		if o.onEdit != nil {
			o.onEdit(o.order)
		}
	})

	deleteBtn := widget.NewButton("🗑️ 삭제", func() {
		if o.onDelete != nil {
			o.onDelete(o.order.ID)
		}
	})

	buttonContainer := container.NewHBox(editBtn, deleteBtn)

	content := container.NewVBox(
		exchangeLabel,
		amountLabel,
		priceLabel,
		dateLabel,
		statusLabel,
		widget.NewSeparator(),
		buttonContainer,
	)

	// 테두리가 있는 카드로 감싸기
	card := widget.NewCard("", "", content)

	return widget.NewSimpleRenderer(card)
}

// SectionCard 섹션 카드 컴포넌트
func NewSectionCard(title string, content fyne.CanvasObject) *fyne.Container {
	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	// 테두리용 색상 사각형
	borderRect := widget.NewCard("", "", content)

	// 헤더 부분
	header := container.NewBorder(nil, nil, titleLabel, nil)

	// 두꺼운 구분선
	separator := widget.NewSeparator()

	card := container.NewVBox(
		container.NewPadded(header),
		separator,
		container.NewPadded(borderRect),
	)

	// 외부 테두리 컨테이너
	return container.NewBorder(nil, nil, nil, nil, card)
}

// ActionButton 액션 버튼 컴포넌트
func NewActionButton(text string, icon string, importance widget.ButtonImportance, action func()) *widget.Button {
	btn := widget.NewButton(fmt.Sprintf("%s %s", icon, text), action)
	btn.Importance = importance
	return btn
}

// SearchEntry 검색 입력 필드 컴포넌트
func NewSearchEntry(placeholder string, onChanged func(string)) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetPlaceHolder(placeholder)
	entry.OnChanged = onChanged
	return entry
}

// InfoLabel 정보 라벨 컴포넌트
func NewInfoLabel(text string, alignment fyne.TextAlign) *widget.Label {
	label := widget.NewLabel(text)
	label.Alignment = alignment
	label.Wrapping = fyne.TextWrapWord
	return label
}

// PrimaryButton 주요 버튼 컴포넌트
func NewPrimaryButton(text string, action func()) *widget.Button {
	btn := widget.NewButton(text, action)
	btn.Importance = widget.HighImportance
	return btn
}

// SecondaryButton 보조 버튼 컴포넌트
func NewSecondaryButton(text string, action func()) *widget.Button {
	btn := widget.NewButton(text, action)
	btn.Importance = widget.MediumImportance
	return btn
}

// LoadingIndicator 로딩 인디케이터 컴포넌트
func NewLoadingIndicator(message string) *fyne.Container {
	progressBar := widget.NewProgressBarInfinite()
	label := widget.NewLabel(message)
	label.Alignment = fyne.TextAlignCenter

	return container.NewVBox(
		progressBar,
		label,
	)
}

// EmptyState 빈 상태 컴포넌트
func NewEmptyState(message, actionText string, action func()) *fyne.Container {
	icon := widget.NewLabel("📭")
	icon.Alignment = fyne.TextAlignCenter
	icon.TextStyle = fyne.TextStyle{Bold: true}

	messageLabel := widget.NewLabel(message)
	messageLabel.Alignment = fyne.TextAlignCenter
	messageLabel.Wrapping = fyne.TextWrapWord

	var content []fyne.CanvasObject
	content = append(content, icon, messageLabel)

	if actionText != "" && action != nil {
		actionBtn := NewPrimaryButton(actionText, action)
		content = append(content, actionBtn)
	}

	return container.NewVBox(content...)
}
