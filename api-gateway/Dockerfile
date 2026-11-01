FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY gen/go ./gen/go
COPY pkg ./pkg
COPY api-gateway ./api-gateway

WORKDIR /app/api-gateway
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/api-gateway ./cmd/app/main.go

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/bin/api-gateway .

USER appuser

EXPOSE 8080

CMD ["./api-gateway"]
