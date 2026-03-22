# My SCID

Unofficial Star Citizen identity provider for fan sites.

My SCID uses Pocket ID as the OIDC backend and a Go companion service to verify RSI account ownership, sync RSI org membership, and manage self-service OIDC client registration.

## What it does today

- Verifies RSI identity by asking the user to place a short token in their public RSI bio
- Stores RSI claims on the Pocket ID account, including handle, verification time, enlistment date, and citizen record number
- Syncs RSI org membership into namespaced Pocket ID groups such as `rsi:SPAWO`
- Lets verified users register, update, rotate secrets for, and delete OIDC clients
- Supports app approval, verified-only app access, app logos, and an optional public app directory
- Exposes a small SCID status API for the frontend and companion workflows

## Repository layout

- `companion/` - Go companion service, API handlers, RSI scraping, and Pocket ID integration
- `frontend/` - SvelteKit frontend for verification, account management, app registration, admin review, and the public app directory
- `docs/` - GitHub Pages documentation site
- `docker-compose.yml` - Local stack for Pocket ID, the companion service, the database, and Caddy
- `docker-compose.prod.yml` - Production stack with PostgreSQL for the companion service

## Stack

- Go 1.25 for the companion service
- Svelte 5 + SvelteKit + Vite 7 for the frontend
- Tailwind CSS 4, bits-ui, formsnap, sveltekit-superforms, zod, @lucide/svelte, svelte-sonner, and mode-watcher
- SQLite for development, PostgreSQL for production
- Caddy for reverse proxying and HTTPS

## Documentation

- [Documentation home](docs/index.md)
- [Integration guide](docs/integration.md)
- [OIDC claims reference](docs/claims.md)

## Current implementation highlights

- RSI profile scraping uses the `/en/citizens/<handle>` and `/en/citizens/<handle>/organizations` pages
- Verification and refresh flows update custom claims plus the user profile picture in Pocket ID
- Org groups are always prefixed with `rsi:` to avoid collisions with built-in Pocket ID groups
- App registration supports redirect URIs, logout URIs, public clients, PKCE, verified-only clients, listing, and logo uploads
- The frontend includes `/verify`, `/register`, `/apps`, `/apps/[id]`, `/discover`, and `/admin/apps`

## Local development

See the repository docs, `docker-compose.yml`, and `docker-compose.prod.yml` for the current development and deployment layout.

## Disclaimer

This is an unofficial fan project and is not affiliated with Cloud Imperium Games.