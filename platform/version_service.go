package platform

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// VersionService 버전 관리 로직을 담당하는 서비스
type VersionService struct {
	getCurrentVersion func() string
}

// NewVersionService 새로운 버전 서비스 생성
func NewVersionService(getCurrentVersion func() string) *VersionService {
	return &VersionService{
		getCurrentVersion: getCurrentVersion,
	}
}

// CompareVersions 버전을 비교합니다
func (vs *VersionService) CompareVersions(config *Config) (*VersionComparison, error) {
	if config == nil {
		return nil, fmt.Errorf("설정이 nil입니다")
	}

	currentVersion := vs.getCurrentVersion()

	// 현재 버전이 mainVer보다 낮은지 확인
	isMainUpdateNeeded := vs.compareVersion(currentVersion, config.MainVer) < 0
	
	// 현재 버전이 minVer보다 낮은지 확인
	isMinUpdateNeeded := vs.compareVersion(currentVersion, config.MinVer) < 0

	comparison := &VersionComparison{
		CurrentVersion:    currentVersion,
		MainVersion:       config.MainVer,
		MinVersion:        config.MinVer,
		IsMainUpdateNeeded: isMainUpdateNeeded,
		IsMinUpdateNeeded:  isMinUpdateNeeded,
		IsUpdateNeeded:    isMainUpdateNeeded || isMinUpdateNeeded,
		IsForceUpdate:     isMinUpdateNeeded,
		UpdateType:        vs.determineUpdateType(isMainUpdateNeeded, isMinUpdateNeeded),
	}

	log.Printf("버전 비교 결과: %+v", comparison)
	return comparison, nil
}

// compareVersion 버전 문자열을 비교합니다 (semantic versioning 지원)
func (vs *VersionService) compareVersion(v1, v2 string) int {
	// 빈 문자열 처리
	if v1 == "" && v2 == "" {
		return 0
	}
	if v1 == "" {
		return -1
	}
	if v2 == "" {
		return 1
	}

	// 정확한 일치
	if v1 == v2 {
		return 0
	}

	// semantic versioning 파싱
	parts1 := vs.parseVersion(v1)
	parts2 := vs.parseVersion(v2)

	// 각 부분을 비교
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		part1 := 0
		part2 := 0

		if i < len(parts1) {
			part1 = parts1[i]
		}
		if i < len(parts2) {
			part2 = parts2[i]
		}

		if part1 < part2 {
			return -1
		}
		if part1 > part2 {
			return 1
		}
	}

	return 0
}

// parseVersion 버전 문자열을 파싱합니다
func (vs *VersionService) parseVersion(version string) []int {
	// 점으로 분리
	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		// 숫자가 아닌 문자 제거
		cleanPart := strings.TrimFunc(part, func(r rune) bool {
			return r < '0' || r > '9'
		})

		if cleanPart == "" {
			result = append(result, 0)
			continue
		}

		if num, err := strconv.Atoi(cleanPart); err == nil {
			result = append(result, num)
		} else {
			result = append(result, 0)
		}
	}

	return result
}

// determineUpdateType 업데이트 타입을 결정합니다
func (vs *VersionService) determineUpdateType(isMainUpdateNeeded, isMinUpdateNeeded bool) string {
	if isMinUpdateNeeded {
		return "force"
	}
	if isMainUpdateNeeded {
		return "recommended"
	}
	return "none"
}

// GetCurrentVersion 현재 버전을 반환합니다
func (vs *VersionService) GetCurrentVersion() string {
	return vs.getCurrentVersion()
}

// VersionComparison 버전 비교 결과
type VersionComparison struct {
	CurrentVersion     string `json:"currentVersion"`
	MainVersion        string `json:"mainVersion"`
	MinVersion         string `json:"minVersion"`
	IsMainUpdateNeeded bool   `json:"isMainUpdateNeeded"`
	IsMinUpdateNeeded  bool   `json:"isMinUpdateNeeded"`
	IsUpdateNeeded     bool   `json:"isUpdateNeeded"`
	IsForceUpdate      bool   `json:"isForceUpdate"`
	UpdateType         string `json:"updateType"`
}
