#!/bin/bash

# Windows용 빌드 스크립트
echo "Windows용 빌드 시작..."

# WAV 파일이 있는지 확인
if [ ! -f "sound/trade_success.wav" ]; then
    echo "WAV 파일이 없습니다. sound/trade_success.wav 파일을 추가해주세요."
    exit 1
fi

# Wails 빌드 (Windows)
echo "Wails 빌드 중..."
wails build -platform windows/amd64

# 빌드 성공 확인
if [ $? -eq 0 ]; then
    echo "빌드 완료!"
    
    # WAV 파일을 실행 파일과 같은 위치로 복사
    echo "WAV 파일을 실행 파일과 같은 위치로 복사 중..."
    cp "sound/trade_success.wav" "build/bin/trade_success.wav"
    
    # sound 폴더도 생성 (폴더 구조 유지)
    mkdir -p "build/bin/sound"
    cp "sound/trade_success.wav" "build/bin/sound/trade_success.wav"
    
    echo "✅ 빌드 완료!"
    echo "실행 파일 위치: build/bin/bitbit-app.exe"
    echo "WAV 파일 위치: build/bin/trade_success.wav (실행 파일과 같은 위치)"
    echo "WAV 파일 위치: build/bin/sound/trade_success.wav (폴더 구조 유지)"
    echo ""
    echo "윈도우에서 실행하려면:"
    echo "cd build/bin"
    echo "bitbit-app.exe"
    echo ""
    echo "zip 파일 생성:"
    echo "cd build/bin && zip -r bitbit-app-windows.zip ."
else
    echo "빌드 실패!"
    exit 1
fi
