FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o server-monitor ./cmd/server-monitor

FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/server-monitor /app/server-monitor
COPY config.yaml /app/config.yaml

EXPOSE 8742

CMD ["/app/server-monitor"]
