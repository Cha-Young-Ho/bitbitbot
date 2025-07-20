#!/bin/bash

# macOS에서 Windows용 Bitcoin Trader 빌드 스크립트

echo "🔨 macOS에서 Windows용 Bitcoin Trader Professional 빌드 시작..."

# mingw-w64 설치 확인
if ! command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo "❌ mingw-w64가 설치되지 않았습니다."
    echo "📦 다음 명령어로 설치하세요:"
    echo "   brew install mingw-w64"
    exit 1
fi

echo "✅ mingw-w64 확인됨"

# 의존성 확인
echo "📦 의존성 확인 중..."
go mod tidy

# 이전 빌드 파일 삭제
if [ -f "bitcoin_trader.exe" ]; then
    echo "🗑️  이전 빌드 파일 삭제 중..."
    rm bitcoin_trader.exe
fi

# Windows용 빌드 (CGO 활성화 + mingw-w64 사용)
echo "🏗️  Windows용 실행 파일 빌드 중..."
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags="-s -w -H windowsgui" -o bitcoin_trader.exe main.go

if [ $? -eq 0 ]; then
    echo "✅ 빌드 성공! bitcoin_trader.exe 파일이 생성되었습니다."
    
    # 파일 정보 표시
    echo "📁 파일 크기: $(ls -lh bitcoin_trader.exe | awk '{print $5}')"
    echo "🔍 파일 타입: $(file bitcoin_trader.exe)"
    echo ""
    echo "🚀 Windows에서 실행하려면:"
    echo "   1. bitcoin_trader.exe 파일을 Windows 컴퓨터로 복사"
    echo "   2. 더블클릭하여 실행"
    echo ""
    echo "✨ 새로운 기능:"
    echo "   📈 실시간 대시보드 - 주식창 스타일 UI"
    echo "   🎨 화이트 베이스 테마 - 깔끔한 디자인"
    echo "   📊 통계 카드 - 한눈에 보는 정보"
    echo "   📋 테이블 형태 - 체계적인 데이터 관리"
    echo "   ⚡ 빠른 액션 - 원클릭 기능"
    echo ""
    echo "💾 데이터는 사용자 홈 디렉토리에 암호화되어 저장됩니다."
else
    echo "❌ 빌드 실패"
    echo ""
    echo "🔧 해결 방법:"
    echo "   1. mingw-w64가 제대로 설치되었는지 확인"
    echo "   2. Go 모듈 의존성 확인: go mod tidy"
    echo "   3. 수동 빌드: CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -o bitcoin_trader.exe main.go"
fi 