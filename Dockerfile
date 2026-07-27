FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o /app/web ./cmd/web/main.go
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o /app/worker ./cmd/worker/main.go


FROM alpine:3.22 AS web
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/web /app/web
COPY .env.production /app/.env
USER appuser
ENV TZ=Asia/Jakarta
EXPOSE 9004
CMD ["/app/web"]

FROM alpine:3.22 AS worker
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/worker /app/worker
COPY .env.production /app/.env
USER appuser
ENV TZ=Asia/Jakarta
CMD ["/app/worker"]