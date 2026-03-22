---
layout: default
title: OIDC Claims Reference
---

# OIDC Claims Reference

These claims are returned in the **ID token** and on the **`/api/oidc/userinfo`** endpoint when the user grants the corresponding scopes.

SCID also exposes a separate status endpoint at **`/api/verify/status`** for the frontend. That endpoint includes SCID-specific fields such as `authenticated`, `verified`, `admin`, `pending_handle`, `pending_expires_at`, `next_sync_at`, and cached org metadata. Those fields are not OIDC claims.

## Standard claims (`openid profile email`)

| Claim | Type | Description |
|---|---|---|
| `sub` | string | Stable SCID user ID (UUID) |
| `preferred_username` | string | SCID username (chosen at registration) |
| `email` | string | User's email address |
| `email_verified` | bool | Whether the email address has been verified |

## RSI identity claims

These are populated after a user completes RSI bio verification. They are always included when the user is in the `verified` group.

| Claim | Type | Example | Description |
|---|---|---|---|
| `rsi_handle` | string | `"CaptainKirk"` | Verified RSI citizen handle |
| `rsi_verified_at` | string (ISO 8601) | `"2025-11-03T14:22:00Z"` | Timestamp of initial verification |
| `rsi_enlisted` | string (ISO 8601 date) | `"2013-04-16"` | RSI enlistment date from the public profile |
| `rsi_citizen_record` | optional(string) | `"40746"` | UEE Citizen Record number (numeric, not present on every profile) |

The `rsi_citizen_record` claim is omitted when RSI shows the value as `n/a` or does not provide one.

## Group claims

| Claim | Type | Description |
|---|---|---|
| `groups` | string[] | All Pocket ID groups the user belongs to |

### Well-known groups

| Group name | Meaning |
|---|---|
| `verified` | User has a confirmed RSI identity |
| `admin` | SCID site administrator |
| `rsi:<SID>` | Member of RSI org with Spectrum ID `<SID>` (e.g. `rsi:LUG`) |

### Notes on groups

- `verified` is the canonical signal for RSI identity verification
- `admin` is used for SCID administration in the companion service and UI
- `rsi:<SID>` groups are always namespaced to avoid collisions with built-in Pocket ID groups

### Using groups for access control

Most OIDC libraries let you require a group claim value. For example with nginx-based auth proxies:

```nginx
# Require verified users only
proxy_set_header X-SCID-Verified $jwt_claim_groups_verified;
```

Or check in application code:

```python
user_info = scid.parse_id_token(token)
if 'verified' not in user_info.get('groups', []):
    abort(403, "RSI identity not verified")
```

## Example userinfo response

```json
{
  "sub": "2fad53a2-de28-42c4-8ef1-efc4fbf899d8",
  "preferred_username": "example",
  "email": "user@example.com",
  "email_verified": false,
  "rsi_handle": "Example",
  "rsi_verified_at": "2025-11-03T14:22:00Z",
  "rsi_enlisted": "2012-10-18",
  "rsi_citizen_record": "40746",
  "groups": ["verified", "rsi:SPAWO", "rsi:LUG"]
}
```

## SCID status example

The status endpoint combines auth state, verification state, and cached profile data for the signed-in user:

```json
{
  "authenticated": true,
  "verified": true,
  "admin": false,
  "user_id": "2fad53a2-de28-42c4-8ef1-efc4fbf899d8",
  "username": "example",
  "handle": "CaptainKirk",
  "verified_at": "2025-11-03T14:22:00Z",
  "enlisted": "2012-10-18",
  "citizen_record": "40746",
  "pending_handle": "CaptainKirk",
  "pending_expires_at": "2025-11-04T14:22:00Z",
  "next_sync_at": "2025-11-10T14:22:00Z",
  "orgs": [
    {
      "sid": "SPAWO",
      "name": "Space Wombats",
      "rank_name": "Oldest",
      "is_main": true,
      "has_logo": true
    }
  ]
}
```

---

[← Back to index](index.md) · [Integration guide](integration.md)
