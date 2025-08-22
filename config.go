package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Config 설정 정보
type Config struct {
	Running     string   `json:"running"`     // "on", "off", "all"
	WhiteList   []string `json:"whiteList"`   // 허용된 사용자 ID 목록
	MainVer     string   `json:"mainVer"`     // 선택적 업데이트 버전
	MinVer      string   `json:"minVer"`      // 필수 업데이트 버전
	UpdateUrl   string   `json:"updateUrl"`   // 업데이트 파일 다운로드 URL
	UpdatePath  string   `json:"updatePath"`  // S3 업데이트 파일 경로
	ForceUpdate bool     `json:"forceUpdate"` // 강제 업데이트 여부
}

var (
	configUrl   string // 빌드 시 주입되는 설정 파일 URL
	config      *Config
	Version     string // 빌드 시 주입되는 버전 정보
	Environment string // 빌드 시 주입되는 환경 정보 (prod, dev 등)
	// 주기적 검증 알림을 위한 채널
	periodicValidationChan chan string
	// 현재 로그인된 사용자 ID
	currentLoggedInUser string
)

// setCurrentUser 현재 로그인된 사용자를 설정합니다
func setCurrentUser(userID string) {
	currentLoggedInUser = userID
}

// getCurrentUser 현재 로그인된 사용자를 반환합니다
func getCurrentUser() string {
	return currentLoggedInUser
}

// clearCurrentUser 현재 로그인된 사용자를 초기화합니다
func clearCurrentUser() {
	currentLoggedInUser = ""
}

// initConfigSettings 빌드 시 주입된 설정을 확인합니다
func initConfigSettings() error {
	if configUrl == "" {
		return fmt.Errorf("설정 URL이 빌드 시 주입되지 않았습니다: -ldflags로 configUrl를 설정해주세요")
	}

	log.Printf("설정 초기화 완료: URL=%s", configUrl)
	return nil
}

// loadConfig 설정 파일을 읽어옵니다 (타임아웃 포함)
func loadConfig() error {
	// 30초 타임아웃으로 설정 파일 읽기
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configUrl, nil)
	if err != nil {
		return fmt.Errorf("http 요청 생성 실패: %v", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http 요청 실패: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http 상태 코드 오류: %d", resp.StatusCode)
	}

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("응답 본문 읽기 실패: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		return fmt.Errorf("json 파싱 실패: %v", err)
	}
	config = &cfg
	return nil
}

// checkProgramStatus 프로그램 상태를 체크합니다
func checkProgramStatus() error {
	if config == nil {
		return fmt.Errorf("설정이 로드되지 않았습니다")
	}

	// Running 상태 체크 먼저 수행
	if config.Running == "off" {
		log.Printf("프로그램이 비활성화됨: Running=%s", config.Running)
		return fmt.Errorf("프로그램이 비활성화되었습니다")
	}

	// Running이 "on"인 경우에만 whiteList 검증 수행
	if config.Running == "on" {
		if err := checkWhiteList(); err != nil {
			return err
		}
	}

	// Running이 "all" 또는 "on"인 경우 버전 체크 수행
	if config.Running == "all" || config.Running == "on" {
		if err := checkVersion(); err != nil {
			return err
		}
	}

	return nil
}

// checkWhiteList 화이트리스트를 체크합니다
func checkWhiteList() error {
	// 현재 로그인된 사용자가 없으면 검증 통과
	currentUser := getCurrentUser()
	if currentUser == "" {
		return nil
	}

	if len(config.WhiteList) == 0 {
		return nil
	}

	// 화이트리스트에 현재 사용자 ID가 있는지 확인
	for _, allowedUser := range config.WhiteList {
		if allowedUser == currentUser {
			return nil
		}
	}

	return fmt.Errorf("invalid account")
}

// checkVersion 버전을 체크하고 필요시 업데이트를 요청합니다
func checkVersion() error {
	// 현재 버전 (빌드 시 설정됨)
	currentVersion := getVersion()

	// MinVer 체크 (필수 업데이트)
	if config.MinVer != "" && currentVersion < config.MinVer {
		return fmt.Errorf("required_update:min_ver_failed")
	}

	// MainVer 체크 (선택적 업데이트) - 무조건 mainVer로 업데이트
	if config.MainVer != "" && currentVersion < config.MainVer {
		return fmt.Errorf("optional_update:main_ver_failed")
	}

	return nil
}

