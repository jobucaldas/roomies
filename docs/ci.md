# CI

`.github/workflows/ci.yml` runs on every push and pull request. The jobs are pre-commit, lint, backend (SQLite and Postgres), flutter (analyze, test, web build), android (release APK artifact) and containers (Compose build and health smoke). None of them needs secrets, so pull requests from forks run the full suite.

`.github/workflows/deploy-dev.yml` is a manual deploy to a development cluster; see [deploy/kustomize/overlays/dev/README.md](../deploy/kustomize/overlays/dev/README.md).

Third-party actions in jobs that can hold secrets are pinned to a commit SHA, with the version in a trailing comment. Update them by resolving the new tag to its commit.

## Android release signing

On pushes to `main`, the `android` job signs the APK with your upload key when these repository secrets exist. Every other run (branches, pull requests, forks) signs with debug keys, so a release-signed APK only comes from reviewed code.

| Secret | Value |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | `base64 -w0 upload-keystore.jks` |
| `ANDROID_KEYSTORE_PASSWORD` | keystore password |
| `ANDROID_KEY_ALIAS` | key alias |
| `ANDROID_KEY_PASSWORD` | key password (omit if it equals the keystore password) |

Create the keystore once and keep it outside the repository. Losing it means you cannot ship updates under the same signature.

```bash
keytool -genkeypair -v -keystore upload-keystore.jks -alias upload \
  -keyalg RSA -keysize 2048 -validity 10000

base64 -w0 upload-keystore.jks | gh secret set ANDROID_KEYSTORE_BASE64
gh secret set ANDROID_KEYSTORE_PASSWORD   # prompts
gh secret set ANDROID_KEY_ALIAS
```

Locally, the same values go in `src/flutter/android/key.properties` (gitignored): `storeFile` (relative to `android/`), `storePassword`, `keyAlias`, `keyPassword`. Without that file, release builds use debug keys.
