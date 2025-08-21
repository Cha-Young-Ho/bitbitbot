# Bitbit App

암호화폐 자동 매도 프로그램

## 자동 빌드 및 배포

이 프로젝트는 GitHub Actions를 통해 자동 빌드 및 배포가 설정되어 있습니다.

### 배포 방법

1. **태그 생성 및 푸시**:
   ```bash
   git tag v0.0.7
   git push origin v0.0.7
   ```

2. **자동 실행 과정**:
   - GitHub Actions가 트리거됨
   - macOS 및 Windows 빌드 실행
   - S3에 빌드 파일 업로드
   - config.json 자동 업데이트
   - GitHub Release 생성

### 버전 관리 규칙

- **Major 버전 변경** (1.2.3 → 2.2.3): `mainVer`와 `minVer` 모두 업데이트
- **Minor 버전 변경** (1.2.3 → 1.3.3): `mainVer`만 업데이트  
- **Patch 버전 변경** (1.2.3 → 1.2.4): `mainVer`만 업데이트

### 설정

GitHub Secrets 설정은 [GITHUB_SECRETS_SETUP.md](./GITHUB_SECRETS_SETUP.md)를 참조하세요.

## S3 설정

프로그램은 시작 시 S3에서 설정 파일을 읽어와서 상태를 체크합니다.

### S3 설정 파일 형식

```json
{
  "running": "off",  // "on", "off", "all"
  "whiteList": [
    "ab", "id1"
  ],
  "mainVer": "1.1.1",  // 선택적 업데이트 버전
  "minVer": "1.0.0",   // 필수 업데이트 버전
  "updateUrl": "https://my-bucket.s3.ap-northeast-2.amazonaws.com/prod/mac_build.1.1.1",
  "updatePath": "prod/mac_build.1.1.1",
  "forceUpdate": false
}
```

### 설정 항목

- **running**: 프로그램 실행 상태
  - `"off"`: 프로그램 실행 불가 (Invalid Request 표시)
  - `"on"`: 화이트리스트 체크 후 실행
  - `"all"`: 모든 사용자 허용
- **whiteList**: 허용된 사용자 ID 목록 (running이 "on"일 때만 사용)
- **mainVer**: 선택적 업데이트 버전 (현재 버전보다 높으면 업데이트 다이얼로그 표시)
- **minVer**: 필수 업데이트 버전 (현재 버전보다 높으면 필수 업데이트 다이얼로그 표시)
- **updateUrl**: 업데이트 파일 다운로드 URL (선택적, 자동 생성됨)
- **updatePath**: S3 업데이트 파일 경로 (선택적)
- **forceUpdate**: 강제 업데이트 여부 (true/false)

## 자동 업데이트 기능

프로그램은 시작 시와 주기적으로 버전을 체크하여 자동 업데이트를 수행합니다.

### 업데이트 종류

1. **필수 업데이트 (minVer)**
   - 현재 버전이 minVer보다 낮으면 필수 업데이트 다이얼로그 표시
   - "업데이트" 또는 "취소" 버튼 제공
   - 취소 시 프로그램 종료

2. **선택적 업데이트 (mainVer)**
   - 현재 버전이 mainVer보다 낮으면 선택적 업데이트 다이얼로그 표시
   - "업데이트" 또는 "취소" 버튼 제공
   - 취소 시 기존 흐름대로 프로그램 실행

### 업데이트 프로세스

1. **버전 체크**: 현재 버전과 S3의 minVer/mainVer 비교
2. **다이얼로그 표시**: 업데이트 필요 시 사용자에게 선택권 제공
3. **업데이트 수행**: 사용자가 업데이트 선택 시 S3에서 새 버전 다운로드
4. **백업 생성**: 현재 실행 파일을 .backup으로 백업
5. **파일 교체**: 새 버전으로 실행 파일 교체
6. **자동 재시작**: 새 버전으로 프로그램 재시작

### 환경별 빌드 경로

S3에 업데이트 파일을 업로드할 때는 다음 형식을 사용합니다:

```
my-bucket/
├── prod/
│   ├── config.json               # 프로덕션 설정 파일
│   ├── win_build.1.0.1.exe      # Windows 프로덕션 빌드
│   ├── mac_build.1.0.1          # Mac 프로덕션 빌드
│   └── linux_build.1.0.1        # Linux 프로덕션 빌드
├── dev/
│   ├── config.json               # 개발 설정 파일
│   ├── win_build.1.0.1.exe      # Windows 개발 빌드
│   ├── mac_build.1.0.1          # Mac 개발 빌드
│   └── linux_build.1.0.1        # Linux 개발 빌드
└── test/
    ├── config.json               # 테스트 설정 파일
    ├── win_build.1.0.1.exe      # Windows 테스트 빌드
    ├── mac_build.1.0.1          # Mac 테스트 빌드
    └── linux_build.1.0.1        # Linux 테스트 빌드
```

### 설정 예시

```json
{
  "running": "on",
  "whiteList": ["user1", "user2"],
  "mainVer": "1.2.0",
  "minVer": "1.0.0",
  "forceUpdate": false
}
```

## 빌드 및 실행

### 1. 빌드 스크립트 사용 (권장)

#### Linux/Mac
```bash
# 사용법: ./build-mac.sh [버전] [환경] [S3_BUCKET]
./build-mac.sh 1.1.1 prod my-bucket
```

#### Windows
```bash
# 사용법: ./build-window.sh [버전] [환경] [S3_BUCKET]
./build-window.sh 1.1.1 prod my-bucket
```

### 2. 수동 빌드

```bash
# 빌드 (환경, 버전, 설정 URL 주입)
go build -ldflags="-X main.Version=1.1.1 -X main.Environment=prod -X main.configUrl=https://my-bucket.s3.ap-northeast-2.amazonaws.com/prod/config.json" -o bitbit-app

# 실행
./bitbit-app
```

### 3. 배포

#### 배포

빌드 후 실행 파일과 S3 설정을 준비하면 됩니다:

1. **실행 파일 배포**:
   - `bitbit-app` (Mac/Linux)
   - `bitbit-app.exe` (Windows)

2. **S3 구조 준비**:
   ```
   my-bucket/
   ├── prod/
   │   ├── config.json               # 프로덕션 설정
   │   ├── mac_build.1.1.1          # Mac 빌드 결과물
   │   └── win_build.1.1.1.exe      # Windows 빌드 결과물
   ```

환경별 설정과 업데이트 파일이 S3에 구성되어 있어 별도의 설정 파일이 필요하지 않습니다.

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
