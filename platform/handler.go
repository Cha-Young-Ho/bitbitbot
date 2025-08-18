package platform

import (
	"bitbit-app/local_file"
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
func (h *Handler) AddPlatform(userID string, platformName string, name string, accessKey, secretKey, passwordPhrase string) map[string]interface{} {
	// 입력값 정리
	userID = strings.TrimSpace(userID)
	platformName = strings.TrimSpace(platformName)
	name = strings.TrimSpace(name)
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	passwordPhrase = strings.TrimSpace(passwordPhrase)

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
				"message": "이미 존재하는 플랫폼 별칭입니다",
			}
		}
	}

	// 새 플랫폼 키 추가
	newPlatformKey := local_file.PlatformKey{
		PlatformName:      platformName,
		Name:              name,
		PlatformAccessKey: accessKey,
		PlatformSecretKey: secretKey,
		PasswordPhrase:    passwordPhrase,
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
			"message": "플랫폼을 찾을 수 없습니다",
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

	// 해당 API Key를 사용하는 모든 예약 매도 워커 중지 및 제거
	for _, order := range userData.SellOrderList {
		if order.Platform == platformName && order.PlatformNickName == name {
			_ = h.workerManager.StopWorker(order.Name)
			_ = h.workerManager.RemoveWorker(order.Name)
		}
	}

	log.Printf("플랫폼 제거 완료: userID=%s, platform=%s, name=%s", userID, platformName, name)
	return map[string]interface{}{
		"success": true,
		"message": "플랫폼이 제거되었습니다.",
	}
}