// performAutoUpdate 자동 업데이트를 수행합니다
func performAutoUpdate() error {
	updateUrl := getUpdateUrl()
	if updateUrl == "" {
		return fmt.Errorf("업데이트 URL을 생성할 수 없습니다")
	}

	log.Printf("자동 업데이트 시작: %s", updateUrl)
	log.Printf("다운로드할 파일 URL: %s", updateUrl)

	// 현재 실행 파일 경로 가져오기
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("실행 파일 경로 확인 실패: %v", err)
	}

	// 임시 디렉토리 생성
	tempDir := os.TempDir()
	tempZipFile := filepath.Join(tempDir, "bitbit-update-temp.zip")
	tempExtractDir := filepath.Join(tempDir, "bitbit-update-extract")

	// 기존 임시 디렉토리 정리
	os.RemoveAll(tempExtractDir)
	os.MkdirAll(tempExtractDir, 0755)

	// ZIP 파일 다운로드
	if err := downloadFile(updateUrl, tempZipFile); err != nil {
		return fmt.Errorf("업데이트 ZIP 파일 다운로드 실패: %v", err)
	}

	// ZIP 파일 압축 해제
	if err := unzipFile(tempZipFile, tempExtractDir); err != nil {
		return fmt.Errorf("ZIP 파일 압축 해제 실패: %v", err)
	}

	// 압축 해제된 실행 파일 찾기
	var extractedFile string
	files, err := os.ReadDir(tempExtractDir)
	if err != nil {
		return fmt.Errorf("압축 해제된 파일 읽기 실패: %v", err)
	}

	log.Printf("압축 해제된 파일들: %d개", len(files))
	for _, file := range files {
		log.Printf("  - %s (디렉토리: %v)", file.Name(), file.IsDir())
		if !file.IsDir() {
			extractedFile = filepath.Join(tempExtractDir, file.Name())
			break
		}
	}

	if extractedFile == "" {
		return fmt.Errorf("압축 해제된 실행 파일을 찾을 수 없습니다")
	}

	log.Printf("추출된 실행 파일: %s", extractedFile)

	// 실행 권한 부여 (Unix/Linux/Mac)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(extractedFile, 0755); err != nil {
			return fmt.Errorf("실행 권한 부여 실패: %v", err)
		}
	}

	// 백업 파일 생성
	backupFile := execPath + ".backup"
	log.Printf("백업 파일 생성: %s", backupFile)
	if err := copyFile(execPath, backupFile); err != nil {
		return fmt.Errorf("백업 파일 생성 실패: %v", err)
	}

	// 새 버전으로 교체
	log.Printf("파일 교체: %s -> %s", extractedFile, execPath)
	if err := copyFile(extractedFile, execPath); err != nil {
		log.Printf("파일 교체 실패, 백업에서 복원: %v", err)
		// 실패 시 백업에서 복원
		copyFile(backupFile, execPath)
		return fmt.Errorf("파일 교체 실패: %v", err)
	}

	log.Printf("파일 교체 완료")

	// 임시 파일 정리
	os.Remove(tempZipFile)
	os.RemoveAll(tempExtractDir)

	log.Printf("자동 업데이트 완료: %s -> %s", getVersion(), config.MainVer)
	return nil
}

// downloadFile 파일을 다운로드합니다
func downloadFile(url, filepath string) error {
	log.Printf("파일 다운로드 시작: URL=%s, 저장경로=%s", url, filepath)

	// HTTP 클라이언트 생성 (타임아웃 5분)
	client := &http.Client{Timeout: 5 * time.Minute}

	// 요청 생성
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 응답 받기
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("다운로드 실패: %v", err)
		return fmt.Errorf("다운로드 실패: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("다운로드 응답: 상태코드=%d, Content-Length=%s", resp.StatusCode, resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP 오류: 상태코드=%d", resp.StatusCode)
		return fmt.Errorf("HTTP 오류: %d", resp.StatusCode)
	}

	// 파일 생성
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("파일 생성 실패: %v", err)
	}
	defer out.Close()

	// 파일에 쓰기
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("파일 쓰기 실패: %v", err)
	}

	return nil
}

