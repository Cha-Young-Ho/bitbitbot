# Bitbit App

암호화폐 자동 매도 프로그램

## S3 설정

프로그램은 시작 시 S3에서 설정 파일을 읽어와서 상태를 체크합니다.

### S3 설정 파일 형식

```json
{
  "running": "off",  // "on", "off", "all"
  "whiteList": [
    "ab", "id1"
  ],
  "mainVer": "1.1.1"
}
```

### 설정 항목

- **running**: 프로그램 실행 상태
  - `"off"`: 프로그램 실행 불가 (Invalid Request 표시)
  - `"on"`: 화이트리스트 체크 후 실행
  - `"all"`: 모든 사용자 허용
- **whiteList**: 허용된 사용자 ID 목록 (running이 "on"일 때만 사용)
- **mainVer**: 최소 요구 버전 (현재 버전보다 높으면 Invalid Version 표시)

## 빌드 및 실행

### 1. 빌드 스크립트 사용 (권장)

#### Linux/Mac
```bash
# 사용법: ./build.sh [버전] [S3_BUCKET] [S3_KEY]
./build.sh 1.1.1 my-bucket config.json
```

#### Windows
```cmd
REM 사용법: build.bat [버전] [S3_BUCKET] [S3_KEY]
build.bat 1.1.1 my-bucket config.json
```

### 2. 수동 빌드

```bash
# 빌드 (S3 설정과 버전 정보 주입)
go build -ldflags="-X main.Version=1.1.1 -X main.s3Bucket=my-bucket -X main.s3Key=config.json" -o bitbit-app

# 실행
./bitbit-app
```

### 3. 배포

#### 자동 패키지 생성 (권장)

##### Linux/Mac
```bash
# 사용법: ./package.sh [버전] [S3_BUCKET] [S3_KEY]
./package.sh 1.1.1 my-bucket config.json
```

##### Windows
```cmd
REM 사용법: package.bat [버전] [S3_BUCKET] [S3_KEY]
package.bat 1.1.1 my-bucket config.json
```

#### 수동 배포

빌드 후 exe 파일만 배포하면 됩니다:
- `bitbit-app` (또는 `bitbit-app.exe`)

S3 설정과 버전 정보가 exe 파일에 내장되어 있어 별도의 설정 파일이 필요하지 않습니다.

## 기능

### 프로그램 시작 시 체크
1. **S3 설정 로드**: S3에서 설정 파일을 읽어옵니다
2. **Running 상태 체크**: "off"이면 프로그램이 종료됩니다
3. **버전 체크**: 현재 버전이 mainVer보다 낮으면 종료됩니다

### 로그인 시 체크
1. **사용자 권한 체크**: whiteList에 사용자 ID가 있는지 확인합니다
2. **에러 메시지**: 권한이 없으면 "Invalid Account" 표시됩니다

## 에러 메시지

- **"Invalid Request"**: S3 설정의 running이 "off"인 경우
- **"Invalid Account"**: 사용자 ID가 whiteList에 없는 경우
- **"Invalid Version"**: 현재 버전이 S3의 mainVer보다 낮은 경우

## 개발 모드

```bash
wails dev
```
