# SCID — Star Citizen Identity Provider

## Project Goal

Provide an OAuth2/OIDC identity provider for Star Citizen fan websites. Since robertsspaceindustries.com (RSI) offers no third-party authentication, SCID verifies a user's RSI identity by having them place a unique token in their publicly visible RSI profile bio, then confirms ownership by scraping/fetching that bio page. Once verified, SCID acts as a standard OIDC provider that any fan site can integrate with.

Fan site operators can request new OIDC client applications through SCID so they can embed "Login with SCID" into their own projects.

---

## Provider Comparison

| Criteria | Pocket ID | Keycloak | Authentik | Authelia |
|---|---|---|---|---|
| **Language / Stack** | Go + SvelteKit | Java (Quarkus) | Python (Django) | Go |
| **OIDC Provider** | Yes | Yes | Yes | Yes |
| **Complexity** | Very low | Very high | Medium-high | Low |
| **Custom Auth Flows** | No (passkey-only) | Yes (SPI, custom authenticators) | Yes (Flows + Stages) | Limited |
| **REST API** | Yes (basic) | Full Admin REST API | Full API | Minimal API |
| **Self-service App Registration** | No (admin only) | Via Dynamic Client Registration | Via API/Blueprints | No |
| **Custom User Attributes** | Custom claims (key/value) | Full user profile + attributes | User properties + attributes | No |
| **User Self-Registration** | Configurable (disabled/token/open) | Yes | Yes | No (file/LDAP backend) |
| **Theming / Branding** | Accent color, app name | Full theme engine (FreeMarker) | Full branding | Limited |
| **Extensibility** | Fork & modify (Go) | SPI plugins (Java) | Policies, Stages, signals | Configuration only |
| **Resource Footprint** | ~50 MB RAM | ~500 MB+ RAM | ~300 MB+ RAM | ~30 MB RAM |
| **License** | BSD-2-Clause | Apache 2.0 | OSS + Enterprise | Apache 2.0 |
| **Community** | 7.2k stars, active | Massive, CNCF | Large, growing | Large |

### Recommendation: Start with Pocket ID

**Why Pocket ID is a good starting point:**

1. **Simplicity** — Pocket ID is deliberately minimal. For a fan community project, operational simplicity matters. It's a single Go binary with a SvelteKit frontend.
2. **Lightweight** — ~50 MB RAM vs 500 MB+ for Keycloak. Easy to self-host cheaply.
3. **OIDC compliant** — Provides standard OIDC endpoints that any fan site client library can consume.
4. **Custom claims** — Supports custom key/value claims per user, which we can use for RSI handle, RSI verification status, org membership, etc.
5. **User groups** — Can assign users to groups, useful for role-based access (verified, unverified, admin).
6. **API keys** — Has admin API for programmatic management.
7. **Go codebase** — Easy to extend with custom verification logic.

**What we need to build on top:**

1. RSI bio token verification flow (custom service)
2. Self-service OIDC client registration (Pocket ID only allows admin-created clients)
3. RSI handle storage & verification status as custom claims
4. A small companion service or modifications to Pocket ID

**When to consider alternatives:**

- If the project outgrows Pocket ID's capabilities (complex RBAC, federation, multi-tenant orgs), **Keycloak** would be the logical upgrade path — it has full custom authenticator SPI support and could have the RSI verification built as a custom authenticator plugin.
- **Authentik** is a middle ground — more capable than Pocket ID but less complex than Keycloak.
- **Authelia** is unsuitable — it is a reverse-proxy auth companion, not a standalone OIDC provider for third-party apps.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                      SCID System                        │
│                                                         │
│  ┌─────────────┐    ┌──────────────────────────────┐    │
│  │  Pocket ID   │◄──│   SCID Companion Service     │    │
│  │  (OIDC IdP)  │   │                              │    │
│  │              │   │  • RSI Bio Verification       │    │
│  │  • Login     │   │  • Token Generation           │    │
│  │  • Consent   │   │  • RSI Profile Scraping       │    │
│  │  • Tokens    │   │  • Client App Requests        │    │
│  │  • User Mgmt │   │  • Verification Status Sync   │    │
│  └──────┬───────┘   └──────────────────────────────┘    │
│         │                                               │
└─────────┼───────────────────────────────────────────────┘
          │ OIDC (Authorization Code Flow)
          │
    ┌─────┴──────────────────────────────┐
    │        Fan Site Clients            │
    │                                    │
    │  • SC Trade Tools                  │
    │  • Org Management Sites            │
    │  • Fleet Trackers                  │
    │  • Community Forums                │
    └────────────────────────────────────┘
