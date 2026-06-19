package platform

import (
	"fmt"
	"math"
	"strings"
)

// truncateToPrecision 주어진 값(value)을 소수점 precision 자리까지 "버림" 처리하여 문자열로 반환합니다.
// 예: precision=4, value=2.15132 -> "2.1513"
func truncateToPrecision(value float64, precision int) string {
	if precision <= 0 {
		return fmt.Sprintf("%.0f", math.Floor(value))
	}

	multiplier := math.Pow(10, float64(precision))
	truncated := math.Floor(value*multiplier) / multiplier
	return fmt.Sprintf("%."+fmt.Sprintf("%d", precision)+"f", truncated)
}

// countDecimalPlaces 문자열 형태의 숫자에서 의미 있는 소수 자릿수를 계산합니다.
// "0.001000" -> 3, "0.000100" -> 4, "1" -> 0
func countDecimalPlaces(s string) int {
	if s == "" {
		return 0
	}

	if !strings.Contains(s, ".") {
		return 0
	}

	parts := strings.SplitN(s, ".", 2)
	frac := strings.TrimRight(parts[1], "0")
	return len(frac)
}










