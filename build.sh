#!/bin/bash

# Windows용 Bitcoin Trader 빌드 스크립트

echo "🔨 Windows용 Bitcoin Trader 빌드 시작..."

# 의존성 확인
echo "📦 의존성 확인 중..."
go mod tidy

# 빌드 시도 (CGO 비활성화)
echo "🏗️  Windows용 실행 파일 빌드 중..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bitcoin_trader.exe main.go

if [ $? -eq 0 ]; then
    echo "✅ 빌드 성공! bitcoin_trader.exe 파일이 생성되었습니다."
    echo "📁 파일 크기: $(ls -lh bitcoin_trader.exe | awk '{print $5}')"
    echo ""
    echo "🚀 Windows에서 실행하려면:"
    echo "   1. bitcoin_trader.exe 파일을 Windows 컴퓨터로 복사"
    echo "   2. 더블클릭하여 실행"
    echo ""
    echo "⚠️  주의: 이 버전은 CGO가 비활성화되어 있어 GUI가 제대로 작동하지 않을 수 있습니다."
    echo "   완전한 GUI 기능을 위해서는 Windows에서 직접 빌드하는 것을 권장합니다."
else
    echo "❌ 빌드 실패"
    echo ""
    echo "🔧 해결 방법:"
    echo "   1. Windows 컴퓨터에서 직접 빌드"
    echo "   2. Docker를 사용한 크로스 컴파일"
    echo "   3. GitHub Actions 사용"
    echo ""
    echo "자세한 내용은 build_windows.md 파일을 참조하세요."
fi 