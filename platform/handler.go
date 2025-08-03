package platform

import (
	"bitbit-app/local_file"
	"fmt"
	"log"
	"strings"
)

type Handler struct {
	localFileHandler *local_file.Handler
	workerManager    *WorkerManager
	workerFactory    *WorkerFactory
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
	workerManager := NewWorkerManager()
	workerFactory := NewWorkerFactory(workerManager)

	return &Handler{
		localFileHandler: localFileHandler,
		workerManager:    workerManager,
		workerFactory:    workerFactory,
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

// StartWorkerForOrder 특정 주문에 대한 워커를 시작합니다
func (h *Handler) StartWorkerForOrder(userID string, orderName string) map[string]interface{} {
	userID = strings.TrimSpace(userID)
	orderName = strings.TrimSpace(orderName)

	if userID == "" || orderName == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID와 주문 이름이 필요합니다.",
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

	// 주문 찾기
	var targetOrder local_file.SellOrder
	found := false
	for _, order := range userData.SellOrderList {
		if order.Name == orderName {
			targetOrder = order
			found = true
			break
		}
	}

	if !found {
		return map[string]interface{}{
			"success": false,
			"message": "주문을 찾을 수 없습니다.",
		}
	}

	// 플랫폼 키 찾기
	var platformKey local_file.PlatformKey
	found = false
	for _, key := range userData.PlatformKeyList {
		if key.PlatformName == targetOrder.Platform && key.Name == targetOrder.PlatformNickName {
			platformKey = key
			found = true
			break
		}
	}

	if !found {
		return map[string]interface{}{
			"success": false,
			"message": "플랫폼 키를 찾을 수 없습니다.",
		}
	}

	// 워커 생성
	worker, err := h.workerFactory.CreateWorker(targetOrder, platformKey.PlatformAccessKey, platformKey.PlatformSecretKey)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 생성 실패: %v", err),
		}
	}

	// 워커 매니저에 추가
	if err := h.workerManager.AddWorker(orderName, userID, worker); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 추가 실패: %v", err),
		}
	}

	// 워커 시작
	if err := h.workerManager.StartWorker(orderName); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 시작 실패: %v", err),
		}
	}

	log.Printf("워커 시작 완료: userID=%s, orderName=%s, platform=%s", userID, orderName, targetOrder.Platform)
	return map[string]interface{}{
		"success": true,
		"message": "워커가 시작되었습니다.",
	}
}

// StopWorkerForOrder 특정 주문에 대한 워커를 중지합니다
func (h *Handler) StopWorkerForOrder(userID string, orderName string) map[string]interface{} {
	userID = strings.TrimSpace(userID)
	orderName = strings.TrimSpace(orderName)

	if userID == "" || orderName == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID와 주문 이름이 필요합니다.",
		}
	}

	// 워커 중지
	if err := h.workerManager.StopWorker(orderName); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 중지 실패: %v", err),
		}
	}

	// 워커 제거
	if err := h.workerManager.RemoveWorker(orderName); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("워커 제거 실패: %v", err),
		}
	}

	log.Printf("워커 중지 완료: userID=%s, orderName=%s", userID, orderName)
	return map[string]interface{}{
		"success": true,
		"message": "워커가 중지되었습니다.",
	}
}

// GetWorkerStatus 모든 워커의 상태를 반환합니다
func (h *Handler) GetWorkerStatus(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	status := h.workerManager.GetWorkerStatus()
	return map[string]interface{}{
		"success": true,
		"status":  status,
	}
}

// GetWorkerLogs 워커 로그를 반환합니다
func (h *Handler) GetWorkerLogs(userID string, limit int) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	if limit <= 0 {
		limit = 100 // 기본값
	}

	// 로그 채널에서 로그 수집
	logs := []WorkerLog{}
	logChan := h.workerManager.GetLogChannel()

	// 최근 로그들을 수집
	for i := 0; i < limit; i++ {
		select {
		case log := <-logChan:
			logs = append(logs, log)
		default:
			// 더 이상 로그가 없으면 종료
			goto done
		}
	}

