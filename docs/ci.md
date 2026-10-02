# CI

Pushes and pull requests run on [Depot CI](https://depot.dev/docs/ci/overview) from `.depot/workflows/ci.yml`. The jobs are pre-commit, lint, backend (SQLite and Postgres), flutter (analyze, test, web build), android (release APK artifact) and containers (Compose build and health smoke).

`.github/workflows/ci.yml` holds the same jobs on GitHub-hosted runners but only runs when started by hand (Actions → CI → Run workflow). Keep the two files in sync; they differ only in the `on:` block and `runs-on` labels.

## One-time setup

1. Create a Depot organization at [depot.dev](https://depot.dev).
2. Depot dashboard → Organization Settings → GitHub Code Access → **Connect to GitHub**, then install the Depot Code Access app on `jobucaldas/roomies`. Depot CI supports repositories owned by personal accounts.
3. Merge a commit containing `.depot/workflows/` to the default branch so Depot registers the push and pull request triggers.

The CI workflow needs no secrets to pass; the Android signing secrets below are optional. `deploy-dev.yml` stays on GitHub Actions (manual dispatch with Tailscale and kubeconfig secrets).

## Android release signing

The `android` job signs the release APK with your upload key when these Depot CI secrets exist, and with debug keys otherwise (fine for CI, not for publishing):

| Secret | Value |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | `base64 -w0 upload-keystore.jks` |
| `ANDROID_KEYSTORE_PASSWORD` | keystore password |
| `ANDROID_KEY_ALIAS` | key alias |
| `ANDROID_KEY_PASSWORD` | key password (omit if it equals the keystore password) |

Create a keystore once and keep it outside the repo (losing it means you cannot ship updates under the same signature):

```bash
keytool -genkeypair -v -keystore upload-keystore.jks -alias upload \
  -keyalg RSA -keysize 2048 -validity 10000
```

Add the secrets in the Depot dashboard (Depot CI settings) or with the CLI:

```bash
base64 -w0 upload-keystore.jks | depot ci secrets add ANDROID_KEYSTORE_BASE64 --repo jobucaldas/roomies
depot ci secrets add ANDROID_KEYSTORE_PASSWORD --repo jobucaldas/roomies   # prompts
depot ci secrets add ANDROID_KEY_ALIAS --repo jobucaldas/roomies
```

Locally, the same keys go in `src/flutter/android/key.properties` (gitignored): `storeFile` (relative to `android/`), `storePassword`, `keyAlias`, `keyPassword`.

## CLI

```bash
# Run the workflow against the local working tree, no push needed.
depot ci run --workflow .depot/workflows/ci.yml
# One job, streaming its logs.
depot ci run --workflow .depot/workflows/ci.yml --job flutter --follow
```

See the [Depot CI command reference](https://depot.dev/docs/cli/reference/depot-ci) for the full list.
