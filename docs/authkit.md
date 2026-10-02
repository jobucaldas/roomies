# WorkOS AuthKit

Roomies uses WorkOS **hosted AuthKit** for sign-in when `WORKOS_API_KEY` and `WORKOS_CLIENT_ID` are set. The Flutter client shows a single **Sign in** action that redirects to AuthKit; sign-up, Google and other enabled methods live on the hosted AuthKit page. Without those variables, the app keeps local email/password (used by CI), collapsed behind **Use email and password**. When AuthKit is enabled, `POST /api/auth/login` and `POST /api/auth/register` return 403.

## WorkOS Dashboard URLs

AuthKit's own emails, IdP flows and logout links use the URLs configured in the WorkOS Dashboard, not only the `redirect_uri` Roomies sends. Point them at the public origin users open (examples use `https://roomies.example.com`):

| Setting | Value |
|---|---|
| Redirect URI | `https://roomies.example.com/callback` |
| Initiate login / homepage / sign-up | `https://roomies.example.com/` |
| Default logout return | `https://roomies.example.com/` |

Add every other origin you sign in from (for example `http://localhost/callback` for local Compose) as an extra redirect URI. Use one host for both the app and the callback so the session cookie and the callback share an origin.

Roomies always sends an explicit `redirect_uri` (`WORKOS_REDIRECT_URI`, or `{ROOMIES_PUBLIC_BASE_URL}/callback`).

## Google OAuth (AuthKit social login)

Google appears on the AuthKit page when enabled and credentials are available. Docs: [AuthKit](https://workos.com/docs/authkit), [Social Login](https://workos.com/docs/authkit/social-login), [Google OAuth](https://workos.com/docs/integrations/google-oauth).

### Staging (quick test)

1. In the WorkOS Dashboard for the Staging environment, open Authentication → OAuth providers → Google.
2. Enable Google for AuthKit.
3. Staging can use WorkOS default Google credentials for testing (WorkOS branding on the consent screen until you add your own client ID/secret).
4. Confirm Redirect URIs still include your app callback (`…/callback`) for each public base URL.

### Production (your Google Cloud project)

1. In [Google Cloud Console](https://console.cloud.google.com/), create (or select) a project and configure the OAuth consent screen.
2. Create an OAuth Web application client. Add the **Redirect URI shown in the WorkOS Dashboard** Google dialog (not the Roomies `/callback` URL) as an authorized redirect URI.
3. Copy the Google Client ID and Client Secret into WorkOS → Authentication → Google → your app credentials, then save.
4. Enable Google for AuthKit in that environment.
5. Publish the Google OAuth app so non-test users are not blocked.

Do **not** put Google client secrets in Roomies env files. Only WorkOS needs them.

## App env

```bash
WORKOS_API_KEY=sk_...          # from WorkOS Dashboard → API Keys
WORKOS_CLIENT_ID=client_...
WORKOS_REDIRECT_URI=https://roomies.example.com/callback   # optional; defaults to {ROOMIES_PUBLIC_BASE_URL}/callback
JWT_ACCESS_TTL_HOURS=8         # session cookie and bearer lifetime (default 8h)
```

For local Compose, `http://localhost/callback` (or `:58000`) works. Configure matching redirect, initiate-login and logout-return URIs in the WorkOS Dashboard for each environment.

## Flow

1. Client `GET /api/auth/config` → `{ authkit: true, password: false }`
2. User taps **Sign in** → `GET /api/auth/workos/authorize` sets HttpOnly `state` + PKCE cookies and returns an AuthKit URL
3. User may pick Google (or email/password/magic) on the AuthKit hosted page
4. AuthKit returns to `/callback?code=...&state=...`
5. Client `POST /api/auth/workos/callback` with the code and state. The browser sends the cookies. The server checks state and sends the PKCE verifier to WorkOS.
6. Backend upserts the local user (`workos_user_id`) and sets an HttpOnly session cookie (`JWT_ACCESS_TTL_HOURS`, default 8h). The web client does not receive or store a bearer token. Native and API clients still receive a bearer token of the same lifetime.
7. Login, register, and the code exchange are limited per client IP.
8. **Log out** clears the Roomies session. When the session came from AuthKit, `POST /api/auth/logout` returns `{ "logout_url": "https://api.workos.com/user_management/sessions/logout?session_id=…" }` and the web client navigates there, so WorkOS ends its session and sends the browser to the dashboard's **Default logout return**. Without this, the next **Sign in** would silently reuse the previous account. Password and pre-existing sessions get `204` and stay in the app.
9. Fresh sign-in lands on Dashboard. Creating a house stores it as the default house (client preference). Restored sessions open that house; switch via sidebar or Settings.

Invitation acceptance matches the invite address to `users.email`.

## AuthKit branding

Match the hosted page to the app in WorkOS Dashboard → Branding so the hand-off looks like one flow:

| Setting | Light | Dark |
|---|---|---|
| Page background | `#F5F5F4` | `#121212` |
| Button / accent (mint) | `#0F7F5C` | `#21C68F` |
| Button / accent (plum) | `#53134B` | `#D38AC7` |
| Logo / wordmark | "Roomies" in Fraunces Bold, `#0D7353` | `#5FD6A8` |
| Body font | Source Sans 3 | Source Sans 3 |

The font files are bundled in `src/flutter/assets/fonts/` (SIL Open Font License).
