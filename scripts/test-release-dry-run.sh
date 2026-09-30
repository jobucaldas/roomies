#!/usr/bin/env bash
# Offline contract tests: disposable committed repository and failing tool shim.
set -euo pipefail
root=$(git rev-parse --show-toplevel)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/repo/scripts" "$fixture/tools"
cp "$root/scripts/release-dry-run.sh" "$fixture/repo/scripts/"
cd "$fixture/repo"
git init -q
git config user.email test@example.invalid
git config user.name release-test
echo lock > flake.lock
mkdir -p deploy/kustomize
echo config > deploy/kustomize/fixture
git add .; git commit -qm fixture
old=$(git rev-parse HEAD)
echo next >> flake.lock; git commit -qam next
sha=$(git rev-parse HEAD)
cat > "$fixture/tools/podman" <<'SH'
#!/bin/sh
echo 'induced podman failure' >&2
exit 42
SH
chmod +x "$fixture/tools/podman"
export PATH="$fixture/tools:$PATH"
reject() {
    if "$@" > "$fixture/result" 2>&1; then echo 'Unexpected success' >&2; exit 1; fi
    test ! -e "artifacts/release/$sha"
    test -z "$(find artifacts/release -mindepth 1 -maxdepth 1 2>/dev/null || :)"
}
reject env RELEASE_SHA="$old" bash scripts/release-dry-run.sh
grep -q 'must resolve' "$fixture/result"
for version in '' latest 'bad/version'; do
    reject env RELEASE_VERSION="$version" bash scripts/release-dry-run.sh
    grep -q 'RELEASE_VERSION' "$fixture/result"
done
for file in flake.lock deploy/kustomize/fixture untracked-fixture; do
    echo dirty >> "$file"
    reject bash scripts/release-dry-run.sh
    grep -q 'clean checkout' "$fixture/result"
    if test "$file" = untracked-fixture; then rm "$file"; else git restore "$file"; fi
done
# Valid explicit identity reaches the induced failure, twice: no poisoned output.
for attempt in 1 2; do
    reject env RELEASE_SHA="$sha" RELEASE_VERSION=review-1 bash scripts/release-dry-run.sh
    grep -q 'induced podman failure' "$fixture/result"
done
echo 'Identity, dirty-source, atomic cleanup and retry contract tests passed'
