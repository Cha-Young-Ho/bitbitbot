#!/bin/bash

# 릴리즈 zip 파일 생성 스크립트
echo "릴리즈 zip 파일 생성 시작..."

# Windows용 빌드
echo "Windows용 빌드 중..."
./build-windows.sh

if [ $? -eq 0 ]; then
    echo ""
    echo "zip 파일 생성 중..."
    cd build/bin
    
    # Windows용 zip 파일 생성
    zip -r ../../bitbit-app-windows.zip . -x "*.app" "*.dmg"
    
    echo "✅ 릴리즈 파일 생성 완료!"
    echo "Windows: bitbit-app-windows.zip"
    echo ""
    echo "zip 파일 내용:"
    echo "- bitbit-app.exe (실행 파일)"
    echo "- trade_success.wav (소리 파일 - 실행 파일과 같은 위치)"
    echo "- sound/trade_success.wav (소리 파일 - 폴더 구조 유지)"
    echo "- 기타 필요한 파일들"
    echo ""
    echo "사용법:"
    echo "1. zip 파일을 다운로드"
    echo "2. 원하는 위치에 압축 해제"
    echo "3. bitbit-app.exe 실행"
else
    echo "빌드 실패로 zip 파일을 생성할 수 없습니다."
    exit 1
fi
