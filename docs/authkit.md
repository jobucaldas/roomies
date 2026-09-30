# WorkOS AuthKit

Roomies uses WorkOS AuthKit for hosted sign-in when `WORKOS_API_KEY` and `WORKOS_CLIENT_ID` are set. Without those variables, the app keeps the local email/password form. When they are set, `POST /api/auth/login` and `POST /api/auth/register` return 403.

## App env

```bash
WORKOS_API_KEY=sk_...          # from WorkOS Dashboard → API Keys
WORKOS_CLIENT_ID=client_...
WORKOS_REDIRECT_URI=http://localhost/callback   # optional; defaults to {ROOMIES_PUBLIC_BASE_URL}/callback
JWT_ACCESS_TTL_HOURS=8         # session cookie and bearer lifetime (default 8h)
```

Configure matching redirect, initiate-login, and logout-return URIs in the WorkOS Dashboard for your environment.

## Flow

1. Client `GET /api/auth/config` → `{ authkit: true, password: false }`
2. User clicks **Sign in with AuthKit** → `GET /api/auth/workos/authorize` sets HttpOnly `state` and PKCE verifier cookies and returns an `api.workos.com` URL
3. AuthKit returns to `/callback?code=...&state=...`
4. Client `POST /api/auth/workos/callback` with the code and state. The browser sends the cookies. The server checks state and sends the PKCE verifier to WorkOS.
5. Backend upserts the local user (`workos_user_id`) and sets an HttpOnly session cookie (`JWT_ACCESS_TTL_HOURS`, default 8h). The web client does not receive or store a bearer token. Native and API clients still receive a bearer token of the same lifetime.
6. Login, register, and the code exchange are limited per client IP.

Invitation acceptance matches the invite address to `users.email`.