```

---

## RSI Bio Verification Flow

This is the core novel feature. Since RSI has no OAuth/API for identity, we verify ownership of an RSI account by having the user prove they control it.

### Flow Steps

1. **User registers** on SCID (via Pocket ID passkey flow or email one-time access)
2. **User initiates RSI verification** in their SCID profile
3. **User enters their RSI handle** (e.g., `CaptainKirk`)
4. **SCID generates a unique verification token** (e.g., `scid:a7b3c9d2e1f0`)
5. **User is instructed** to add this token to their RSI profile bio at `https://robertsspaceindustries.com/citizens/<handle>`
6. **User clicks "Verify"** when they've added the token
7. **SCID fetches the public RSI profile page** for the given handle
8. **SCID checks** if the token exists in the bio text
9. **If found:**
   - Add the user to the `verified` Pocket ID group
   - Store the RSI handle as a custom claim (`rsi_handle`)
   - Store verification timestamp (`rsi_verified_at`)
   - Store citizen record number (`rsi_citizen_record`) and enlistment date (`rsi_enlisted`) from the profile page
   - User can optionally remove the token from their bio
10. **If not found:**
    - Show error, allow retry
    - Token remains valid for a configurable period (e.g., 24 hours)

### Security Considerations

- **Rate limiting** on verification attempts to prevent scraping abuse
- **Token expiry** — verification tokens expire after a configurable window
- **Handle uniqueness** — one RSI handle can only be linked to one SCID account
- **Re-verification** — periodic or on-demand re-verification to detect handle changes
- **Respectful scraping** — cache RSI profile fetches, respect robots.txt, add reasonable delays, use proper User-Agent
- **Org group name collisions** — RSI org SIDs are user-created and could collide with Pocket ID built-in groups (e.g., an org literally named "admin" with SID `ADMIN`). All RSI org groups must be **namespaced** with a prefix (e.g., `rsi:SPAWO`, `rsi:LUG`) to prevent privilege escalation. Never map raw org SIDs directly to Pocket ID group names.

### RSI Profile Structure

The public profile page is at:
```
https://robertsspaceindustries.com/en/citizens/<handle>
```

**Available data on the profile page (no auth required):**
- Handle (display name)
- UEE Citizen Record number
- Profile image
- Enlistment date
- Fluency (languages)
- Bio (free-text field — this is where the verification token goes)

**Organization data** is on a separate page:
```
https://robertsspaceindustries.com/en/citizens/<handle>/organizations
```

Each org entry shows:
- Organization name
- **SID** (Spectrum Identification) — the unique org identifier from the URL `/orgs/<SID>`
- Membership type: **Main** (one per user) or **Affiliation** (zero or more)
- Rank within the org
- Member count

**Example** (handle `CyFreeze`):
| Org Name | SID | Type | Rank |
|---|---|---|---|
| Space Wombats | SPAWO | Main | Oldest |
| Linux Users Group | LUG | Affiliation | Penguin |
| Æon | AEON | Affiliation | President |

The org SID is a short uppercase string and serves as a stable, unique identifier for groups.

---

## Self-Service Client Application Registration

Fan site operators need to register their site as an OIDC client to enable "Login with SCID."

### Flow

1. **Authenticated user** navigates to "Register Application" in SCID
2. **User fills out application form:**
   - Application name
   - Application URL (homepage)
   - Redirect URIs (OAuth callback URLs)
   - Description / purpose
3. **Application is created** with status `pending` (or auto-approved, configurable)
4. **Admin reviews** and approves (if manual approval is enabled)
5. **User receives:**
   - Client ID
   - Client Secret
   - OIDC discovery URL
   - Integration instructions
6. **User can manage** their applications (rotate secrets, update redirect URIs, view usage)

### Implementation Approach

Since Pocket ID only supports admin-created OIDC clients, we have two options:

**Option A: SCID Companion Service with Pocket ID Admin API**
- Build a service that uses Pocket ID's admin API key to create OIDC clients on behalf of users
- Manage the approval workflow in the companion service
- Pros: No Pocket ID fork needed
- Cons: Depends on Pocket ID API stability

**Option B: Fork Pocket ID**
- Add self-service client registration directly to Pocket ID
- Add RSI verification as a built-in feature
- Pros: Single deployment, tighter integration
- Cons: Maintenance burden for upstream updates

