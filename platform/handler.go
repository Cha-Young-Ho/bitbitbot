package platform

import (
	"bitbit-app/local_file"
	"fmt"
	"log"
	"strings"
)

type Handler struct {
	localFileHandler *local_file.Handler
}

// Platform enum
type Platform string

const (
	Upbit    Platform = "Upbit"
	Bithumb  Platform = "Bithumb"
	Coinone  Platform = "Coinone"
	Korbit   Platform = "Korbit"
	Binance  Platform = "Binance"
	Bybit    Platform = "Bybit"
	Bitget   Platform = "Bitget"
	Huobi    Platform = "Huobi"
	Mexc     Platform = "Mexc"
	KuCoin   Platform = "KuCoin"
	Coinbase Platform = "Coinbase"
)

func NewHandler(localFileHandler *local_file.Handler) *Handler {
	return &Handler{
		localFileHandler: localFileHandler,
	}
}

// GetPlatformInfo 사용자의 플랫폼 정보를 조회합니다
func (h *Handler) GetPlatformInfo(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 비어있습니다.",
		}
	}

	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	return map[string]interface{}{
		"success":   true,
		"platforms": userData.PlatformKeyList,
	}
}

// AddPlatform 사용자에게 플랫폼을 추가합니다
func (h *Handler) AddPlatform(userID string, platformName string, name string, accessKey, secretKey string) map[string]interface{} {
	// 입력값 정리
	userID = strings.TrimSpace(userID)
	platformName = strings.TrimSpace(platformName)
	name = strings.TrimSpace(name)
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)

	// 입력값 검증
	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 비어있습니다.",
		}
	}

	if platformName == "" {
		return map[string]interface{}{
			"success": false,
			"message": "플랫폼 이름이 비어있습니다.",
		}
	}

	if name == "" {
		return map[string]interface{}{
			"success": false,
			"message": "별칭을 입력해주세요.",
		}
	}

	// 사용자 조회
	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	// 기존 플랫폼 확인 (별칭으로 구분)
	for _, existingPlatform := range userData.PlatformKeyList {
		if existingPlatform.PlatformName == platformName && existingPlatform.Name == name {
			return map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("이미 존재하는 플랫폼 별칭입니다: %s - %s", platformName, name),
			}
		}
	}

	// 새 플랫폼 키 추가
	newPlatformKey := local_file.PlatformKey{
		PlatformName:      platformName,
		Name:              name,
		PlatformAccessKey: accessKey,
		PlatformSecretKey: secretKey,
	}

	userData.PlatformKeyList = append(userData.PlatformKeyList, newPlatformKey)

	// 사용자 데이터 업데이트
	err = h.localFileHandler.UpdateUser(*userData)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	log.Printf("플랫폼 추가 완료: userID=%s, platform=%s, name=%s", userID, platformName, name)
	return map[string]interface{}{
		"success":  true,
		"message":  "플랫폼이 추가되었습니다.",
		"platform": newPlatformKey,
	}
}

// RemovePlatform 사용자에서 플랫폼을 제거합니다
func (h *Handler) RemovePlatform(userID string, platformName string, name string) map[string]interface{} {
	// 입력값 정리
	userID = strings.TrimSpace(userID)
	platformName = strings.TrimSpace(platformName)
	name = strings.TrimSpace(name)

	if userID == "" || platformName == "" || name == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID, 플랫폼 이름, 별칭이 필요합니다.",
		}
	}

	// 사용자 조회
	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	// 플랫폼 제거
	newPlatformKeys := []local_file.PlatformKey{}
	found := false
	for _, platform := range userData.PlatformKeyList {
		if platform.PlatformName == platformName && platform.Name == name {
			found = true
		} else {
			newPlatformKeys = append(newPlatformKeys, platform)
		}
	}

	if !found {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("플랫폼을 찾을 수 없습니다: %s - %s", platformName, name),
		}
	}

	userData.PlatformKeyList = newPlatformKeys

	// 사용자 데이터 업데이트
	err = h.localFileHandler.UpdateUser(*userData)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	log.Printf("플랫폼 제거 완료: userID=%s, platform=%s, name=%s", userID, platformName, name)
	return map[string]interface{}{
		"success": true,
		"message": "플랫폼이 제거되었습니다.",
	}
}

