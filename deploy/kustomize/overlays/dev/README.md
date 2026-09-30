# Dev overlay (home-lab)

CI/dev target for the private home-lab cluster. Do **not** apply
`overlays/production` from CI.

## Prerequisites (one-time, out of band)

1. Namespace `roomies-dev` (also created by this overlay).
2. Secret `roomies-secrets` with at least `DATABASE_URL` and `JWT_SECRET`
   (see `docs/credentials.example.env`). SMTP/push may be stubbed for dev.
3. Pull secret `roomies-ghcr` (`kubernetes.io/dockerconfigjson`) for
   `ghcr.io/jobucaldas/*` with `read:packages` only.
4. In-cluster Postgres (or external) matching `DATABASE_URL`.
5. Cloudflare Tunnel hostname `roomies-dev.jobucaldas.com` → Traefik
   (same pattern as sibling `*.jobucaldas.com` apps).

## Render

```bash
kubectl kustomize deploy/kustomize/overlays/dev
```

CI pins image tags to `sha-<git-sha>` before apply. Deploy is
**manual only** (`workflow_dispatch` on Deploy Dev) until cluster
bootstrap above is confirmed — see `.github/workflows/deploy-dev.yml`.
