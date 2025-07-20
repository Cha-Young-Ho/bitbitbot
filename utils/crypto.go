package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// CryptoService 암호화 서비스
type CryptoService struct{}

// NewCryptoService 새로운 암호화 서비스 생성
func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

// Encrypt 텍스트를 암호화합니다
func (c *CryptoService) Encrypt(text, key string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("암호화할 텍스트가 비어있습니다")
	}
	if key == "" {
		return "", fmt.Errorf("암호화 키가 비어있습니다")
	}

	keyHash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", fmt.Errorf("암호화 블록 생성 실패: %w", err)
	}

	plaintext := []byte(text)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("IV 생성 실패: %w", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return hex.EncodeToString(ciphertext), nil
}

// Decrypt 암호화된 텍스트를 복호화합니다
func (c *CryptoService) Decrypt(cipherHex, key string) (string, error) {
	if cipherHex == "" {
		return "", fmt.Errorf("복호화할 텍스트가 비어있습니다")
	}
	if key == "" {
		return "", fmt.Errorf("복호화 키가 비어있습니다")
	}

	keyHash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", fmt.Errorf("복호화 블록 생성 실패: %w", err)
	}

	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return "", fmt.Errorf("헥스 디코딩 실패: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("암호화된 텍스트가 너무 짧습니다")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

// ValidateKey 키의 유효성을 검증합니다
func (c *CryptoService) ValidateKey(key string) error {
	if len(key) < 6 {
		return fmt.Errorf("키는 최소 6자리 이상이어야 합니다")
	}
	if len(key) > 256 {
		return fmt.Errorf("키는 최대 256자리까지 가능합니다")
	}
	return nil
}
