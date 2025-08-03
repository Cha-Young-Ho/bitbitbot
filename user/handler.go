package user

import (
	"bitbit-app/local_file"
	"bitbit-app/platform"
	"fmt"
	"log"
	"strings"
)

type Handler struct {
	localFileHandler *local_file.Handler
	platformHandler  *platform.Handler
}

func NewHandler(localFileHandler *local_file.Handler, platformHandler *platform.Handler) *Handler {
	return &Handler{
		localFileHandler: localFileHandler,
		platformHandler:  platformHandler,
	}
}

// Login 사용자 로그인을 처리합니다
func (h *Handler) Login(userID, password string) map[string]interface{} {
	// 입력값 정리 및 검증
	userID = strings.TrimSpace(userID)
	password = strings.TrimSpace(password)

	// 입력값 검증
	if userID == "" || password == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID와 비밀번호를 입력해주세요.",
		}
	}

	// 로컬 파일에서 사용자 조회
	userData, err := h.localFileHandler.GetUserByID(userID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": "계정이 없거나 아이디 또는 비밀번호가 올바르지 않습니다.",
		}
	}

	// 비밀번호 검증
	if userData.Password != password {
		return map[string]interface{}{
			"success": false,
			"message": "계정이 없거나 아이디 또는 비밀번호가 올바르지 않습니다.",
		}
	}

	log.Printf("로그인 성공: userID=%s", userID)

	// 로그인 성공 시 사용자의 모든 매도 예약 주문에 대한 워커 생성
	go func() {
		if err := h.createWorkersForUser(userID, userData); err != nil {
			log.Printf("워커 생성 중 오류 발생: userID=%s, error=%v", userID, err)
		}
	}()

	return map[string]interface{}{
		"success": true,
		"message": "로그인 성공",
		"user": map[string]interface{}{
			"userId":       userID,
			"platformKeys": userData.PlatformKeyList,
		},
	}
}

// Register 사용자 회원가입을 처리합니다
func (h *Handler) Register(userID, password string) map[string]interface{} {
	// 입력값 정리 및 검증
	userID = strings.TrimSpace(userID)
	password = strings.TrimSpace(password)

	// 입력값 검증
	if userID == "" || password == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID와 비밀번호를 입력해주세요.",
		}
	}

	// 새 사용자 데이터 생성
	newUser := local_file.UserData{
		ID:              userID,
		Password:        password,
		PlatformKeyList: []local_file.PlatformKey{},
	}

	// 로컬 파일에 사용자 추가
	err := h.localFileHandler.AddUser(newUser)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "회원가입 성공",
		"userId":  userID,
	}
}

// GetUserInfo 사용자 정보를 조회합니다
func (h *Handler) GetUserInfo(userID string) map[string]interface{} {
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
		"success":      true,
		"userId":       userID,
		"platformKeys": userData.PlatformKeyList,
	}
}

// GetAccountInfo 사용자 계정 정보를 조회합니다 (별칭)
func (h *Handler) GetAccountInfo(userID string) map[string]interface{} {
	return h.GetUserInfo(userID)
}

// createWorkersForUser 사용자의 모든 매도 예약 주문에 대한 워커를 생성합니다
func (h *Handler) createWorkersForUser(userID string, userData *local_file.UserData) error {
	log.Printf("사용자 워커 생성 시작: userID=%s, 주문 수=%d", userID, len(userData.SellOrderList))

	createdCount := 0
	failedCount := 0

	// 각 매도 예약 주문에 대해 워커 생성
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
			log.Printf("플랫폼 키를 찾을 수 없음: order=%s, platform=%s, nickname=%s",
				order.Name, order.Platform, order.PlatformNickName)
			failedCount++
			continue
		}

		// 워커 생성 및 시작
		if err := h.platformHandler.CreateWorkerForOrder(order, userID, platformKey.PlatformAccessKey, platformKey.PlatformSecretKey); err != nil {
			log.Printf("워커 생성 실패: order=%s, error=%v", order.Name, err)
			failedCount++
			continue
		}

		log.Printf("워커 생성 및 시작 완료: order=%s, platform=%s", order.Name, order.Platform)
		createdCount++
	}

	log.Printf("사용자 워커 생성 완료: userID=%s, 생성=%d, 실패=%d", userID, createdCount, failedCount)

	if failedCount > 0 {
		return fmt.Errorf("일부 워커 생성 실패: %d개", failedCount)
	}

	return nil
}

// Logout 사용자 로그아웃을 처리합니다
func (h *Handler) Logout(userID string) map[string]interface{} {
	userID = strings.TrimSpace(userID)

	if userID == "" {
		return map[string]interface{}{
			"success": false,
			"message": "사용자 ID가 비어있습니다.",
		}
	}

	// 로그아웃 시 사용자의 모든 워커 중지
	go h.stopWorkersForUser(userID)

	log.Printf("로그아웃: userID=%s", userID)
	return map[string]interface{}{
		"success": true,
		"message": "로그아웃 성공",
	}
}

// stopWorkersForUser 사용자의 모든 워커를 중지합니다
func (h *Handler) stopWorkersForUser(userID string) {
	log.Printf("사용자 워커 중지 시작: userID=%s", userID)

	// 모든 워커 중지
	h.platformHandler.GetWorkerManager().StopAllWorkers()

	log.Printf("사용자 워커 중지 완료: userID=%s", userID)
}