// copyFile 파일을 복사합니다
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// unzipFile ZIP 파일을 압축 해제합니다
func unzipFile(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("ZIP 파일 열기 실패: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		// 파일 경로 생성
		filePath := filepath.Join(destDir, file.Name)

		// 디렉토리인 경우 생성
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}

		// 파일인 경우 디렉토리 생성 후 파일 생성
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패: %v", err)
		}

		// 파일 생성
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("파일 생성 실패: %v", err)
		}

		// ZIP 파일에서 읽기
		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("ZIP 파일 읽기 실패: %v", err)
		}

		// 파일에 쓰기
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("파일 쓰기 실패: %v", err)
		}
	}

	return nil
}

// checkUserAccess 사용자 접근 권한을 체크합니다
func checkUserAccess(userID string) error {
	if config == nil {
		return fmt.Errorf("설정이 로드되지 않았습니다")
	}

	log.Printf("사용자 접근 검증 시작 - UserID: %s, Running: %s", userID, config.Running)

	// Running이 "all"이면 모든 사용자 허용 (whiteList 검증 하지 않음)
	if config.Running == "all" {
		log.Printf("Running이 'all'이므로 모든 사용자 허용 - UserID: %s", userID)
		return nil
	}

	// Running이 "on"일 때만 whiteList 체크
	if config.Running == "on" {
		log.Printf("Running이 'on'이므로 whiteList 검증 - UserID: %s, WhiteList: %v", userID, config.WhiteList)
		// WhiteList 체크
		for _, allowedID := range config.WhiteList {
			if allowedID == userID {
				log.Printf("whiteList 검증 성공 - UserID: %s", userID)
				return nil
			}
		}
		log.Printf("whiteList 검증 실패 - UserID: %s가 허용 목록에 없음", userID)
		return fmt.Errorf("invalid account")
	}

	// Running이 "off"이면 접근 거부
	log.Printf("Running이 'off'이므로 접근 거부 - UserID: %s", userID)
	return fmt.Errorf("invalid account")
}

// getVersion 현재 버전을 반환합니다 (빌드 시 주입됨)
func getVersion() string {
	// 빌드 시 -ldflags로 주입되는 버전 정보
	if Version != "" {
		return Version
	}
	return "1.0.0" // 기본값
}

// getUpdateUrl 환경과 버전에 따른 업데이트 URL을 생성합니다
func getUpdateUrl() string {
	if config.UpdateUrl != "" {
		log.Printf("업데이트 URL (설정에서 가져옴): %s", config.UpdateUrl)
		return config.UpdateUrl
	}

	// 환경별 기본 경로 생성
	if Environment == "" {
		Environment = "prod" // 기본값
	}

	// 운영체제별 파일명 생성 (무조건 mainVer로 업데이트)
	var fileName string
	updateVersion := config.MainVer

	log.Printf("업데이트 버전 결정: MainVer=%s, MinVer=%s, 선택된 버전=%s (무조건 mainVer 사용)", config.MainVer, config.MinVer, updateVersion)

	switch runtime.GOOS {
	case "windows":
		fileName = fmt.Sprintf("win_build.%s.zip", updateVersion)
	case "darwin":
		fileName = fmt.Sprintf("mac_build.%s.zip", updateVersion)
	case "linux":
		fileName = fmt.Sprintf("linux_build.%s.zip", updateVersion)
	default:
		fileName = fmt.Sprintf("build.%s.zip", updateVersion)
	}

	log.Printf("생성된 파일명: %s (OS: %s)", fileName, runtime.GOOS)

	// S3 URL 생성 (configUrl에서 버킷 정보 추출)
	if configUrl != "" {
		// https://bucket.s3.region.amazonaws.com/path 형태에서 버킷 추출
		if strings.Contains(configUrl, "s3.") {
			parts := strings.Split(configUrl, "/")
			if len(parts) >= 3 {
				bucket := strings.Split(parts[2], ".")[0]
				updateUrl := fmt.Sprintf("https://%s.s3.ap-northeast-2.amazonaws.com/%s/%s",
					bucket, Environment, fileName)
				log.Printf("생성된 업데이트 URL: %s", updateUrl)
				return updateUrl
			}
		}
	}

	log.Printf("업데이트 URL 생성 실패: configUrl=%s", configUrl)
	return ""
}

