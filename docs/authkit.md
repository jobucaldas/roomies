# WorkOS AuthKit

Roomies uses WorkOS AuthKit for hosted sign-in when `WORKOS_API_KEY` and `WORKOS_CLIENT_ID` are set. Without those variables, the Flutter app keeps the local email/password form (used by CI).

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
```

## Flow

1. Flutter `GET /api/auth/config` → `{ authkit: true }`
2. User clicks **Sign in with AuthKit** → `GET /api/auth/workos/authorize` → redirect to AuthKit
3. AuthKit returns to `/callback?code=...`
4. Flutter `POST /api/auth/workos/callback` with the code
5. Backend exchanges the code with WorkOS, upserts the local user (`workos_user_id`), issues the Roomies JWT
