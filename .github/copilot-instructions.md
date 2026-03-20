---
applyTo: "**"
---

# SCID — Star Citizen Identity Provider

## Project Context

SCID is an identity provider for Star Citizen fan websites. It uses Pocket ID (a lightweight OIDC provider) as the authentication backend and adds a companion service for RSI bio-based identity verification and self-service OIDC client registration.

See [PLAN.md](../PLAN.md) for the full architecture and design decisions.

## Tech Stack

- **OIDC Provider:** Pocket ID (Go + SvelteKit, deployed via Docker)
- **Companion Service:** Go (standard library or Chi/Echo router)
- **Frontend:** SvelteKit (TypeScript)
- **Database:** PostgreSQL (production) or SQLite (development)
- **Deployment:** Docker Compose with Caddy reverse proxy

## Code Conventions

### Go (Companion Service)

- Use Go 1.22+ with standard library where possible
- Structure: `cmd/` for entrypoints, `internal/` for private packages
- Use `context.Context` for all external calls (HTTP, DB)
- Return `error` values — do not panic in library code
- Use `slog` for structured logging
- HTTP handlers should validate input at the boundary, not deep in business logic
- Database access goes through a `store` package — no SQL in handlers
- Use `net/http` or a lightweight router (Chi) — no heavy frameworks
- Tests go in `_test.go` files next to the code they test

### SvelteKit (Frontend)

- Svelte 5 with TypeScript for all `.ts` and `.svelte` files
- Use SvelteKit's built-in fetch for API calls
- Server-side rendering where possible
- Tailwind CSS 4 for styling with `tailwind-merge`, `tailwind-variants`, `tw-animate-css` (matching Pocket ID's exact stack)
- UI components via `bits-ui` (headless), `formsnap`, `sveltekit-superforms` + `zod` for forms
- Icons via `@lucide/svelte`, toasts via `svelte-sonner`, dark/light mode via `mode-watcher`
- Components in `$lib/components/`, utilities in `$lib/utils/`
- Star Citizen visual twist: dark blues, teals, metallic accents — but keep Pocket ID's structural patterns

### General

- No over-engineering — build only what's needed for the current phase
- Keep RSI scraping logic isolated in `internal/rsi/` so it can be updated if RSI changes their page structure
- All secrets (API keys, DB credentials) via environment variables, never hardcoded
- Use `.env` files for local development only — never commit them

## Key Domain Concepts

- **RSI Handle:** A user's Star Citizen username on robertsspaceindustries.com
- **Bio Verification:** The process of proving RSI account ownership by placing a token in the user's public RSI profile bio
- **Verification Token:** A unique string (format: `scid:<random>`) that the user places in their RSI bio
- **OIDC Client:** A fan site application registered to use SCID for authentication
- **Custom Claims:** Key/value pairs stored on Pocket ID users (e.g., `rsi_handle`, `rsi_verified_at`)
- **Verified Group:** The Pocket ID group `verified` — users are added to this group upon successful RSI bio verification. OIDC clients can require this group to restrict access to verified accounts only.
- **Org SID:** The Spectrum Identification code for an RSI organization (e.g., `SPAWO`, `LUG`), extracted from `/orgs/<SID>` URLs
- **Org Group:** A namespaced Pocket ID group mapped from an RSI org SID (format: `rsi:<SID>`, e.g., `rsi:SPAWO`)

## RSI Profile Scraping

- The public profile URL is: `https://robertsspaceindustries.com/en/citizens/<handle>`
- The organizations page is: `https://robertsspaceindustries.com/en/citizens/<handle>/organizations`
- Be respectful: rate-limit requests, cache responses, use a proper User-Agent header
- The scraping logic must be isolated so it can be updated when RSI changes their HTML
- Never store or process any data beyond what's needed for verification (handle, bio text, citizen record number, enlistment date, org SIDs)
- Extract citizen record number and enlistment date from the profile page during verification
- Extract org SIDs from the organizations page for group mapping (future phase)
- Ship/fleet data is not available on public RSI profile pages

## Security Guidelines

- Rate-limit all public endpoints (verification attempts, client registration)
- Verification tokens must be cryptographically random and time-limited
- One RSI handle maps to exactly one SCID account (enforce uniqueness)
- Validate all redirect URIs strictly when registering OIDC clients
- Never log sensitive data (tokens, secrets, full profile content)
- All external HTTP calls must use timeouts
- Sanitize any data extracted from RSI pages before storage
- **Org group name collision prevention:** Always prefix RSI org SIDs with `rsi:` when creating Pocket ID groups (e.g., `rsi:ADMIN` not `admin`). An RSI org with SID `ADMIN` must never map to Pocket ID's built-in `admin` group.

## Testing

- Write unit tests for verification logic and token generation
- Integration tests for the API endpoints
- Mock RSI profile responses in tests — never hit real RSI in CI
- Test OIDC flows end-to-end with a test Pocket ID instance