// UpdatePlatform 사용자의 플랫폼 정보를 업데이트합니다
func (h *Handler) UpdatePlatform(userID string, oldPlatformName string, oldName string, newPlatformName string, newName string, accessKey, secretKey string) map[string]interface{} {
	// 입력값 정리
	userID = strings.TrimSpace(userID)
	oldPlatformName = strings.TrimSpace(oldPlatformName)
	oldName = strings.TrimSpace(oldName)
	newPlatformName = strings.TrimSpace(newPlatformName)
	newName = strings.TrimSpace(newName)
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)

	// 입력값 검증
	if userID == "" || newPlatformName == "" || newName == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID, 플랫폼 이름, 별칭이 필요합니다.",
		}
	}

	// 사용자 조회
	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	// 기존 플랫폼 찾기 (oldPlatformName과 oldName으로 찾기)
	found := false
	for i, existingPlatform := range userData.PlatformKeyList {
		if existingPlatform.PlatformName == oldPlatformName && existingPlatform.Name == oldName {
			// 기존 항목 업데이트
			userData.PlatformKeyList[i] = local_file.PlatformKey{
				PlatformName:      newPlatformName,
				Name:              newName,
				PlatformAccessKey: accessKey,
				PlatformSecretKey: secretKey,
			}
			found = true
			break
		}
	}

	if !found {
		// 기존 항목이 없으면 새로 추가
		newPlatformKey := local_file.PlatformKey{
			PlatformName:      newPlatformName,
			Name:              newName,
			PlatformAccessKey: accessKey,
			PlatformSecretKey: secretKey,
		}
		userData.PlatformKeyList = append(userData.PlatformKeyList, newPlatformKey)
	}

	// 사용자 데이터 업데이트
	err = h.localFileHandler.UpdateUser(*userData)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	if found {
		log.Printf("플랫폼 업데이트 완료: userID=%s, oldPlatform=%s, oldName=%s, newPlatform=%s, newName=%s", userID, oldPlatformName, oldName, newPlatformName, newName)
		return map[string]interface{}{
			"success": true,
			"message": "플랫폼이 업데이트되었습니다.",
		}
	} else {
		log.Printf("플랫폼 추가 완료: userID=%s, platform=%s, name=%s", userID, newPlatformName, newName)
		return map[string]interface{}{
			"success": true,
			"message": "플랫폼이 추가되었습니다.",
		}
	}
}

// GetAllPlatforms 지원되는 모든 플랫폼 목록을 반환합니다
func (h *Handler) GetAllPlatforms() map[string]interface{} {
	platforms := []map[string]interface{}{
		{"name": "Upbit", "value": "Upbit"},
		{"name": "Bithumb", "value": "Bithumb"},
		{"name": "Coinone", "value": "Coinone"},
		{"name": "Korbit", "value": "Korbit"},
		{"name": "Binance", "value": "Binance"},
		{"name": "Bybit", "value": "Bybit"},
		{"name": "Bitget", "value": "Bitget"},
		{"name": "Huobi", "value": "Huobi"},
		{"name": "Mexc", "value": "Mexc"},
		{"name": "KuCoin", "value": "KuCoin"},
		{"name": "Coinbase", "value": "Coinbase"},
	}

	return map[string]interface{}{
		"success":   true,
		"platforms": platforms,
	}
}

// GetPlatforms 사용자의 플랫폼 목록을 반환합니다 (별칭)
func (h *Handler) GetPlatforms(userID string) map[string]interface{} {
	return h.GetPlatformInfo(userID)
}

// AddSellOrder 사용자에게 예약 매도 주문을 추가합니다
func (h *Handler) AddSellOrder(userID string, orderName string, symbol string, price float64, quantity float64, term float64, platformName string, platformNickName string) map[string]interface{} {
	// 입력값 정리
	userID = strings.TrimSpace(userID)
	orderName = strings.TrimSpace(orderName)
	symbol = strings.TrimSpace(symbol)
	platformName = strings.TrimSpace(platformName)
	platformNickName = strings.TrimSpace(platformNickName)

	// 입력값 검증
	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 비어있습니다.",
		}
	}

	if orderName == "" {
		return map[string]interface{}{
			"success": false,
			"message": "매도 주문 별칭을 입력해주세요.",
		}
	}

	if symbol == "" {
		return map[string]interface{}{
			"success": false,
			"message": "심볼을 입력해주세요.",
		}
	}

	if price <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "목표가는 0보다 커야 합니다.",
		}
	}

	if quantity <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "수량은 0보다 커야 합니다.",
		}
	}

	if term <= 0 {
		return map[string]interface{}{
			"success": false,
			"message": "주기는 0보다 커야 합니다.",
		}
	}

	// 사용자 조회
	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	// 매도 주문 별칭 중복 검증
	for _, existingOrder := range userData.SellOrderList {
		if existingOrder.Name == orderName {
			return map[string]interface{}{
				"success": false,
				"message": "이미 존재하는 매도 주문 별칭입니다.",
			}
		}
	}

	// 새로운 예약 매도 주문 생성
	newSellOrder := local_file.SellOrder{
		Name:             orderName,
		Symbol:           symbol,
		Price:            price,
		Quantity:         quantity,
		Term:             term,
		Platform:         platformName,
		PlatformNickName: platformNickName,
	}

	// 사용자 데이터에 예약 매도 주문 추가
	userData.SellOrderList = append(userData.SellOrderList, newSellOrder)

	// 사용자 데이터 업데이트
	err = h.localFileHandler.UpdateUser(*userData)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	log.Printf("예약 매도 주문 추가 완료: userID=%s, orderName=%s, platform=%s", userID, orderName, platformName)
	return map[string]interface{}{
		"success": true,
		"message": "예약 매도 주문이 추가되었습니다.",
	}
}

// GetSellOrders 사용자의 예약 매도 주문 목록을 조회합니다
func (h *Handler) GetSellOrders(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 비어있습니다.",
		}
	}

	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}
	return map[string]interface{}{
		"success":    true,
		"sellOrders": userData.SellOrderList,
	}
}
