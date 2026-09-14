#!/usr/bin/env bash
# Run inside nix develop .#container. No registry writes or cluster access.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
test -z "$(git status --porcelain --untracked-files=all)" || { echo "Release requires a clean checkout (including untracked files)" >&2; exit 1; }
sha=$(git rev-parse --verify "${RELEASE_SHA-HEAD}^{commit}")
test "$sha" = "$(git rev-parse HEAD)" || { echo "RELEASE_SHA must resolve to checked-out HEAD" >&2; exit 1; }
version=${RELEASE_VERSION-$sha}
[[ "$version" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]] && [[ "$version" != latest ]] || { echo "RELEASE_VERSION must be a non-latest OCI tag (1-128 characters)" >&2; exit 1; }
source=https://github.com/jobucaldas/roomies
final="$PWD/artifacts/release/$sha"
test ! -e "$final" || { echo "Bundle already exists: $final (move it before rebuilding)" >&2; exit 1; }
mkdir -p "${final%/*}"
out=$(mktemp -d "${final%/*}/.$sha.XXXXXX")
work=
trap 'rm -rf "$out"; if test -n "$work"; then rm -rf "$work"; fi' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
work=$(mktemp -d)
git archive "$sha" > "$out/source.tar"
tar -xf "$out/source.tar" -C "$work"
jq -n --arg revision "$sha" --arg version "$version" --arg source "$source" \
  '{schemaVersion:1,revision:$revision,version:$version,source:$source,publication:"none",sbom:"Syft SPDX JSON; local inventory, not an attestation"}' > "$out/release.json"
{ podman --version; skopeo --version; kustomize version; syft version; } > "$out/tool-versions.txt"
cp "$work/flake.lock" "$out/flake.lock"
cp -R "$work/deploy/kustomize" "$out/kustomize"
for component in backend frontend; do
    image="localhost/roomies-$component:$sha"
    context="$work"
    dockerfile="$work/src/app/Dockerfile"
    if test "$component" = backend; then
        context="$work/src/backend"
        dockerfile="$context/Dockerfile"
    fi
    podman build --label "org.opencontainers.image.source=$source" \
        --label "org.opencontainers.image.revision=$sha" \
        --label "org.opencontainers.image.version=$version" \
        -t "$image" -f "$dockerfile" "$context"
    podman save --format oci-archive -o "$out/$component.oci.tar" "$image"
    skopeo inspect --raw "oci-archive:$out/$component.oci.tar" > "$out/$component.manifest.json"
    skopeo inspect --config "oci-archive:$out/$component.oci.tar" > "$out/$component.config.json"
    digest="sha256:$(sha256sum "$out/$component.manifest.json" | cut -d ' ' -f1)"
    jq -e --arg sha "$sha" --arg version "$version" --arg source "$source" '.config.Labels | .["org.opencontainers.image.revision"] == $sha and .["org.opencontainers.image.version"] == $version and .["org.opencontainers.image.source"] == $source' "$out/$component.config.json" >/dev/null
    jq -n --arg image "$image" --arg digest "$digest" --arg archive "$component.oci.tar" \
      '{localTag:$image,archive:$archive,ociManifestDigest:$digest,published:false}' > "$out/$component.image.json"
    (cd "$out/kustomize/overlays/production"; kustomize edit set image "roomies-$component=$image")
done
# Capture the third-party runtime locally too; do not pretend its local digest is published.
podman pull docker.io/caddy:2.8-alpine
podman save --format oci-archive -o "$out/caddy.oci.tar" docker.io/caddy:2.8-alpine
skopeo inspect --raw "oci-archive:$out/caddy.oci.tar" > "$out/caddy.manifest.json"
skopeo inspect --config "oci-archive:$out/caddy.oci.tar" > "$out/caddy.config.json"
digest="sha256:$(sha256sum "$out/caddy.manifest.json" | cut -d ' ' -f1)"
jq -n --arg digest "$digest" '{localTag:"docker.io/caddy:2.8-alpine",archive:"caddy.oci.tar",ociManifestDigest:$digest,published:false}' > "$out/caddy.image.json"
# Inspect inventory without credentials, updates, or registry access.
for component in backend frontend caddy; do
    mkdir "$work/$component-oci"
    tar -xf "$out/$component.oci.tar" -C "$work/$component-oci"
    python3 "$work/scripts/validate-oci.py" "$work/$component-oci"
    digest=$(jq -r '.ociManifestDigest' "$out/$component.image.json")
    jq -e --arg digest "$digest" '.manifests | any(.digest == $digest)' "$work/$component-oci/index.json" >/dev/null
    cmp "$out/$component.manifest.json" "$work/$component-oci/blobs/sha256/${digest#sha256:}"
    config=$(jq -r '.config.digest' "$out/$component.manifest.json")
    diff <(jq -S . "$out/$component.config.json") <(jq -S . "$work/$component-oci/blobs/sha256/${config#sha256:}")
    cp "$work/$component-oci/blobs/sha256/${config#sha256:}" "$out/$component.config.json"
    cp "$work/$component-oci/index.json" "$out/$component.index.json"
    SYFT_CHECK_FOR_APP_UPDATE=false syft "oci-archive:$out/$component.oci.tar" -o "spdx-json=$out/$component.spdx.json"
done
kustomize build "$out/kustomize/overlays/production" > "$out/production.yaml"
! grep -q ':latest' "$out/production.yaml"
for component in backend frontend; do grep -q "localhost/roomies-$component:$sha" "$out/production.yaml"; done
cat > "$out/README.txt" <<'NOTE'
Local-only bundle from a clean committed source. Read release.json for identity.
*.oci.tar are local OCI archives; *.manifest.json are their actual raw manifests.
*.image.json records SHA-256 of those manifest bytes, NOT a registry pull digest.
production.yaml uses local SHA tags for Roomies and a version tag for Caddy.
Load archives and arrange local tags on every test node before any local deployment.
Nothing was pushed; these references are not production registry coordinates.
Base tags/dependency downloads can change: repeatable procedure, not bit-reproducible builds.
Checksums cover all bundle files except SHA256SUMS itself. SBOMs are local inventories,
not signed attestations. Scanner IDs/timestamps and builds are not byte-reproducible.
NOTE
(cd "$out"; find . -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 sha256sum > SHA256SUMS; sha256sum -c SHA256SUMS)
mv -T --no-clobber "$out" "$final"
test ! -d "$out" || { echo "Bundle destination appeared during build" >&2; exit 1; }
echo "Local release bundle: $final"
