package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gui-app/models"
)

const (
	DataFileName = "bitcoin_trader_data.json"
)

// LoadEncryptedData 암호화된 데이터를 로드합니다
func LoadEncryptedData(userKey string) (*models.AppData, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("홈 디렉토리 조회 실패: %v", err)
	}

	filePath := filepath.Join(homeDir, DataFileName)

	// 파일이 존재하지 않으면 새로운 데이터 반환
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &models.AppData{
			Exchanges:  []models.Exchange{},
			SellOrders: []models.SellOrder{},
		}, nil
	}

	// 파일 읽기
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %v", err)
	}

	// 암호화되지 않은 경우 (이전 버전 호환성)
	if len(encryptedData) == 0 {
		return &models.AppData{
			Exchanges:  []models.Exchange{},
			SellOrders: []models.SellOrder{},
		}, nil
	}

	// 복호화
	crypto := NewCryptoService()
	decryptedData, err := crypto.Decrypt(string(encryptedData), userKey)
	if err != nil {
		return nil, fmt.Errorf("복호화 실패: %v", err)
	}

	// JSON 디코딩
	var appData models.AppData
	if err := json.Unmarshal([]byte(decryptedData), &appData); err != nil {
		return nil, fmt.Errorf("JSON 디코딩 실패: %v", err)
	}

	return &appData, nil
}

// SaveEncryptedData 데이터를 암호화하여 저장합니다
func SaveEncryptedData(data *models.AppData, userKey string) error {
	fmt.Println("=== SaveEncryptedData 시작 ===")

	// JSON 인코딩
	fmt.Println("JSON 인코딩 시작...")
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("JSON 인코딩 실패: %v\n", err)
		return fmt.Errorf("JSON 인코딩 실패: %v", err)
	}
	fmt.Printf("JSON 인코딩 완료. 크기: %d bytes\n", len(jsonData))

	// 암호화
	fmt.Println("암호화 시작...")
	crypto := NewCryptoService()
	encryptedData, err := crypto.Encrypt(string(jsonData), userKey)
	if err != nil {
		fmt.Printf("암호화 실패: %v\n", err)
		return fmt.Errorf("암호화 실패: %v", err)
	}
	fmt.Printf("암호화 완료. 크기: %d bytes\n", len(encryptedData))

	// 파일 경로 생성
	fmt.Println("파일 경로 생성...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("홈 디렉토리 조회 실패: %v\n", err)
		return fmt.Errorf("홈 디렉토리 조회 실패: %v", err)
	}

	filePath := filepath.Join(homeDir, DataFileName)
	fmt.Printf("저장할 파일 경로: %s\n", filePath)

	// 파일 저장
	fmt.Println("파일 저장 시작...")
	if err := os.WriteFile(filePath, []byte(encryptedData), 0600); err != nil {
		fmt.Printf("파일 저장 실패: %v\n", err)
		return fmt.Errorf("파일 저장 실패: %v", err)
	}
	fmt.Println("파일 저장 성공")
	fmt.Println("=== SaveEncryptedData 완료 ===")

	return nil
}
