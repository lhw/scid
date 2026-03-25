"""
SCID Python Flask example — confidential client (authorization code flow).

Configure via environment variables or a .env file:
  SCID_BASE      e.g. https://scid.example.com   (no trailing slash)
  CLIENT_ID      OIDC client_id from SCID
  CLIENT_SECRET  OIDC client_secret from SCID
  REDIRECT_URI   must match what you registered, e.g. http://localhost:5000/callback
  SECRET_KEY     any random string used for Flask session signing
"""

import hashlib
import os
import secrets
import urllib.parse

import requests
from dotenv import load_dotenv
from flask import Flask, abort, redirect, render_template, request, session, url_for

load_dotenv()

app = Flask(__name__)
app.secret_key = os.environ["SECRET_KEY"]

SCID_BASE     = os.environ["SCID_BASE"].rstrip("/")
CLIENT_ID     = os.environ["CLIENT_ID"]
CLIENT_SECRET = os.environ["CLIENT_SECRET"]
REDIRECT_URI  = os.environ["REDIRECT_URI"]

AUTH_ENDPOINT  = f"{SCID_BASE}/authorize"
TOKEN_ENDPOINT = f"{SCID_BASE}/api/oidc/token"
USERINFO_URL   = f"{SCID_BASE}/api/oidc/userinfo"


@app.route("/")
def index():
    user = session.get("user")
    return render_template("index.html", user=user)


@app.route("/login")
def login():
    state = secrets.token_urlsafe(16)
    session["oauth_state"] = state

    params = urllib.parse.urlencode({
        "response_type": "code",
        "client_id":     CLIENT_ID,
        "redirect_uri":  REDIRECT_URI,
        "scope":         "openid profile email",
        "state":         state,
    })
    return redirect(f"{AUTH_ENDPOINT}?{params}")


@app.route("/callback")
def callback():
    error = request.args.get("error_description") or request.args.get("error")
    if error:
        abort(400, description=f"Authorization failed: {error}")

    code  = request.args.get("code")
    state = request.args.get("state")

    if not code:
        abort(400, description="No authorization code returned.")

    saved_state = session.pop("oauth_state", None)
    if not saved_state or saved_state != state:
        abort(400, description="State mismatch — possible CSRF. Please try again.")

    # Exchange code for tokens using client_secret_basic auth.
    token_resp = requests.post(
        TOKEN_ENDPOINT,
        data={
            "grant_type":   "authorization_code",
            "code":         code,
            "redirect_uri": REDIRECT_URI,
        },
        auth=(CLIENT_ID, CLIENT_SECRET),
        timeout=10,
    )
    if not token_resp.ok:
        err = token_resp.json() if token_resp.content else {}
        abort(502, description=err.get("error_description") or f"Token exchange failed ({token_resp.status_code})")

    tokens = token_resp.json()
    access_token = tokens.get("access_token")

    # Fetch user info.
    userinfo_resp = requests.get(
        USERINFO_URL,
        headers={"Authorization": f"Bearer {access_token}"},
        timeout=10,
    )
    if not userinfo_resp.ok:
        abort(502, description=f"Userinfo fetch failed ({userinfo_resp.status_code})")

    session["user"] = userinfo_resp.json()
    return redirect(url_for("profile"))


@app.route("/profile")
def profile():
    user = session.get("user")
    if not user:
        return redirect(url_for("login"))
    return render_template("profile.html", user=user)


@app.route("/logout")
def logout():
    session.clear()
    return redirect(url_for("index"))
