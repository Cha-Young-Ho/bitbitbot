#!/bin/bash

# Mac용 빌드 스크립트
# 사용법:
#   1) ./build-mac.sh [버전] [환경] [CONFIG_URL]
#   2) ./build-mac.sh [버전] [환경] [S3_BUCKET] (환경별 config.json 자동 생성)
#   3) ./build-mac.sh (대화형 입력)

validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "❌ 잘못된 버전 형식입니다. (예: 0.0.1, 1.2.3)"
        return 1
    fi
    return 0
}

validate_environment() {
    local env=$1
    if [[ ! $env =~ ^(prod|dev|test|staging)$ ]]; then
        echo "❌ 잘못된 환경입니다. (예: prod, dev, test, staging)"
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
ENVIRONMENT=""
CONFIG_URL=""
UPDATE_URL=""

# 인자 처리
if [ $# -eq 3 ]; then
    VERSION=$1; ENVIRONMENT=$2; THIRD_ARG=$3
    
    # 세 번째 인자가 URL인지 S3 버킷인지 확인
    if [[ $THIRD_ARG == *"http"* ]]; then
        # URL인 경우
        CONFIG_URL=$THIRD_ARG
        echo "빌드 정보:"; echo "  버전: $VERSION"; echo "  환경: $ENVIRONMENT"; echo "  설정 URL: $CONFIG_URL"; echo ""
                                                                                                
        if ! validate_version "$VERSION"; then exit 1; fi
        if ! validate_environment "$ENVIRONMENT"; then exit 1; fi
        if ! validate_not_empty "$CONFIG_URL" "설정 URL"; then exit 1; fi
    else
        # S3 버킷인 경우
        S3_BUCKET=$THIRD_ARG
        CONFIG_URL="https://${S3_BUCKET}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/config.json"
        UPDATE_URL="https://${S3_BUCKET}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/mac_build.${VERSION}.zip"

        echo "빌드 정보:"; echo "  버전: $VERSION"; echo "  환경: $ENVIRONMENT"; echo "  S3 Bucket: $S3_BUCKET"; echo "  생성된 설정 URL: $CONFIG_URL"; echo "  생성된 업데이트 URL: $UPDATE_URL"; echo ""              
        if ! validate_version "$VERSION"; then exit 1; fi
        if ! validate_environment "$ENVIRONMENT"; then exit 1; fi
        if ! validate_not_empty "$S3_BUCKET" "S3 Bucket"; then exit 1; fi
    fi
else
    echo "=== Mac용 빌드 스크립트 ==="; echo ""
    
    # 버전 입력
    read -p "버전을 입력하세요 (예: 1.0.0): " VERSION
    if ! validate_version "$VERSION"; then exit 1; fi
    
    # 환경 입력
    read -p "환경을 입력하세요 (prod/dev/test/staging): " ENVIRONMENT
    if ! validate_environment "$ENVIRONMENT"; then exit 1; fi
    
    # 설정 URL 입력 방식 선택
    echo ""; echo "설정 URL 입력 방식을 선택하세요:"
    echo "1) 직접 URL 입력"
    echo "2) S3 버킷과 키로 자동 생성"
    read -p "선택 (1 또는 2): " choice
    
    case $choice in
        1)
            read -p "설정 URL을 입력하세요: " CONFIG_URL
            if ! validate_not_empty "$CONFIG_URL" "설정 URL"; then exit 1; fi
            ;;
        2)
            read -p "S3 버킷을 입력하세요: " S3_BUCKET
            if ! validate_not_empty "$S3_BUCKET" "S3 Bucket"; then exit 1; fi
            CONFIG_URL="https://${S3_BUCKET}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/config.json"
            UPDATE_URL="https://${S3_BUCKET}.s3.ap-northeast-2.amazonaws.com/${ENVIRONMENT}/mac_build.${VERSION}.zip"

            echo "생성된 설정 URL: $CONFIG_URL"
            echo "생성된 업데이트 URL: $UPDATE_URL"
            ;;
        *)
            echo "❌ 잘못된 선택입니다."
            exit 1
            ;;
    esac
fi

echo "=== 빌드 시작 ==="

# 기존 빌드 파일 정리
if [ -d "build" ]; then
    rm -rf build
    echo "기존 빌드 파일 제거 완료"
fi

# 환경 변수 설정
export VERSION=$VERSION
export ENVIRONMENT=$ENVIRONMENT
export CONFIG_URL=$CONFIG_URL

# Wails 빌드 명령어 구성
BUILD_CMD="wails build -ldflags=\"-X main.Version=${VERSION} -X main.Environment=${ENVIRONMENT} -X main.configUrl=${CONFIG_URL}\""

# 업데이트 URL이 있으면 추가
if [ ! -z "$UPDATE_URL" ]; then
    BUILD_CMD="wails build -ldflags=\"-X main.Version=${VERSION} -X main.Environment=${ENVIRONMENT} -X main.configUrl=${CONFIG_URL} -X main.updateUrl=${UPDATE_URL}\""
fi

echo "빌드 명령어: $BUILD_CMD"
echo ""

# 빌드 실행
if eval $BUILD_CMD; then
    echo "✅ 빌드 성공!"
    echo "생성된 파일: build/bin/bitbit-app.app/Contents/MacOS/bitbit-app"
    
    # S3 업로드용 파일 생성
    if [ ! -z "$UPDATE_URL" ]; then
        S3_FILENAME="mac_build.${VERSION}"
        S3_FILEPATH="build/${S3_FILENAME}"
        S3_ZIP_FILENAME="mac_build.${VERSION}.zip"
        S3_ZIP_FILEPATH="build/${S3_ZIP_FILENAME}"
        
        # 실행 파일을 S3 업로드용으로 복사
        cp "build/bin/bitbit-app.app/Contents/MacOS/bitbit-app" "$S3_FILEPATH"
        chmod +x "$S3_FILEPATH"
        
        # zip 파일 생성
        cd build
        zip "$S3_ZIP_FILENAME" "$S3_FILENAME"
        cd ..
        
        echo "S3 업로드용 파일 생성: $S3_FILEPATH"
        echo "S3 업로드용 ZIP 파일 생성: $S3_ZIP_FILEPATH"
        echo "업데이트 URL: $UPDATE_URL"
    fi
    
    echo "버전: $VERSION"
    echo "환경: $ENVIRONMENT"
    echo "설정 URL: $CONFIG_URL"
else
    echo "❌ 빌드 실패!"
    exit 1
fi
