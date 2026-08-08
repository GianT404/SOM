FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o som-backend ./cmd/server/main.go

FROM alpine:latest

RUN apk add --no-cache python3 py3-pip ffmpeg ca-certificates \
    && pip install --no-cache-dir --break-system-packages yt-dlp \
    && adduser -D -u 10001 somuser \
    && mkdir -p /var/cache/som/yt-dlp \
    && chown -R somuser:somuser /var/cache/som/yt-dlp

WORKDIR /app

COPY --from=builder /app/som-backend .

USER somuser

EXPOSE 8080

CMD ["./som-backend"]