# SCID Python Flask Example (Confidential Client)

A minimal Flask web app demonstrating OIDC authentication with SCID using the **confidential client flow** (client secret, server-side token exchange).

## What it shows

- Server-side authorization code flow with a client secret
- Session-based login / logout
- Displaying OIDC claims (including SCID custom claims like `rsi_handle`)

## Setup

1. **Register your app on SCID**
   - Go to `https://<your-scid>/apps` and register a new application.
   - Leave **Client Type** as `Confidential` (default).
   - Add `http://localhost:5000/callback` as a redirect URI.
   - Note your **Client ID** and **Client Secret**.

2. **Install dependencies**

   ```bash
   python3 -m venv .venv
   source .venv/bin/activate
   pip install -r requirements.txt
   ```

3. **Create a `.env` file** (never commit this):

   ```dotenv
   SCID_BASE=https://<your-scid>
   CLIENT_ID=<your-client-id>
   CLIENT_SECRET=<your-client-secret>
   REDIRECT_URI=http://localhost:5000/callback
   SECRET_KEY=change-me-to-a-random-string
   ```

4. **Run**

   ```bash
   flask run
   ```

   Open `http://localhost:5000` in your browser.

## Files

| File | Purpose |
|------|---------|
| `app.py` | Flask application |
| `requirements.txt` | Python dependencies |
| `templates/index.html` | Home / login page |
| `templates/profile.html` | Post-login profile page |

## Source

Part of the [SCID project](https://github.com/example/scid/).
