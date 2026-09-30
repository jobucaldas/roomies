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
JWT_ACCESS_TTL_HOURS=8         # session cookie and bearer lifetime (default 8h)
```

## Flow

1. Flutter `GET /api/auth/config` → `{ authkit: true, password: false }`
2. User clicks **Sign in with AuthKit** → `GET /api/auth/workos/authorize` sets HttpOnly `state` and PKCE verifier cookies and returns an `api.workos.com` URL
3. AuthKit returns to `/callback?code=...&state=...`
4. Flutter `POST /api/auth/workos/callback` with the code and state. The browser sends the cookies. The server checks state and sends the PKCE verifier to WorkOS.
5. Backend upserts the local user (`workos_user_id`) and sets an HttpOnly session cookie (`JWT_ACCESS_TTL_HOURS`, default 8h). The web client does not receive or store a bearer token. Native and API clients still receive a bearer token of the same lifetime.
6. Login, register, and the code exchange are limited per client IP.

Invitation acceptance compares the invite address to `users.email`, not the email claim inside the session token.
