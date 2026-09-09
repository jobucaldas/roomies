#!/usr/bin/env bash
# Render and validate the bounded workload security-context contract.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
kustomize_bin=${KUSTOMIZE:-kustomize}
python_bin=${PYTHON:-python3}
kustomize_tree=${KUSTOMIZE_TREE:-"$root/deploy/kustomize"}
parser=${SECURITY_PARSER:-"$root/scripts/check_kustomize_security.py"}

for overlay in base overlays/production; do
    rendered=$(mktemp)
    trap 'rm -f "$rendered"' EXIT
    "$kustomize_bin" build "$kustomize_tree/$overlay" >"$rendered"
    "$python_bin" "$parser" "$rendered"
    rm -f "$rendered"
    trap - EXIT
done

printf '%s\n' 'Kubernetes workload security-context checks passed for base and production.'
