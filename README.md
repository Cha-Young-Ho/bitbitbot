# 🚀 Bitcoin Trader Professional v3.0

**세련된 모던 UI와 전문가급 기능을 갖춘 비트코인 거래소 관리 플랫폼**

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![Fyne](https://img.shields.io/badge/Fyne-v2.6+-00ADD8?logo=go&logoColor=white)
![Security](https://img.shields.io/badge/Security-AES--256-green)

## ✨ 주요 특징

### 🎨 **kmong 스타일 모던 UI**
- 깔끔한 화이트 베이스 디자인
- 직관적인 대시보드 레이아웃
- 반응형 카드 기반 인터페이스
- 실시간 데이터 업데이트

### 📊 **프로페셔널 대시보드**
- 실시간 통계 카드
- 좌우 분할 레이아웃 (거래소 관리 | 주문 관리)
- 한눈에 보는 핵심 지표
- 드래그 앤 드롭 지원

### 🏢 **다중 거래소 통합**
- 업비트, 빗썸, 바이낸스 등 지원
- API Key 안전한 암호화 저장
- 거래소별 주문 관리
- 실시간 상태 모니터링

### 💰 **정밀 매도 주문 시스템**
- 소수점 15자리까지 정확한 수량 입력
- 실시간 주문 생성/수정/삭제
- 주문 상태 추적
- 자동 계산 및 검증

### 🔒 **군사급 보안**
- AES-256 암호화
- 로컬 데이터 저장
- 마스터 키 인증
- 민감 정보 마스킹

## 🏗️ 아키텍처

### 📁 프로젝트 구조
```
bitcoin-trader/
├── main.go                 # 애플리케이션 진입점
├── config/                 # 설정 관리
│   └── config.go
├── models/                 # 데이터 모델
│   └── models.go
├── services/              # 비즈니스 로직
│   └── data_service.go
├── utils/                 # 유틸리티
│   ├── crypto.go          # 암호화 서비스
│   └── helpers.go         # 헬퍼 함수
└── ui/                    # 사용자 인터페이스
    ├── theme.go           # 모던 테마
    ├── components.go      # 재사용 컴포넌트
    ├── screens.go         # 화면 관리
    └── dialogs.go         # 다이얼로그
```

### 🔧 계층 분리
- **UI Layer**: 사용자 인터페이스 및 이벤트 처리
- **Service Layer**: 비즈니스 로직 및 데이터 처리
- **Model Layer**: 데이터 구조 정의
- **Utils Layer**: 공통 유틸리티 및 헬퍼
- **Config Layer**: 애플리케이션 설정

## 🚀 빠른 시작

### 📋 시스템 요구사항
- **Go**: 1.21 이상
- **OS**: Windows 10+, macOS 10.15+, Linux
- **메모리**: 최소 512MB RAM
- **저장공간**: 50MB 이상

### 💿 설치 및 실행

#### **방법 1: 바이너리 다운로드**
1. [Releases](https://github.com/your-repo/releases)에서 최신 버전 다운로드
2. 실행 파일을 더블클릭하여 실행

#### **방법 2: 소스코드 빌드**
```bash
# 프로젝트 클론
git clone https://github.com/your-repo/bitcoin-trader.git
cd bitcoin-trader

# 의존성 설치
go mod tidy

# macOS/Linux 빌드
go build -o bitcoin_trader main.go

# Windows 빌드 (macOS에서)
./build_windows_mac.sh

# 실행
./bitcoin_trader
```

## 🔧 빌드 가이드

### 🍎 **macOS에서 Windows용 빌드**
```bash
# mingw-w64 설치
brew install mingw-w64

# Windows 실행 파일 생성
./build_windows_mac.sh
```

### 🐧 **Linux 빌드**
```bash
# 필요한 라이브러리 설치 (Ubuntu/Debian)
sudo apt-get install gcc pkg-config libgl1-mesa-dev xorg-dev

# 빌드
go build -o bitcoin_trader main.go
```

### 🐳 **Docker 빌드**
```bash
# Docker 이미지 빌드
docker build -t bitcoin-trader .

# 컨테이너 실행
docker run -it bitcoin-trader
```

## 📖 사용법

### 1️⃣ **첫 실행**
- 애플리케이션 실행 후 마스터 키 설정
- 마스터 키는 6자리 이상 권장 (예: `123123`)

### 2️⃣ **거래소 등록**
- 대시보드에서 "🏢 거래소 추가" 클릭
- 거래소 이름, API Key, Secret Key 입력
- API는 읽기 전용 권한 권장

### 3️⃣ **매도 주문 생성**
- 거래소 카드에서 "💰 매도 주문" 클릭
- BTC 수량 (소수점 15자리까지)
- 매도 가격 (KRW) 입력

### 4️⃣ **주문 관리**
- 실시간 주문 현황 모니터링
- 주문 수정/삭제 기능
- 통계 대시보드로 한눈에 파악

## 🎨 UI 미리보기

### 🔐 **로그인 화면**
- 깔끔한 카드 기반 레이아웃
- 기능 소개 및 보안 안내
- 엔터키 지원

### 📊 **메인 대시보드**
- 실시간 통계 카드 (거래소 수, 활성 주문, 총 가치)
- 좌우 분할 레이아웃
- 거래소 관리 섹션
- 주문 관리 섹션

### 💬 **다이얼로그**
- 모던하고 직관적인 입력 폼
- 실시간 유효성 검증
- 도움말 및 가이드 제공

## 🔒 보안 기능

### 🛡️ **데이터 보호**
- **AES-256 암호화**: 모든 민감 데이터 암호화
- **로컬 저장**: 데이터가 사용자 컴퓨터를 벗어나지 않음
- **키 마스킹**: API Key 자동 마스킹 처리
- **메모리 보안**: 민감 정보 메모리 자동 정리

### 🔐 **인증**
- 마스터 키 기반 인증
- 키 유효성 검증 (최소 6자리)
- 암호화 실패 시 자동 차단

## ⚡ 성능 최적화

### 🚀 **속도 개선**
- 고루틴 기반 비동기 처리
- 메모리 풀링으로 GC 압박 감소
- 지연 로딩으로 초기 실행 속도 향상
- 효율적인 데이터 구조 사용

### 📊 **메모리 관리**
- sync.RWMutex로 동시성 제어
- 슬라이스 재할당 최소화
- 인터페이스 기반 추상화
- 가비지 컬렉션 최적화

## 🧪 테스트

### 🔍 **품질 보증**
```bash
# 유닛 테스트 실행
go test ./...

# 벤치마크 테스트
go test -bench=. ./...

# 커버리지 확인
go test -cover ./...

# 레이스 컨디션 검사
go test -race ./...
```

## 🐛 문제 해결

### ❓ **자주 묻는 질문**

**Q: 마스터 키를 잊어버렸어요**
A: 데이터 파일(`bitcoin_trader_data.json`)을 삭제하고 다시 시작하세요.

**Q: Windows에서 실행이 안 됩니다**
A: Visual C++ Redistributable을 설치하거나 관리자 권한으로 실행해보세요.

**Q: API Key가 작동하지 않습니다**
A: 거래소에서 발급받은 정확한 키인지 확인하고, 권한 설정을 확인하세요.

### 🔧 **디버깅**
```bash
# 디버그 모드로 실행
go run -race main.go

# 로그 레벨 설정
export LOG_LEVEL=debug
./bitcoin_trader
```

## 🤝 기여하기

### 📝 **개발 가이드**
1. 이슈 생성 또는 기존 이슈 확인
2. 프로젝트 포크 및 브랜치 생성
3. 코드 작성 및 테스트
4. Pull Request 제출

### 🎯 **코딩 스타일**
- Go 표준 컨벤션 준수
- 함수/변수명은 한글 주석 포함
- 테스트 코드 필수 작성
- 인터페이스 우선 설계

## 📄 라이선스

MIT License - 자유롭게 사용, 수정, 배포 가능

## 🙏 감사의 말

- **Fyne**: 훌륭한 Go GUI 프레임워크
- **Go 커뮤니티**: 지속적인 지원과 피드백
- **사용자들**: 소중한 의견과 버그 리포트

---

<div align="center">

**⭐ 이 프로젝트가 도움이 되었다면 스타를 눌러주세요! ⭐**

Made with ❤️ in Go | Powered by Fyne | Secured by AES-256

</div> 