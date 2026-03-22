FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies for the companion build
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /scid ./cmd/scid

# ---------------------------------------------------------------------------
FROM alpine:3.20 AS runner

RUN apk add --no-cache ca-certificates tzdata curl

# Run as non-root
RUN addgroup -S scid && adduser -S scid -G scid

WORKDIR /app

COPY --from=builder /scid /app/scid

# Data volume for companion runtime data and cached logos
VOLUME ["/data"]
RUN mkdir -p /data && chown scid:scid /data

USER scid

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -sf http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/scid"]
