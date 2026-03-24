---
layout: default
title: My SCID — Documentation
---

# My SCID

**Unofficial Star Citizen Identity Provider**

My SCID lets fan sites offer "Login with SCID" — a standard OpenID Connect login backed by verified RSI account ownership. No RSI passwords are ever shared.

The current implementation combines:

- A Pocket ID OIDC provider
- A Go companion service for RSI verification, org syncing, and app administration
- A SvelteKit frontend for users, app owners, and admins

---

## How it works

1. A user creates a SCID account through the registration flow.
2. They verify ownership of their RSI handle by placing a short token in their public RSI bio.
3. SCID scrapes the public RSI profile and organizations pages, then stores the result on the Pocket ID account.
4. Fan sites integrate SCID as a standard OIDC provider — any OIDC client library works.
5. When a user logs in, the fan site receives the user's verified RSI handle and org memberships as standard JWT claims.

## Current user flows

- `/register` - create a SCID account
- `/verify` - link or re-link an RSI handle
- `/apps` - manage registered OIDC clients
- `/apps/[id]` - view and edit one client
- `/discover` - public app directory
- `/admin/apps` - approve or reject app registrations

## What the companion service stores

- RSI handle
- Verification timestamp
- RSI enlistment date
- UEE Citizen Record number when available
- RSI org membership as namespaced Pocket ID groups
- Cached org metadata and org logos

---

## Guides

- [Integrating Login with SCID](integration.md) — OIDC client setup for fan site operators
- [OIDC Claims Reference](claims.md) — all claims returned in tokens and userinfo

---

## Quick start (fan site operators)

1. Log in to SCID with a verified account.
2. Go to [My Apps](https://auth.scid.my/apps) and register a new application.
3. Fill in your application name, launch URL, redirect URIs, and any optional logout URIs.
4. Choose whether the client is public, PKCE-only, verified-only, and whether it should be listed in the public directory.
5. Submit for approval if the instance requires it, then use the issued Client ID and Client Secret with any OIDC library.

```
Discovery URL: https://auth.scid.my/.well-known/openid-configuration
```

## OIDC behavior

- The authorization, token, userinfo, and JWKS endpoints are published through OIDC discovery
- Verified users are marked by membership in the `verified` Pocket ID group
- RSI orgs are synced to `rsi:<SID>` groups so fan sites can authorize by org membership
- Public directory entries only show approved apps that opted into listing and provided a launch URL

---

> This is an unofficial fan project — not affiliated with Cloud Imperium Games.
