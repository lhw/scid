# =============================================================================
# Caddy — custom build with replace-response and cloudflare-dns plugins
# =============================================================================
FROM caddy:2-builder AS builder

RUN xcaddy build \
    --with github.com/caddyserver/replace-response \
    --with github.com/caddy-dns/cloudflare

FROM caddy:2-alpine
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