// UpdatePlatform 사용자의 플랫폼 정보를 업데이트합니다
func (h *Handler) UpdatePlatform(userID string, oldPlatformName string, oldName string, newPlatformName string, newName string, accessKey, secretKey, passwordPhrase string) map[string]interface{} {
	// 입력값 정리
	userID = strings.TrimSpace(userID)
	oldPlatformName = strings.TrimSpace(oldPlatformName)
	oldName = strings.TrimSpace(oldName)
	newPlatformName = strings.TrimSpace(newPlatformName)
	newName = strings.TrimSpace(newName)
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	passwordPhrase = strings.TrimSpace(passwordPhrase)

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
				PasswordPhrase:    passwordPhrase,
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
			PasswordPhrase:    passwordPhrase,
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

	// 방금 추가한 주문에 대해 워커 즉시 생성/시작 (API Key는 저장된 것을 사용)
	var pk local_file.PlatformKey
	found := false
	for _, key := range userData.PlatformKeyList {
		if key.PlatformName == platformName && key.Name == platformNickName {
			pk = key
			found = true
			break
		}
	}
	if found {
		if err := h.CreateWorkerForOrder(newSellOrder, userID, pk.PlatformAccessKey, pk.PlatformSecretKey, pk.PasswordPhrase); err != nil {
			h.workerManager.SendSystemLog("Handler", "AddSellOrder", "워커 생성 실패", "error", userID, orderName, err.Error())
		}
	} else {
		h.workerManager.SendSystemLog("Handler", "AddSellOrder", "플랫폼 키를 찾을 수 없습니다", "error", userID, orderName, "")
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

// UpdateSellOrder 예약 매도 주문 수정
func (h *Handler) UpdateSellOrder(userID string, oldName string, orderName string, symbol string, price float64, quantity float64, term float64, platformName string, platformNickName string) map[string]interface{} {
	userID = strings.TrimSpace(userID)
	oldName = strings.TrimSpace(oldName)
	orderName = strings.TrimSpace(orderName)
	symbol = strings.TrimSpace(symbol)
	platformName = strings.TrimSpace(platformName)
	platformNickName = strings.TrimSpace(platformNickName)
	if userID == "" || oldName == "" || orderName == "" || symbol == "" || price <= 0 || quantity <= 0 || term <= 0 {
		return map[string]interface{}{"success": false, "message": "필수 값 오류"}
	}
	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	updated := local_file.SellOrder{Name: orderName, Symbol: symbol, Price: price, Quantity: quantity, Term: term, Platform: platformName, PlatformNickName: platformNickName}
	if err := h.localFileHandler.UpdateSellOrder(userID, oldName, updated); err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	// 워커 재생성 필요: 기존 워커가 있다면 중지 후 새로 시작
	if err := h.workerManager.StopWorker(oldName); err == nil {
		_ = h.workerManager.RemoveWorker(oldName)
	}
	// 이름이 변경되었으면 새 이름으로 워커 생성 (필요 시)
	// 플랫폼 키 조회
	var pk local_file.PlatformKey
	found := false
	for _, key := range userData.PlatformKeyList {
		if key.PlatformName == platformName && key.Name == platformNickName {
			pk = key
			found = true
			break
		}
	}
	if found {
		// 새 워커 생성/시작은 사용자의 의도에 따라 다를 수 있어 선택적으로 수행
		order := updated
		if err := h.CreateWorkerForOrder(order, userID, pk.PlatformAccessKey, pk.PlatformSecretKey, pk.PasswordPhrase); err != nil {
			h.workerManager.SendSystemLog("Handler", "UpdateSellOrder", "워커 재시작 실패", "error", userID, orderName, err.Error())
		}
	}
	return map[string]interface{}{"success": true, "message": "예약 매도 주문이 수정되었습니다."}
}

// RemoveSellOrder 예약 매도 주문 삭제
func (h *Handler) RemoveSellOrder(userID string, orderName string) map[string]interface{} {
	userID = strings.TrimSpace(userID)
	orderName = strings.TrimSpace(orderName)
	if userID == "" || orderName == "" {
		return map[string]interface{}{"success": false, "message": "필수 값 오류"}
	}
	if err := h.localFileHandler.RemoveSellOrder(userID, orderName); err != nil {
		return map[string]interface{}{"success": false, "message": err.Error()}
	}
	// 해당 워커가 돌고 있으면 중지/제거
	_ = h.workerManager.StopWorker(orderName)
	_ = h.workerManager.RemoveWorker(orderName)
	return map[string]interface{}{"success": true, "message": "예약 매도 주문이 삭제되었습니다."}
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
	worker, err := h.workerFactory.CreateWorker(targetOrder, platformKey.PlatformAccessKey, platformKey.PlatformSecretKey, platformKey.PasswordPhrase)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "워커 생성 실패",
		}
	}

	// 워커 매니저에 추가
	if err := h.workerManager.AddWorker(orderName, userID, worker); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "워커 추가 실패",
		}
	}

	// 워커 시작
	if err := h.workerManager.StartWorker(orderName); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "워커 시작 실패",
		}
	}

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
			"message": "워커 중지 실패",
		}
	}

	// 워커 제거
	if err := h.workerManager.RemoveWorker(orderName); err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "워커 제거 실패",
		}
	}

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
		worker, err := h.workerFactory.CreateWorker(order, platformKey.PlatformAccessKey, platformKey.PlatformSecretKey, platformKey.PasswordPhrase)
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

	return map[string]interface{}{
		"success":      true,
		"startedCount": startedCount,
		"failedCount":  failedCount,
		"message":      "모든 워커가 시작되었습니다.",
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
func (h *Handler) CreateWorkerForOrder(order local_file.SellOrder, userID string, accessKey, secretKey, passwordPhrase string) error {
	// 워커 생성
	worker, err := h.workerFactory.CreateWorker(order, accessKey, secretKey, passwordPhrase)
	if err != nil {
		return err
	}

	// 워커 매니저에 추가
	if err := h.workerManager.AddWorker(order.Name, userID, worker); err != nil {
		return err
	}

	// 워커 시작
	if err := h.workerManager.StartWorker(order.Name); err != nil {
		return err
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
// (삭제) GetWorkerLogsStream: 웹소켓 통합으로 불필요
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
// (삭제) GetOrderLogs: 웹소켓 통합으로 불필요
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

// RemoveAllWorkers 모든 워커를 제거합니다
func (h *Handler) RemoveAllWorkers() {
	h.workerManager.RemoveAllWorkers()
}
