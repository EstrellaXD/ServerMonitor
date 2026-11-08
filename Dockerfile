FROM node:20-alpine AS frontend

WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM golang:1.23-alpine AS backend

WORKDIR /build

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/cmd/ cmd/
COPY backend/internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o server-monitor ./cmd/server-monitor

FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend /build/server-monitor /app/server-monitor
COPY --from=frontend /build/dist /app/static

EXPOSE 8742

CMD ["/app/server-monitor"]
