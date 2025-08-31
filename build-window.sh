#!/bin/bash

# Windows 빌드 스크립트
# 사용법: ./build-window.sh <VERSION> <ENVIRONMENT> <BUCKET_NAME>
# 예시: ./build-window.sh 0.0.7 prod yh-bitbit-bucket

set -e

# 인자 확인
if [ $# -ne 3 ]; then
    echo "사용법: $0 <VERSION> <ENVIRONMENT> <BUCKET_NAME>"
    echo "예시: $0 0.0.7 prod yh-bitbit-bucket"
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

# S3 업로드용 파일 생성
if [ ! -z "$UPDATE_URL" ]; then
    S3_FILENAME="window_build.${VERSION}"
    S3_FILEPATH="build/${S3_FILENAME}"
    S3_ZIP_FILENAME="window_build.${VERSION}.zip"
    S3_ZIP_FILEPATH="build/${S3_ZIP_FILENAME}"
    
    # 실행 파일을 S3 업로드용으로 복사
    cp "build/bin/bitbit-app.exe" "$S3_FILEPATH"
    chmod +x "$S3_FILEPATH"
    
    # zip 파일 생성
    cd build
    zip "$S3_ZIP_FILENAME" "$S3_FILENAME"
    cd ..
    
    echo "S3 업로드용 파일 생성: $S3_FILEPATH"
    echo "S3 업로드용 ZIP 파일 생성: $S3_ZIP_FILEPATH"
    echo "업데이트 URL: $UPDATE_URL"
fi

echo "Windows 빌드 완료!"
