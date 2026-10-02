# Dev overlay

Development deployment applied by the manual **Deploy Dev** workflow (`.github/workflows/deploy-dev.yml`). Do **not** apply `overlays/production` from CI.

## Prerequisites (one-time, out of band)

1. Namespace `roomies-dev` (also created by this overlay).
2. Secret `roomies-secrets` with at least `DATABASE_URL` and `JWT_SECRET` (see `docs/credentials.example.env`). SMTP and push may be stubbed for dev.
3. Pull secret `roomies-ghcr` (`kubernetes.io/dockerconfigjson`) for `ghcr.io/jobucaldas/*` with `read:packages` only.
4. Postgres reachable at `DATABASE_URL`.
5. Public hostname (`ingress.yaml`) routed to the cluster's ingress controller.

The workflow needs repository secrets `TS_OAUTH_CLIENT_ID`, `TS_AUDIENCE` and `KUBE_CONFIG`, and the repository variable `KUBE_API_SERVER` (the kubeconfig server URL it is allowed to deploy to).

## Render

```bash
kubectl kustomize deploy/kustomize/overlays/dev
```

The workflow pins image tags to `sha-<git-sha>` through a temporary overlay on top of this one, then applies it.
