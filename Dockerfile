FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o som-backend ./cmd/server/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/som-backend .

EXPOSE 8080

CMD ["./som-backend"]