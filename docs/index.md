---
layout: default
title: My SCID — Documentation
---

# My SCID

**Unofficial Star Citizen Identity Provider**

My SCID lets fan sites offer "Login with SCID" — a standard OpenID Connect login backed by verified RSI account ownership. No RSI passwords are ever shared.

---

## How it works

1. A user creates a SCID account (passkey-based, no password).
2. They verify ownership of their RSI handle by placing a short token in their public RSI bio.
3. Fan sites integrate SCID as a standard OIDC provider — any OIDC client library works.
4. When a user logs in, the fan site receives the user's verified RSI handle and org memberships as standard JWT claims.

---

## Guides

- [Integrating Login with SCID](integration.md) — OIDC client setup for fan site operators
- [OIDC Claims Reference](claims.md) — all claims returned in tokens and userinfo

---

## Quick start (fan site operators)

1. Log in to [My SCID](https://scid.my) with your verified SCID account.
2. Go to **My Apps** → **Register a new application**.
3. Fill in your application name, homepage URL, and one or more OAuth redirect URIs.
4. Submit for admin review; you'll receive an email once approved.
5. Use the provided **Client ID** and **Client Secret** with any OIDC library.

```
Discovery URL: https://id.scid.my/.well-known/openid-configuration
```

---

> This is an unofficial fan project — not affiliated with Cloud Imperium Games.