**Recommended: Option A initially**, moving to Option B if the project matures and needs tighter integration.

---

## Custom Claims (OIDC Token Content)

When a fan site authenticates a user via SCID, the ID token / userinfo endpoint will include:

| Claim | Type | Description |
|---|---|---|
| `sub` | string | Unique SCID user ID |
| `preferred_username` | string | SCID username |
| `email` | string | User's email |
| `rsi_handle` | string | Verified RSI handle (empty if unverified) |
| `rsi_verified_at` | string (ISO 8601) | When the RSI handle was verified |
| `rsi_citizen_record` | string | UEE Citizen Record number (e.g., `#40746`) |
| `rsi_enlisted` | string (ISO 8601) | Enlistment date from RSI profile |
| `groups` | string[] | Pocket ID groups — always includes `verified` for verified users, future: namespaced RSI orgs (e.g., `rsi:SPAWO`) |

### Group-Based Access Control

Verification status is represented as membership in the **`verified`** Pocket ID group rather than a custom claim boolean. This is the preferred approach because:

- OIDC clients can restrict access to verified users by requiring the `verified` group — no custom claim parsing needed
- Pocket ID already exposes `groups` in the standard OIDC `groups` claim
- Fan site operators can simply check `"verified" in token.groups` in their OIDC client config
- If a user fails re-verification, removing them from the `verified` group immediately revokes access across all fan sites

The `rsi_citizen_record` and `rsi_enlisted` claims let fan sites gauge account age — useful for trust scoring or gating features behind account maturity.

These claims allow fan sites to:
- Know the user's verified RSI identity
- Gate access to verified-only users via the `verified` group
- Assess account age via enlistment date and citizen record number
- Show org-specific content (future: via `rsi:*` groups)

**Note:** Ship/fleet data is not available on RSI's public profile pages and cannot be scraped.

### Org-to-Group Mapping (Future)

RSI org SIDs are synced as Pocket ID groups with a `rsi:` prefix to avoid collisions with built-in groups like `admin`. The companion service:

1. Fetches the user's `/citizens/<handle>/organizations` page
2. Extracts all org SIDs (no distinction between main org and affiliations — all treated equally)
3. Creates Pocket ID groups for any new `rsi:<SID>` values
4. Assigns the user to the appropriate groups
5. Removes the user from `rsi:*` groups they are no longer a member of

---

## Technical Components

### 1. Pocket ID (Upstream)
- Deployed as-is via Docker
- Handles all OIDC protocol concerns
- Manages user accounts, passkey auth, sessions
- Stores custom claims per user

### 2. SCID Companion Service
- **Language:** Go (to match Pocket ID ecosystem)
- **Framework:** Standard library or lightweight framework (e.g., Chi, Echo)
- **Database:** Shares Pocket ID's database or uses its own SQLite/PostgreSQL
- **Responsibilities:**
  - RSI bio verification engine
  - Self-service client app registration & management
  - Admin approval workflow for new apps
  - Periodic re-verification jobs
  - User-facing UI for verification and app management

### 3. Frontend
- **Framework:** SvelteKit (Svelte 5) — matching Pocket ID
- **Build:** Vite, pnpm
- **Styling:** Tailwind CSS 4 with `tailwind-merge`, `tailwind-variants`, `tw-animate-css` — matching Pocket ID's exact stack
- **UI Components:** `bits-ui` (headless Svelte components, same as Pocket ID), `formsnap`, `sveltekit-superforms` + `zod` for form validation
- **Icons:** `@lucide/svelte`
- **Toasts:** `svelte-sonner`
- **Dark/Light mode:** `mode-watcher`
- **i18n:** `@inlang/paraglide-js` (optional, for future localization)
- **Star Citizen visual twist:** Custom Tailwind theme with SC-inspired color palette (dark blues, teals, metallic accents). Use RSI-style typography and spacing. The goal is recognizably "Star Citizen" while remaining visually compatible with Pocket ID's component patterns.
- **Pages:**
  - RSI verification wizard
  - Application registration & management
  - Admin dashboard (approve/reject apps, view verifications)

### 4. Infrastructure
- Docker Compose deployment
- Reverse proxy (Caddy recommended, for automatic HTTPS)
- PostgreSQL or SQLite for persistence
- Optional: Redis for rate limiting / job queues

---

## Project Phases

