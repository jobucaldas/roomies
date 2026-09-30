# WorkOS AuthKit

Roomies uses WorkOS AuthKit for hosted sign-in when `WORKOS_API_KEY` and `WORKOS_CLIENT_ID` are set. Without those variables, the Flutter app keeps the local email/password form (used by CI). When they are set, `POST /api/auth/login` and `POST /api/auth/register` return 403, and AuthKit will not attach to an existing local password user.

## Staging (configured)

- Environment: Staging (`environment_01M3QCMJY7A19GT034QVKHKAJ4`)
- Client ID: `client_01M3QCMK75B35RPC8EAJA5GREP`
- Redirect URIs: `http://localhost/callback`, `http://localhost:58000/callback` (+ `127.0.0.1` variants)
- Initiate login URI: `http://localhost/`
- Logout return URIs: `http://localhost/`, `http://localhost:58000/`
- Auth methods: password + magic auth, signup allowed

## App env

```bash
WORKOS_API_KEY=sk_...          # from WorkOS Dashboard → API Keys (secret; not readable via MCP)
WORKOS_CLIENT_ID=client_01M3QCMK75B35RPC8EAJA5GREP
WORKOS_REDIRECT_URI=http://localhost/callback   # optional; defaults to {ROOMIES_PUBLIC_BASE_URL}/callback
JWT_ACCESS_TTL_HOURS=8         # Roomies access JWT lifetime (default 8h; was 72h)
```

## Flow

1. Flutter `GET /api/auth/config` → `{ authkit: true, password: false, access_token_ttl_s: … }`
2. Client generates a PKCE S256 verifier/challenge pair and stores the verifier in browser `sessionStorage` (SharedPreferences on native)
3. `GET /api/auth/workos/authorize?code_challenge=…&code_challenge_method=S256` → backend returns AuthKit URL plus an HMAC-signed `state` bound to that challenge
4. AuthKit returns to `/callback?code=…&state=…`
5. Flutter checks returned `state` against the pending value, then `POST /api/auth/workos/callback` with `{ code, state, code_verifier }`
6. Backend verifies state + PKCE, exchanges the code with WorkOS (including verifier), upserts the local user (`workos_user_id`), issues the Roomies JWT

Login, register, and AuthKit authorize/callback are rate-limited per IP (30 requests / 15 minutes).

## Session storage tradeoffs

Roomies issues its own bearer JWT after AuthKit (or password) login. The Flutter client stores that token in `SharedPreferences`, which on web is backed by `localStorage`.

**Why not httpOnly cookies today**

- The same Flutter codebase targets web, iOS, and Android. Native apps cannot use first-party httpOnly cookies the way a browser can; they need a bearer token (or platform secure storage + refresh).
- Switching web-only to cookies would fork auth transport (`Authorization` header vs cookie), CORS credential mode, and CSRF protections, while native still needed bearer auth.
- Preferable long-term: short-lived access JWT + refresh token in httpOnly cookie on web, and refresh in secure storage on native. That work is intentionally deferred.

**Current hardening**

- Access JWT TTL defaults to **8 hours** (`JWT_ACCESS_TTL_HOURS`), down from 72h, shrinking the XSS window if `localStorage` is stolen.
- AuthKit OAuth is bound to the initiating browser via PKCE + signed state.
- Auth endpoints are rate-limited.
- Invitation acceptance uses the persisted account email from the database, not the JWT `email` claim.
