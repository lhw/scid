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
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck

# Pre-create the data directory owned by the distroless nonroot uid (65532).
RUN mkdir -p /data && chown 65532:65532 /data

# ---------------------------------------------------------------------------
# Stage 3: Distroless runtime image (no shell, no package manager)
# gcr.io/distroless/static-debian13 bundles ca-certificates and tzdata only.
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian13:nonroot AS runner

COPY --from=go-builder /scid /scid
COPY --from=go-builder /healthcheck /healthcheck
COPY --from=go-builder --chown=65532:65532 /data /data

# Data volume for companion runtime data and cached logos
VOLUME ["/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/healthcheck"]

ENTRYPOINT ["/scid"]