### Phase 1: Foundation
- [x] Set up Pocket ID instance with Docker Compose
- [ ] Configure Pocket ID (branding, user signup, email)
- [x] Build SCID companion service skeleton (Go)
- [x] Implement RSI bio verification flow
- [x] Scrape and store citizen record number and enlistment date from profile page
- [x] Store verification results as Pocket ID custom claims via API (`rsi_handle`, `rsi_verified_at`, `rsi_citizen_record`, `rsi_enlisted`)
- [x] RSI avatar scraped and uploaded to Pocket ID profile picture (`PUT /api/users/{id}/profile-picture`)
- [x] Enlisted date stored as ISO 8601 (`2023-04-16`); `n/a` citizen record omitted
- [x] Basic UI for verification wizard (`/verify`)
- [x] Home page shows profile card when verified (avatar, handle, claims, Refresh button)
- [x] `POST /api/verify/refresh` — refreshes RSI profile data and avatar without re-verification
- [x] `GET /api/verify/status` returns full claim set (`user_id`, `username`, `handle`, `verified_at`, `enlisted`, `citizen_record`, `avatar_url`)
- [x] Home page hero + feature cards shown when not verified; header "Verify Identity" button removed

---

### Profile Card (Home Page — `/`)

The home page (`/`) is the profile hub for logged-in users. Implemented.

**Verified users see:**
- Avatar (served from `{POCKET_ID_URL}/api/users/{id}/profile-picture.png` — RSI avatar uploaded on verification/refresh)
- RSI handle + green "Verified" badge
- Info grid: verified date, enlistment date, citizen record number
- Next org re-sync time bar (human relative, e.g. "in 3 days") — persists correctly after refresh
- "Refresh RSI info" button — calls `POST /api/verify/refresh`, shows spinner + toast; response now always includes `next_sync_at`
- "Change RSI handle" link — inline amber warning → navigates to `/verify` wizard with pre-warning banner; existing verification is replaced on success
- "Delete account" link — inline red warning + confirm; same pattern as change-handle

**Logged-in but unverified users see:**
- Hero section with SCID branding
- "Create Your RSI Identity" call-to-action button
- Feature explanation cards

**Unauthenticated users see:**
- Same hero + feature cards (no user data shown)

### Phase 2: Client App Registration
- [x] `POST /api/apps` — create OIDC client via Pocket ID admin API (name, redirect URIs, homepage)
- [x] `GET /api/apps` — list requester's registered apps (filter by `owner_id` claim)
- [x] `GET /api/apps/{id}` — get single app + secret
- [x] `DELETE /api/apps/{id}` — delete app
- [x] `POST /api/apps/{id}/secret` — rotate client secret
- [x] Store owner metadata in companion DB (pocket_id client ID → SCID user ID)
- [x] `/apps` SvelteKit page — registration form (with logo upload) + list of user's apps
- [x] `/apps/{id}` SvelteKit page — app detail, secret display (once), rotate, delete, logo upload, edit redirect URIs
- [x] Only verified users can register OIDC client applications
- [x] Admin approval workflow — `APP_REQUIRE_APPROVAL` env var (default `true`); new apps are set to `pending` and restricted to `scid:pending` sentinel group (no real members → no logins until approved); admins approve / reject via `/admin/apps`
- [x] `GET /api/admin/apps` — list all apps (filterable by status); admin only
- [x] `POST /api/admin/apps/{id}/approve` — approve app (removes pending restriction, applies verified/unrestricted groups)
- [x] `POST /api/admin/apps/{id}/reject` — reject app with reason (re-applies pending restriction)
- [x] `GET /api/verify/status` includes `admin: true` for users in the Pocket ID `admin` group
- [x] `/admin/apps` SvelteKit page — filter tabs (pending / approved / rejected / all), approve/reject buttons, reject with reason modal
- [x] RSI org membership scraping (`/citizens/<handle>/organizations`) + caching in SQLite
- [x] `rsi:<SID>`-prefixed Pocket ID groups synced from org SIDs on verification and refresh
- [x] App directory listing — `listed` flag on `app_registrations`; approved apps with a launch URL can opt into the public directory; owners toggle via "List in App Directory" checkbox in the app edit form; requires `approved` status
- [x] `GET /api/apps/directory` — public endpoint returning minimal `{id, name, launch_url, has_logo, verified_only}` for all listed+approved apps; no auth required
- [x] `/discover` SvelteKit page — app cards with logo, name, verified-only badge, launch link; linked from header nav for authenticated users