// showInvalidVersionAndExit Invalid Version 메시지를 표시합니다 (프론트엔드에서 처리)
func showInvalidVersionAndExit() {
	log.Printf("Invalid Version - 프론트엔드에서 처리해야 함")
	// Wails 앱에서는 프론트엔드에서 처리하므로 프로그램을 종료하지 않음
}

// showUpdateCompleteAndRestart 업데이트 완료 메시지를 표시합니다
func showUpdateCompleteAndRestart() {
	log.Printf("업데이트 완료 - 사용자에게 재시작 안내")
	// Wails 앱에서는 프론트엔드에서 재시작 안내를 처리하므로 여기서는 로그만 남김
}

// showInvalidAccessAndExit 잘못된 접근 메시지를 표시하고 워커를 삭제합니다
func showInvalidAccessAndExit() {
	log.Printf("잘못된 접근 - 워커 삭제 및 종료 안내")
	// Wails 앱에서는 프론트엔드에서 처리하므로 여기서는 로그만 남김
}

// restartProgram 프로그램을 재시작합니다
func restartProgram(execPath string) error {
	log.Printf("프로그램 재시작 시도: %s", execPath)

	// 현재 프로세스의 인자들을 가져오기
	args := os.Args[1:]
	log.Printf("재시작 인자: %v", args)

	// 새 프로세스 시작
	cmd := exec.Command(execPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// 백그라운드에서 실행
	if err := cmd.Start(); err != nil {
		log.Printf("새 프로세스 시작 실패: %v", err)
		return fmt.Errorf("새 프로세스 시작 실패: %v", err)
	}

	log.Printf("새 프로세스 시작 성공: PID=%d", cmd.Process.Pid)

	// Wails 앱에서는 os.Exit 대신 더 안전한 방법 사용
	// 현재 프로세스를 종료하기 전에 잠시 대기
	time.Sleep(1 * time.Second)

	// 현재 프로세스 종료
	log.Printf("현재 프로세스 종료")
	os.Exit(0)
	return nil
}

// startPeriodicConfigCheck 주기적으로 설정을 체크하는 고루틴을 시작합니다
func startPeriodicConfigCheck() {
	// 채널 초기화
	periodicValidationChan = make(chan string, 10)

	go func() {
		// 시작하자마자 1번 실행
		performPeriodicValidationCheck()

		// 설정된 시간마다 실행 (5초)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			performPeriodicValidationCheck()
		}
	}()
}

// performPeriodicValidationCheck 주기적 검증을 수행합니다
func performPeriodicValidationCheck() {
	if err := performConfigValidation(); err != nil {
		// 에러 타입에 따라 다른 처리
		if strings.Contains(err.Error(), "required_update") || strings.Contains(err.Error(), "optional_update") {
			// 버전 관련 문제 - 업데이트 다이얼로그 표시
			// 프론트엔드로 알림 전송
			select {
			case periodicValidationChan <- err.Error():
			default:
			}
		} else if strings.Contains(err.Error(), "invalid account") || strings.Contains(err.Error(), "프로그램이 비활성화되었습니다") {
			// 접근 권한 문제 - 잘못된 접근 메시지 표시
			// 프론트엔드로 알림 전송
			select {
			case periodicValidationChan <- "invalid_access":
			default:
			}
		} else {
			// 기타 문제 - 기본 처리
			showInvalidVersionAndExit()
			return
		}
	}
}

// performConfigValidation 설정을 검증합니다 (초기 로드 + 주기적 체크용)
func performConfigValidation() error {
	// 설정 다시 로드
	if err := loadConfig(); err != nil {
		return fmt.Errorf("config_load_failed: %v", err)
	}

	// 프로그램 상태 체크
	if err := checkProgramStatus(); err != nil {
		return fmt.Errorf("program_status_failed: %v", err)
	}

	return nil
}
