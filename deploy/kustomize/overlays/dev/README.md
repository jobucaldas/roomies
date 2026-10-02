# Dev overlay

The maintainer's own development deployment, kept as a working example of `deploy/kustomize/base`. Copy it and change the values for your own server.

What to change in a copy:

- `kustomization.yaml`: `ROOMIES_PUBLIC_BASE_URL` / `CORS_ALLOWED_ORIGINS` (your public address) and the image owner if you publish your own images.
- `ingress.yaml`: your hostname.
- `image-pull-secrets.yaml`, `nightly-updater.yaml`: the `kubernetes.io/hostname` node selector (or remove it).

## Images

The overlay tracks the rolling GHCR tags that CI publishes on every push to `main`:

- `ghcr.io/jobucaldas/roomies-backend:nightly` (backend and worker)
- `ghcr.io/jobucaldas/roomies-frontend:nightly`

CI also publishes `:dev` and immutable `YYYYMMDDHHMMSS_<shortsha>` tags. The overlay uses `:nightly` with `imagePullPolicy: Always`.

A CronJob (`roomies-nightly-updater`, every 15 minutes) compares the GHCR manifest digest for `:nightly` with the running pods and runs `kubectl rollout restart` only when the digest moved. CI needs no cluster credentials.

## Prerequisites (one-time)

1. Namespace `roomies-dev` (also created by this overlay).
2. Secret `roomies-secrets` with at least `DATABASE_URL` and `JWT_SECRET` (see `docs/credentials.example.env`). The backend refuses to start without `JWT_SECRET` on a non-localhost address.
3. Postgres reachable at `DATABASE_URL`.
4. GHCR pull secret `roomies-ghcr`, used to pull images and by the updater to read digests:

```bash
kubectl -n roomies-dev create secret docker-registry roomies-ghcr \
  --docker-server=ghcr.io \
  --docker-username=<github-user> \
  --docker-password="$(gh auth token)"
```

The token needs at least `read:packages`. Re-create the secret when the token rotates.

## Render / apply

```bash
kubectl kustomize deploy/kustomize/overlays/dev
kubectl apply -k deploy/kustomize/overlays/dev
```

Force an immediate digest check:

```bash
kubectl -n roomies-dev create job --from=cronjob/roomies-nightly-updater \
  "roomies-nightly-updater-manual-$(date +%s)"
```
