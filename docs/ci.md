# CI

Pushes and pull requests run on [Depot CI](https://depot.dev/docs/ci/overview) from `.depot/workflows/ci.yml`. The jobs are pre-commit, lint, backend (SQLite and Postgres), flutter (analyze, test, web build), android (release APK artifact) and containers (Compose build and health smoke).

`.github/workflows/ci.yml` holds the same jobs on GitHub-hosted runners but only runs when started by hand (Actions → CI → Run workflow). Keep the two files in sync; they differ only in the `on:` block and `runs-on` labels.

## One-time setup

1. Create a Depot organization at [depot.dev](https://depot.dev).
2. Depot dashboard → Organization Settings → GitHub Code Access → **Connect to GitHub**, then install the Depot Code Access app on `jobucaldas/roomies`. Depot CI supports repositories owned by personal accounts.
3. Merge a commit containing `.depot/workflows/` to the default branch so Depot registers the push and pull request triggers.

The CI workflow needs no secrets. `deploy-dev.yml` stays on GitHub Actions (manual dispatch with Tailscale and kubeconfig secrets).

## CLI

```bash
# Run the workflow against the local working tree, no push needed.
depot ci run --workflow .depot/workflows/ci.yml
# One job, streaming its logs.
depot ci run --workflow .depot/workflows/ci.yml --job flutter --follow
```

See the [Depot CI command reference](https://depot.dev/docs/cli/reference/depot-ci) for the full list.
