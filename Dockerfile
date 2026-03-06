# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build từ file main.go trong cmd/server
RUN go build -o server ./cmd/server/main.go

# Run stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
# Cài đặt yt-dlp và ffmpeg nếu backend cần xử lý video/audio [cite: 2026-03-05]
RUN apk add --no-cache ca-certificates ffmpeg python3 && \
    wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

EXPOSE 8080
CMD ["./server"]