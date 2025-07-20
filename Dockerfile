# Windows용 Go 애플리케이션 빌드를 위한 Dockerfile
FROM golang:1.21-alpine AS builder

# 필요한 패키지 설치
RUN apk add --no-cache git gcc musl-dev

# 작업 디렉토리 설정
WORKDIR /app

# Go 모듈 파일 복사
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사
COPY . .

# Windows용 빌드 (CGO 비활성화)
RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -installsuffix cgo -o bitcoin_trader.exe .

# 최종 이미지 (빌드 결과물만 포함)
FROM scratch
COPY --from=builder /app/bitcoin_trader.exe /bitcoin_trader.exe 