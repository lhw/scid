---
layout: default
title: Integrating Login with SCID
---

# Integrating Login with SCID

My SCID is a standard [OpenID Connect (OIDC)](https://openid.net/developers/how-connect-works/) provider. Any OIDC client library works — no custom code needed.

## Prerequisites

- A verified SCID account (you need to link your RSI handle first)
- A registered application in [My Apps](https://id.scid.my/apps)

## Registering an application

In **My Apps**, the current registration flow lets you configure:

- Application name
- Launch URL for the public app directory
- One or more redirect URIs
- Optional logout redirect URIs
- Public client mode for SPAs and mobile apps
- PKCE requirement
- Verified-only access
- Optional app listing in the public directory
- Optional app logo upload

New apps may require admin approval depending on the instance configuration.

## OIDC Discovery

The auto-discovery document is available at:

```
https://id.scid.my/.well-known/openid-configuration
```

All OIDC endpoints (authorization, token, userinfo, JWKS) are listed there. Most libraries accept a discovery URL and configure themselves automatically.

## Flow: Authorization Code + PKCE

My SCID supports **Authorization Code Flow with PKCE** (recommended for all clients) and the standard Authorization Code Flow with a client secret.

### Step 1 — Redirect the user to My SCID

```
GET https://id.scid.my/authorize
  ?response_type=code
  &client_id=YOUR_CLIENT_ID
  &redirect_uri=https://yoursite.example/callback
  &scope=openid profile email
  &code_challenge=BASE64URL(SHA256(code_verifier))
  &code_challenge_method=S256
  &state=RANDOM_CSRF_TOKEN
```

### Step 2 — Exchange the code for tokens

```
POST https://id.scid.my/api/oidc/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=AUTH_CODE
&redirect_uri=https://yoursite.example/callback
&client_id=YOUR_CLIENT_ID
&code_verifier=YOUR_CODE_VERIFIER
```

Response:
```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "id_token": "..."
}
```

### Step 3 — Fetch the userinfo

```
GET https://id.scid.my/api/oidc/userinfo
Authorization: Bearer ACCESS_TOKEN
```

See [OIDC Claims Reference](claims.md) for all returned fields.

## Restricting access to verified users

Set the OIDC client's required groups to `verified` in the My SCID admin panel or check the `groups` claim in your application. Users not in the `verified` group will be denied access before the consent screen when the client is configured that way.

In your own code you can also check:

```json
"groups": ["verified", "rsi:SPAWO"]
```

If `"verified"` is present in the `groups` claim, the user has a confirmed RSI identity.

## Claim reference

See [OIDC Claims Reference](claims.md) for the current claim set and the separate SCID status fields returned by `/api/verify/status`.

## Example: Python (authlib)

```python
from authlib.integrations.flask_client import OAuth

oauth = OAuth(app)
scid = oauth.register(
    name='scid',
    server_metadata_url='https://id.scid.my/.well-known/openid-configuration',
    client_id='YOUR_CLIENT_ID',
    client_secret='YOUR_CLIENT_SECRET',
    client_kwargs={'scope': 'openid profile email'},
)
```

## Example: Node.js (openid-client)

```js
import { Issuer } from 'openid-client';

const scidIssuer = await Issuer.discover('https://id.scid.my');
const client = new scidIssuer.Client({
  client_id: 'YOUR_CLIENT_ID',
  client_secret: 'YOUR_CLIENT_SECRET',
  redirect_uris: ['https://yoursite.example/callback'],
  response_types: ['code'],
});
```

---

[← Back to index](index.md)
