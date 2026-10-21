FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY gen/go ./gen/go
COPY pkg ./pkg
COPY order-service ./order-service

WORKDIR /app/order-service
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/order-service ./cmd/app/main.go

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/bin/order-service .

USER appuser

EXPOSE 50054

CMD ["./order-service"]
