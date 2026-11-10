# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /stemhub-backend ./cmd/api


FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /stemhub-backend ./stemhub-backend

# Manual de usuario que entrega GET /ayuda/manual (MANUAL_USUARIO_PATH).
COPY --from=builder /app/assets/manual ./assets/manual

EXPOSE 8080

CMD ["./stemhub-backend"]