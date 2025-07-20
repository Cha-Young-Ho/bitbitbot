# Windows용 실행 파일 생성 가이드

## ✅ 방법 1: macOS에서 크로스 컴파일 (권장)

macOS에서도 Windows용 실행 파일을 생성할 수 있습니다!

### 1단계: mingw-w64 설치
```bash
brew install mingw-w64
```

### 2단계: 빌드 스크립트 실행
```bash
./build_windows_mac.sh
```

또는 직접 명령어 실행:
```bash
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags="-s -w -H windowsgui" -o bitcoin_trader.exe main.go
```

### 결과
- **파일 크기**: 약 22MB
- **파일 타입**: Windows GUI 실행 파일
- **기능**: 완전한 GUI 지원
- **데이터 저장**: 사용자 홈 디렉토리에 암호화

## 방법 2: Windows 컴퓨터에서 직접 빌드

1. Windows 컴퓨터에 Go를 설치합니다 (https://golang.org/dl/)
2. 이 프로젝트 폴더를 Windows 컴퓨터로 복사합니다
3. Windows 명령 프롬프트 또는 PowerShell에서 다음 명령을 실행합니다:

```bash
go mod tidy
go build -o bitcoin_trader.exe main.go
```

## 방법 3: Fyne 패키징 도구 사용

더 전문적인 패키지를 만들려면:

```bash
# Windows에서 실행
go install fyne.io/tools/cmd/fyne@latest
fyne package --target windows --name "Bitcoin Trader"
```

## 방법 4: Docker를 사용한 크로스 컴파일

Docker가 설치되어 있다면:

```bash
docker run --rm -v "$PWD":/usr/src/app -w /usr/src/app golang:1.21-windowsservercore-ltsc2022 go build -o bitcoin_trader.exe main.go
```

## 방법 5: GitHub Actions를 사용한 자동 빌드

GitHub에 프로젝트를 업로드하고 GitHub Actions를 사용하여 자동으로 Windows용 실행 파일을 생성할 수 있습니다.

## 실행 방법

1. `bitcoin_trader.exe` 파일을 Windows 컴퓨터로 복사
2. 더블클릭하여 실행
3. 마스터 키 입력 (예: "123123")
4. 거래소 등록 및 매도 주문 관리

## 주요 기능

- 🔐 AES-256 암호화로 데이터 보안
- 🏪 다중 거래소 지원
- 💰 소수점 15자리까지 비트코인 수량 지원
- 📊 실시간 주문 관리
- 🎨 직관적인 GUI 인터페이스

## 문제 해결

### mingw-w64 설치 문제
```bash
# Homebrew 업데이트
brew update
brew install mingw-w64
```

### 빌드 실패 시
```bash
# 의존성 정리
go mod tidy
go clean -cache

# 다시 빌드
./build_windows_mac.sh
```

### Windows에서 실행 오류 시
- Windows Defender에서 차단되는 경우 예외 처리
- 관리자 권한으로 실행
- Visual C++ Redistributable 설치 필요할 수 있음 