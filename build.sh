#!/bin/bash

# 통합 빌드 스크립트
# 사용법: ./build.sh [VERSION] [BUCKET_NAME]
# 또는 ./build.sh (사용자 입력 모드)

set -e

# 사용자 입력 처리
if [ $# -eq 0 ]; then
    # 인자가 없으면 사용자 입력 모드
    echo "=== 빌드 설정 입력 ==="
    echo -n "버전을 입력해주세요 : "
    read VERSION
    echo -n "환경을 입력해주세요 : "
    read ENVIRONMENT
    if [ -z "$ENVIRONMENT" ]; then
        ENVIRONMENT="prod"
    fi
    echo -n "버킷명을 입력해주세요 : "
    read BUCKET_NAME
    echo ""
elif [ $# -eq 1 ]; then
    # 버전만 입력된 경우
    VERSION=$1
    ENVIRONMENT="prod"
    echo -n "버킷명을 입력해주세요 : "
    read BUCKET_NAME
    echo ""
elif [ $# -eq 2 ]; then
    # 버전과 버킷명이 입력된 경우
    VERSION=$1
    BUCKET_NAME=$2
    ENVIRONMENT="prod"
elif [ $# -eq 3 ]; then
    # 모든 인자가 입력된 경우
    VERSION=$1
    ENVIRONMENT=$2
    BUCKET_NAME=$3
else
    echo "사용법: $0 [VERSION] [BUCKET_NAME]"
    echo "또는 $0 (사용자 입력 모드)"
    exit 1
fi

# 입력값 검증
if [ -z "$VERSION" ] || [ -z "$BUCKET_NAME" ]; then
    echo "❌ 오류: 버전과 버킷명은 필수입니다."
    exit 1
fi

echo "=== 통합 빌드 시작 ==="
echo "버전: $VERSION"
echo "버킷: $BUCKET_NAME"
echo "환경: $ENVIRONMENT"
echo ""

# 빌드 디렉토리 생성
mkdir -p build

# 공통 설정
CONFIG_URL="https://${BUCKET_NAME}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/config.json"

# 1. macOS 빌드
echo "=== macOS 빌드 시작 ==="
MAC_UPDATE_URL="https://${BUCKET_NAME}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/mac_build.${VERSION}.zip"

echo "macOS 빌드 명령어:"
echo "wails build -ldflags=\"-X main.Version=${VERSION} -X main.Environment=${ENVIRONMENT} -X main.configUrl=${CONFIG_URL} -X main.updateUrl=${MAC_UPDATE_URL}\""

wails build -ldflags="-X main.Version=${VERSION} -X main.Environment=${ENVIRONMENT} -X main.configUrl=${CONFIG_URL} -X main.updateUrl=${MAC_UPDATE_URL}"

# macOS 빌드 결과 확인
if [ ! -f "build/bin/bitbit-app.app/Contents/MacOS/bitbit-app" ]; then
    echo "❌ macOS 빌드 실패: bitbit-app 파일이 생성되지 않았습니다."
    exit 1
fi

echo "✅ macOS 빌드 완료: build/bin/bitbit-app.app/Contents/MacOS/bitbit-app"

# macOS S3 업로드용 파일 생성
MAC_S3_FILENAME="mac_build.${VERSION}"
MAC_S3_FILEPATH="build/${MAC_S3_FILENAME}"
MAC_S3_ZIP_FILENAME="mac_build.${VERSION}.zip"
MAC_S3_ZIP_FILEPATH="build/${MAC_S3_ZIP_FILENAME}"

cp "build/bin/bitbit-app.app/Contents/MacOS/bitbit-app" "$MAC_S3_FILEPATH"
chmod +x "$MAC_S3_FILEPATH"

cd build
zip "$MAC_S3_ZIP_FILENAME" "$MAC_S3_FILENAME"
cd ..

echo "✅ macOS S3 업로드용 파일 생성: $MAC_S3_ZIP_FILEPATH"
echo ""

# 2. Windows 빌드
echo "=== Windows 빌드 시작 ==="
WIN_UPDATE_URL="https://${BUCKET_NAME}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/window_build.${VERSION}.zip"

echo "Windows 빌드 명령어:"
echo "wails build -platform windows/amd64 -ldflags=\"-X main.Version=${VERSION} -X main.Environment=${ENVIRONMENT} -X main.configUrl=${CONFIG_URL} -X main.updateUrl=${WIN_UPDATE_URL}\""

wails build -platform windows/amd64 -ldflags="-X main.Version=${VERSION} -X main.Environment=${ENVIRONMENT} -X main.configUrl=${CONFIG_URL} -X main.updateUrl=${WIN_UPDATE_URL}"

# Windows 빌드 결과 확인
if [ ! -f "build/bin/bitbit-app.exe" ]; then
    echo "❌ Windows 빌드 실패: bitbit-app.exe 파일이 생성되지 않았습니다."
    exit 1
fi

echo "✅ Windows 빌드 완료: build/bin/bitbit-app.exe"

# Windows S3 업로드용 파일 생성
WIN_S3_FILENAME="window_build.${VERSION}"
WIN_S3_FILEPATH="build/${WIN_S3_FILENAME}"
WIN_S3_ZIP_FILENAME="window_build.${VERSION}.zip"
WIN_S3_ZIP_FILEPATH="build/${WIN_S3_ZIP_FILENAME}"

cp "build/bin/bitbit-app.exe" "$WIN_S3_FILEPATH"
chmod +x "$WIN_S3_FILEPATH"

cd build
zip "$WIN_S3_ZIP_FILENAME" "$WIN_S3_FILENAME"
cd ..

echo "✅ Windows S3 업로드용 파일 생성: $WIN_S3_ZIP_FILEPATH"
echo ""

# 3. 빌드 결과 요약
echo "=== 빌드 완료 ==="
echo "생성된 파일들:"
echo "  macOS: $MAC_S3_ZIP_FILEPATH"
echo "  Windows: $WIN_S3_ZIP_FILEPATH"
echo ""
echo "S3 업로드 URL:"
echo "  macOS: $MAC_UPDATE_URL"
echo "  Windows: $WIN_UPDATE_URL"
echo ""
echo "✅ 모든 빌드가 완료되었습니다!"
