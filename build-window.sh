#!/bin/bash

# Windows 빌드 스크립트
# 사용법: ./build-window.sh <VERSION> <ENVIRONMENT> <BUCKET_NAME>
# 예시: ./build-window.sh 0.0.7 prod aa

set -e

# 인자 확인
if [ $# -ne 3 ]; then
    echo "사용법: $0 <VERSION> <ENVIRONMENT> <BUCKET_NAME>"
    echo "예시: $0 0.0.7 prod aa"
    exit 1
fi

VERSION=$1
ENVIRONMENT=$2
BUCKET_NAME=$3

echo "Windows 빌드 시작: VERSION=$VERSION, ENVIRONMENT=$ENVIRONMENT, BUCKET_NAME=$BUCKET_NAME"

# 빌드 디렉토리 생성
mkdir -p build

# S3 URL 생성
if [ ! -z "$BUCKET_NAME" ]; then
    UPDATE_URL="https://${BUCKET_NAME}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/window_build.${VERSION}.zip"
    echo "업데이트 URL: $UPDATE_URL"
fi

# Wails 빌드 (Windows용)
echo "Wails 빌드 시작..."
wails build -platform windows/amd64 -ldflags "-X main.Version=${VERSION} -X main.Environment=${ENVIRONMENT} -X main.configUrl=https://${BUCKET_NAME}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/config.json"

# 빌드 결과 확인
if [ ! -f "build/bin/bitbit-app.exe" ]; then
    echo "빌드 실패: bitbit-app.exe 파일이 생성되지 않았습니다."
    exit 1
fi

echo "Windows 빌드 완료: build/bin/bitbit-app.exe"

# 친구용 배포 패키지 생성
echo "친구용 배포 패키지 생성 중..."

# 배포 디렉토리 생성
DEPLOY_DIR="build/bitbit-app-windows-${VERSION}"
mkdir -p "$DEPLOY_DIR"

# 실행파일 복사
cp "build/bin/bitbit-app.exe" "$DEPLOY_DIR/"

# README 파일 생성
cat > "$DEPLOY_DIR/README.txt" << EOF
BitBit Bot - Windows 실행파일
버전: ${VERSION}
환경: ${ENVIRONMENT}

사용법:
1. 이 폴더의 모든 파일을 윈도우 컴퓨터의 원하는 위치에 복사하세요
2. bitbit-app.exe 파일을 더블클릭하여 실행하세요
3. 프로그램이 자동으로 시작됩니다

주의사항:
- Windows 10/11에서 실행됩니다
- 바이러스 백신 프로그램에서 차단될 수 있습니다 (정상적인 프로그램입니다)
- 실행이 안 될 경우 우클릭 > "관리자 권한으로 실행"을 시도해보세요

문제가 발생하면 개발자에게 문의하세요.
EOF

# 실행 배치 파일 생성 (Windows용)
cat > "$DEPLOY_DIR/실행.bat" << EOF
@echo off
echo BitBit Bot을 시작합니다...
start bitbit-app.exe
echo 프로그램이 백그라운드에서 실행됩니다.
pause
EOF

# 친구용 ZIP 파일 생성
cd build
zip -r "bitbit-app-windows-${VERSION}.zip" "bitbit-app-windows-${VERSION}"
cd ..

echo "친구용 배포 패키지 생성 완료!"
echo "배포 디렉토리: $DEPLOY_DIR"
echo "ZIP 파일: build/bitbit-app-windows-${VERSION}.zip"
echo ""
echo "친구에게 전달할 파일: build/bitbit-app-windows-${VERSION}.zip"
echo "압축을 풀면 바로 실행할 수 있습니다!"

echo "Windows 빌드 완료!"
