package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Min 두 정수 중 작은 값을 반환합니다
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max 두 정수 중 큰 값을 반환합니다
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FormatBTC BTC 수량을 포맷팅합니다
func FormatBTC(amount float64) string {
	return fmt.Sprintf("%.8f", amount)
}

// FormatKRW 원화를 포맷팅합니다
func FormatKRW(price float64) string {
	return fmt.Sprintf("₩ %,.0f", price)
}

// FormatDateTime 시간을 포맷팅합니다
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatDate 날짜를 포맷팅합니다
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// ValidateAmount BTC 수량 유효성을 검증합니다
func ValidateAmount(amountStr string) (float64, error) {
	if amountStr == "" {
		return 0, fmt.Errorf("수량을 입력해주세요")
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("올바른 수량을 입력해주세요")
	}

	if amount <= 0 {
		return 0, fmt.Errorf("수량은 0보다 커야 합니다")
	}

	// 소수점 15자리 검증
	if strings.Contains(amountStr, ".") {
		parts := strings.Split(amountStr, ".")
		if len(parts) > 1 && len(parts[1]) > 15 {
			return 0, fmt.Errorf("소수점은 15자리까지만 입력 가능합니다")
		}
	}

	return amount, nil
}

// ValidatePrice 가격 유효성을 검증합니다
func ValidatePrice(priceStr string) (float64, error) {
	if priceStr == "" {
		return 0, fmt.Errorf("가격을 입력해주세요")
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, fmt.Errorf("올바른 가격을 입력해주세요")
	}

	if price <= 0 {
		return 0, fmt.Errorf("가격은 0보다 커야 합니다")
	}

	return price, nil
}

// GenerateID 간단한 ID를 생성합니다
func GenerateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// TruncateString 문자열을 자릅니다
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// MaskAPIKey API Key를 마스킹합니다
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}

	start := apiKey[:4]
	end := apiKey[len(apiKey)-4:]
	middle := strings.Repeat("*", len(apiKey)-8)

	return start + middle + end
}
