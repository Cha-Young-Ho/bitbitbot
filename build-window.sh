#!/bin/bash

# Windows용 빌드 스크립트
# 사용법:
#   1) ./build-window.sh [버전] [CONFIG_URL]
#   2) ./build-window.sh [버전] [S3_BUCKET] [S3_KEY] (자동으로 URL 생성)
#   3) ./build-window.sh (대화형 입력)

validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "❌ 잘못된 버전 형식입니다. (예: 0.0.1, 1.2.3)"
        return 1
    fi
    return 0
}

validate_not_empty() {
    local value=$1
    local field_name=$2
    if [[ -z "$value" ]]; then
        echo "❌ $field_name은(는) 비어있을 수 없습니다."
        return 1
    fi
    return 0
}

VERSION=""
CONFIG_URL=""

if [ $# -eq 2 ]; then
    VERSION=$1; CONFIG_URL=$2
    echo "빌드 정보:"; echo "  버전: $VERSION"; echo "  설정 URL: $CONFIG_URL"; echo ""
    if ! validate_version "$VERSION"; then exit 1; fi
    if ! validate_not_empty "$CONFIG_URL" "설정 URL"; then exit 1; fi
elif [ $# -eq 3 ]; then
    VERSION=$1; S3_BUCKET=$2; S3_KEY=$3
    # 버킷과 키가 제공되면 자동으로 URL 생성
    CONFIG_URL="https://${S3_BUCKET}.s3.ap-northeast-2.amazonaws.com/${S3_KEY}"
    echo "빌드 정보:"; echo "  버전: $VERSION"; echo "  S3 Bucket: $S3_BUCKET"; echo "  S3 Key: $S3_KEY"; echo "  생성된 URL: $CONFIG_URL"; echo ""
    if ! validate_version "$VERSION"; then exit 1; fi
    if ! validate_not_empty "$S3_BUCKET" "S3 Bucket"; then exit 1; fi
    if ! validate_not_empty "$S3_KEY" "S3 Key"; then exit 1; fi
else
    echo "=== Windows용 빌드 스크립트 ==="; echo ""
    while true; do read -p "버전을 입력하세요 (예: 0.0.1): " VERSION; if validate_version "$VERSION"; then break; fi; done
    read -p "설정 URL을 직접 입력하시겠습니까? (입력 시 URL 우선, 미입력 시 Bucket/Key 사용): " CONFIG_URL
    if [[ -z "$CONFIG_URL" ]]; then
        while true; do read -p "S3 Bucket을 입력하세요: " S3_BUCKET; if validate_not_empty "$S3_BUCKET" "S3 Bucket"; then break; fi; done
        while true; do read -p "S3 Key를 입력하세요 (예: prod/config.json): " S3_KEY; if validate_not_empty "$S3_KEY" "S3 Key"; then break; fi; done
        CONFIG_URL="https://${S3_BUCKET}.s3.ap-northeast-2.amazonaws.com/${S3_KEY}"
        echo "입력된 정보:"; echo "  버전: $VERSION"; echo "  S3 Bucket: $S3_BUCKET"; echo "  S3 Key: $S3_KEY"; echo "  생성된 URL: $CONFIG_URL"; echo ""
    else
        echo "입력된 정보:"; echo "  버전: $VERSION"; echo "  설정 URL: $CONFIG_URL"; echo ""
    fi
fi

echo "빌드 시작..."; echo "버전: $VERSION"; echo "설정 URL: $CONFIG_URL"; echo ""

# Windows용 Wails 빌드 실행 (버전 및 설정 URL 주입)
GOOS=windows GOARCH=amd64 wails build -ldflags="-X main.Version=$VERSION -X main.configUrl=$CONFIG_URL"

if [ $? -eq 0 ]; then
    echo ""; echo "✅ 빌드 완료: build/bin/bitbit-app.exe"; echo "📁 파일 크기: $(ls -lh build/bin/bitbit-app.exe | awk '{print $5}')"; echo "🚀 exe 파일만 배포하면 됩니다. 설정이 내장되어 있습니다."
else
    echo ""; echo "❌ 빌드 실패!"; exit 1
fi 