### Phase 3: Polish & Hardening
- [x] Rate limiting on verification and API endpoints — Cloudflare Turnstile on signup, moved to dedicated `/register` page
- [x] Periodic re-verification background job — weekly goroutine re-fetches profile + orgs for all `verified` group members (`internal/api/reverify.go`)
- [x] Cloudflare Turnstile captcha on signup — server-side validation in `register.go`, widget on `/register` page (themed, styled); skipped when `TURNSTILE_SECRET_KEY` is unset (dev mode)
- [x] Session persistence — access token moved from `sessionStorage` to `localStorage` so sessions survive tab closes and new tabs; PKCE state stays in sessionStorage
- [x] OIDC callback full-page reload — `window.location.replace()` ensures layout re-mounts after login so header links appear immediately
- [x] Manage Account link navigates same-tab — avoids `noopener` session isolation issue
- [x] `/register` dedicated page — Turnstile widget removed from `/verify`; "Create SCID account" links to `/register`, which forwards to Pocket ID `/signup` with validated token
- [x] CitizenRecord stored as integer-only (strip leading `#`) in scraper; frontend re-adds `#` for display
- [x] Group members endpoint fixed — uses `GET /api/users?groupId=` instead of non-existent `/api/user-groups/{id}/members`
- [x] Impressum page (`/impressum`) with obfuscated contact info, linked from footer
- [x] Header branding — "My SCID" as site name, "Unofficial Star Citizen Identity Provider" as tagline
- [x] Profile card shows next org re-sync time (human relative, e.g. "in 3 days"); `next_sync_at` now correctly returned by `POST /api/verify/refresh` (was missing, causing it to vanish from UI after refresh)
- [x] Sign out button added to header nav — clears `localStorage` token and redirects to `/`
- [x] Audit logging
- [x] Monitoring & health checks
- [x] User documentation — GitHub Pages docs site (`docs/`) with integration guide, OIDC claims reference, setup instructions

### Phase 4: Future Enhancements
- [x] Community portal / directory of fan sites using SCID — `/discover` page + `GET /api/apps/directory` (see Phase 2)
- [x] Re-verification / handle change — "Change RSI handle" link on profile card triggers inline amber warning → continues to `/verify` wizard; wizard shows a change-handle banner when accessed this way; backend (`handleVerifyStart`) already supports overwriting an existing verified handle

---

## RSI Fan Content & Licensing

RSI's Terms of Service (Section XIII.D — "Personal and Fansite Use") permit fan sites to use certain RSI-designated images, graphics, artwork ("RSI Fansite Content") and trademarks/logos ("RSI Marks") under these conditions:

- Must clearly state: **"This is an unofficial Fansite"**
- May **not** charge subscription/access fees or generate advertising/sponsor revenue (except gameplay streaming)
- RSI can amend or revoke permissions at any time
- Cannot create or sell merchandise based on RSI content
- Must include RSI's trademark/copyright notices when displaying RSI Content

**Fan Site Kit status:** RSI's dedicated fan site kit page (`/community/fan-kit`) no longer exists — it redirects to the Community Hub. The support article URL (`support.robertsspaceindustries.com/.../Star-Citizen-Fan-Content-Policy`) returns 404. The fan content policy is currently only available within the TOS.

**Practical approach for SCID:**
- Use the Star Citizen *aesthetic* (colors, typography, visual language) without embedding RSI copyrighted assets
- If RSI-designated fansite assets become available, they can be incorporated with proper attribution
- Include the "unofficial fansite" disclaimer in the footer
- Avoid using the Star Citizen logo directly — design a distinct SCID logo/mark

---

## Development Setup

```
scid/
├── docker-compose.yml          # Pocket ID + SCID service + DB + Caddy
├── .env.example                # Environment configuration template
├── companion/                  # SCID Companion Service (Go)
│   ├── cmd/
│   │   └── scid/
│   │       └── main.go
│   ├── internal/
│   │   ├── rsi/                # RSI profile fetching & verification
│   │   ├── clients/            # OIDC client registration management
│   │   ├── api/                # HTTP handlers
│   │   └── store/              # Database access
│   ├── go.mod
│   └── go.sum
├── frontend/                   # SvelteKit frontend for SCID-specific UI
│   ├── src/
│   │   ├── routes/
│   │   │   ├── verify/         # RSI verification wizard
│   │   │   ├── apps/           # Client app registration & management
│   │   │   └── admin/          # Admin dashboard
│   │   └── lib/
│   ├── package.json
│   └── svelte.config.js
├── PLAN.md                     # This file
├── .github/
│   └── copilot-instructions.md # Copilot coding guidelines
├── README.md
└── LICENSE
```
