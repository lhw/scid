# =============================================================================
# Multi-stage build — SvelteKit frontend + Go companion in one image
# Build context: repository root  (docker build -f companion/Dockerfile .)
# =============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Build SvelteKit frontend
# ---------------------------------------------------------------------------
FROM node:22-alpine AS frontend-builder

RUN npm install -g pnpm

WORKDIR /frontend

# Install deps first so this layer is cached independently of source changes.
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm build

# ---------------------------------------------------------------------------
# Stage 2: Build Go binary
# The frontend output is copied into the //go:embed path before go build runs.
# ---------------------------------------------------------------------------
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY companion/go.mod companion/go.sum ./
RUN go mod download

COPY companion/ ./

# Overlay the real frontend build into the embed directory so //go:embed picks it up.
COPY --from=frontend-builder /frontend/build ./internal/frontend/dist/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /scid ./cmd/scid

# ---------------------------------------------------------------------------
# Stage 3: Minimal runtime image
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS runner

RUN apk add --no-cache ca-certificates tzdata curl

# Run as non-root
RUN addgroup -S scid && adduser -S scid -G scid

WORKDIR /app

COPY --from=go-builder /scid /app/scid

# Data volume for companion runtime data and cached logos
VOLUME ["/data"]
RUN mkdir -p /data && chown scid:scid /data

USER scid

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -sf http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/scid"]