done:
	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"count":   len(logs),
	}
}

// StartAllWorkersForUser 사용자의 모든 주문에 대한 워커를 시작합니다
func (h *Handler) StartAllWorkersForUser(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
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

	startedCount := 0
	failedCount := 0

	// 각 주문에 대해 워커 시작
	for _, order := range userData.SellOrderList {
		// 플랫폼 키 찾기
		var platformKey local_file.PlatformKey
		found := false
		for _, key := range userData.PlatformKeyList {
			if key.PlatformName == order.Platform && key.Name == order.PlatformNickName {
				platformKey = key
				found = true
				break
			}
		}

		if !found {
			failedCount++
			continue
		}

		// 워커 생성
		worker, err := h.workerFactory.CreateWorker(order, platformKey.PlatformAccessKey, platformKey.PlatformSecretKey)
		if err != nil {
			failedCount++
			continue
		}

		// 워커 매니저에 추가
		if err := h.workerManager.AddWorker(order.Name, userID, worker); err != nil {
			failedCount++
			continue
		}

		// 워커 시작
		if err := h.workerManager.StartWorker(order.Name); err != nil {
			failedCount++
			continue
		}

		startedCount++
	}

	log.Printf("사용자 워커 시작 완료: userID=%s, 시작=%d, 실패=%d", userID, startedCount, failedCount)
	return map[string]interface{}{
		"success":      true,
		"startedCount": startedCount,
		"failedCount":  failedCount,
		"message":      fmt.Sprintf("%d개 워커가 시작되었습니다. (실패: %d개)", startedCount, failedCount),
	}
}

// StopAllWorkersForUser 사용자의 모든 워커를 중지합니다
func (h *Handler) StopAllWorkersForUser(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	// 모든 워커 중지
	h.workerManager.StopAllWorkers()

	log.Printf("사용자 워커 중지 완료: userID=%s", userID)
	return map[string]interface{}{
		"success": true,
		"message": "모든 워커가 중지되었습니다.",
	}
}

// GetWorkerFactory 워커 팩토리를 반환합니다
func (h *Handler) GetWorkerFactory() *WorkerFactory {
	return h.workerFactory
}

// GetWorkerManager 워커 매니저를 반환합니다
func (h *Handler) GetWorkerManager() *WorkerManager {
	return h.workerManager
}

// CreateWorkerForOrder 주문에 대한 워커를 생성하고 시작합니다
func (h *Handler) CreateWorkerForOrder(order local_file.SellOrder, userID string, accessKey, secretKey string) error {
	// 워커 생성
	worker, err := h.workerFactory.CreateWorker(order, accessKey, secretKey)
	if err != nil {
		return fmt.Errorf("워커 생성 실패: %w", err)
	}

	// 워커 매니저에 추가
	if err := h.workerManager.AddWorker(order.Name, userID, worker); err != nil {
		return fmt.Errorf("워커 추가 실패: %w", err)
	}

	// 워커 시작
	if err := h.workerManager.StartWorker(order.Name); err != nil {
		return fmt.Errorf("워커 시작 실패: %w", err)
	}

	return nil
}

// GetWorkerInfoByUserID 특정 사용자의 워커 정보들을 반환합니다
func (h *Handler) GetWorkerInfoByUserID(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	workerInfo := h.workerManager.GetWorkerInfoByUserID(userID)
	return map[string]interface{}{
		"success":     true,
		"workerInfo":  workerInfo,
		"workerCount": len(workerInfo),
	}
}

