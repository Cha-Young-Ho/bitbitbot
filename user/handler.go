package user

import (
	"bitbit-app/local_file"
	"log"
	"strings"
)

type Handler struct {
	localFileHandler *local_file.Handler
}

func NewHandler(localFileHandler *local_file.Handler) *Handler {
	return &Handler{
		localFileHandler: localFileHandler,
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
