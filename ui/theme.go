package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ModernTheme kmong 스타일의 모던한 테마
type ModernTheme struct{}

// NewModernTheme 새로운 모던 테마를 생성합니다
func NewModernTheme() fyne.Theme {
	return &ModernTheme{}
}

// Color 테마 색상을 정의합니다
func (m *ModernTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{18, 32, 25, 255} // 어두운 초록 배경
	case theme.ColorNameForeground:
		return color.RGBA{220, 255, 235, 255} // 밝은 초록빛 텍스트
	case theme.ColorNameButton:
		return color.RGBA{34, 60, 45, 255} // 어두운 초록 버튼
	case theme.ColorNamePrimary:
		return color.RGBA{52, 211, 153, 255} // 밝은 초록색 (primary)
	case theme.ColorNameSuccess:
		return color.RGBA{34, 197, 94, 255} // 성공 초록색
	case theme.ColorNameWarning:
		return color.RGBA{251, 191, 36, 255} // 노란색 (경고)
	case theme.ColorNameError:
		return color.RGBA{248, 113, 113, 255} // 빨간색 (오류)
	case theme.ColorNameHover:
		return color.RGBA{55, 90, 70, 255} // 호버 시 더 밝은 초록
	case theme.ColorNameFocus:
		return color.RGBA{52, 211, 153, 255} // 포커스 초록색
	case theme.ColorNameShadow:
		return color.RGBA{0, 0, 0, 80} // 그림자
	case theme.ColorNameInputBackground:
		return color.RGBA{30, 55, 40, 255} // 입력 필드 배경
	case theme.ColorNameHeaderBackground:
		return color.RGBA{22, 40, 30, 255} // 헤더 배경
	case theme.ColorNameMenuBackground:
		return color.RGBA{25, 45, 35, 255} // 메뉴 배경
	case theme.ColorNameOverlayBackground:
		return color.RGBA{0, 0, 0, 120} // 오버레이 배경
	case theme.ColorNameSeparator:
		return color.RGBA{80, 140, 100, 180} // 밝은 초록 구분선
	}
	return theme.DefaultTheme().Color(name, variant)
}

// Font 폰트를 정의합니다
func (m *ModernTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Icon 아이콘을 정의합니다
func (m *ModernTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size 크기를 정의합니다
func (m *ModernTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNamePadding:
		return 12 // 패딩 증가
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameScrollBar:
		return 8
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameSeparatorThickness:
		return 2 // 구분선 두께 증가
	case theme.SizeNameInputBorder:
		return 2
	case theme.SizeNameInputRadius:
		return 8
	}
	return theme.DefaultTheme().Size(name)
}

// 커스텀 색상 상수
var (
	ColorPurple    = color.RGBA{74, 158, 107, 255}  // 초록빛 보라색
	ColorIndigo    = color.RGBA{59, 142, 101, 255}  // 초록빛 인디고
	ColorTeal      = color.RGBA{45, 184, 136, 255}  // 청록색
	ColorGray100   = color.RGBA{40, 70, 55, 255}    // 어두운 초록 회색 (밝음)
	ColorGray200   = color.RGBA{35, 60, 48, 255}    // 어두운 초록 회색
	ColorGray300   = color.RGBA{30, 50, 40, 255}    // 더 어두운 초록 회색
	ColorGray700   = color.RGBA{20, 35, 28, 255}    // 매우 어두운 초록 회색
	ColorGray800   = color.RGBA{15, 25, 20, 255}    // 거의 검은 초록색
	ColorWhite     = color.RGBA{220, 255, 235, 255} // 밝은 초록빛 흰색
	ColorBlue50    = color.RGBA{40, 70, 55, 255}    // 연한 초록색
	ColorBlue500   = color.RGBA{52, 211, 153, 255}  // 기본 밝은 초록색
	ColorGreen50   = color.RGBA{35, 65, 50, 255}    // 연한 초록색
	ColorGreen500  = color.RGBA{34, 197, 94, 255}   // 기본 초록색
	ColorYellow50  = color.RGBA{60, 75, 50, 255}    // 연한 초록빛 노란색
	ColorYellow500 = color.RGBA{251, 191, 36, 255}  // 기본 노란색
)