// GetWorkerLogsByOrderName 특정 주문의 워커 로그를 반환합니다
func (h *Handler) GetWorkerLogsByOrderName(userID string, orderName string, limit int) map[string]interface{} {
	userID = strings.TrimSpace(userID)
	orderName = strings.TrimSpace(orderName)

	if userID == "" || orderName == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID와 주문 이름이 필요합니다.",
		}
	}

	if limit <= 0 {
		limit = 100 // 기본값
	}

	logs, err := h.workerManager.GetWorkerLogs(orderName, limit)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"count":   len(logs),
	}
}

// ClearWorkerLogsByOrderName 특정 주문의 워커 로그를 초기화합니다
func (h *Handler) ClearWorkerLogsByOrderName(userID string, orderName string) map[string]interface{} {
	userID = strings.TrimSpace(userID)
	orderName = strings.TrimSpace(orderName)

	if userID == "" || orderName == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID와 주문 이름이 필요합니다.",
		}
	}

	if err := h.workerManager.ClearWorkerLogs(orderName); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "로그가 초기화되었습니다.",
	}
}

// GetWorkerLogsStream 사용자의 워커 로그를 실시간으로 스트리밍합니다
func (h *Handler) GetWorkerLogsStream(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	// 사용자의 워커 정보 조회
	workerInfo := h.workerManager.GetWorkerInfoByUserID(userID)

	// 최근 로그들을 수집
	allLogs := []WorkerLog{}
	for orderName := range workerInfo {
		logs, err := h.workerManager.GetWorkerLogs(orderName, 50) // 최근 50개 로그
		if err == nil {
			allLogs = append(allLogs, logs...)
		}
	}

	return map[string]interface{}{
		"success": true,
		"logs":    allLogs,
		"count":   len(allLogs),
	}
}

// GetOrderLogs 특정 주문의 로그를 반환합니다
func (h *Handler) GetOrderLogs(userID string, orderName string) map[string]interface{} {
	userID = strings.TrimSpace(userID)
	orderName = strings.TrimSpace(orderName)

	log.Printf("GetOrderLogs 호출: userID=%s, orderName=%s", userID, orderName)

	if userID == "" || orderName == "" {
		log.Printf("GetOrderLogs 실패: 빈 파라미터")
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID와 주문 이름이 필요합니다.",
		}
	}

	// 로컬 파일에서 주문 로그 조회
	logs, err := h.localFileHandler.GetOrderLogs(userID, orderName)
	if err != nil {
		log.Printf("GetOrderLogs 실패: %v", err)
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	log.Printf("GetOrderLogs 성공: %d개 로그 반환", len(logs))
	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"count":   len(logs),
	}
}

// SubscribeToLogs 로그 구독을 시작합니다
func (h *Handler) SubscribeToLogs(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	// 클라이언트 구독
	_ = h.workerManager.SubscribeClient(userID)

	return map[string]interface{}{
		"success": true,
		"message": "로그 구독이 시작되었습니다.",
		"userID":  userID,
	}
}

// UnsubscribeFromLogs 로그 구독을 해제합니다
func (h *Handler) UnsubscribeFromLogs(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	// 클라이언트 구독 해제
	h.workerManager.UnsubscribeClient(userID)

	return map[string]interface{}{
		"success": true,
		"message": "로그 구독이 해제되었습니다.",
		"userID":  userID,
	}
}

// GetUnifiedLogs 통합된 로그를 반환합니다
func (h *Handler) GetUnifiedLogs(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 필요합니다.",
		}
	}

	// 통합된 로그 채널에서 최근 로그들 수집
	logs := []UnifiedLog{}
	logChan := h.workerManager.GetUnifiedLogChannel()

	// 최근 50개 로그 수집
	for i := 0; i < 50; i++ {
		select {
		case log := <-logChan:
			if log.UserID == userID {
				logs = append(logs, log)
			}
		default:
			// 더 이상 로그가 없으면 종료
			goto done
		}
	}

done:
	return map[string]interface{}{
		"success": true,
		"logs":    logs,
		"count":   len(logs),
	}
}
