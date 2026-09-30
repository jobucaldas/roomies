#!/usr/bin/env bash
# Structured mutation tests for the rendered workload security contract.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
kustomize_bin=${KUSTOMIZE:-kustomize}
python_bin=${PYTHON:-python3}
kustomize_tree=${KUSTOMIZE_TREE:-"$root/deploy/kustomize"}
parser=${SECURITY_MUTATION_TEST:-"$root/scripts/test_kustomize_security.py"}

rendered=$(mktemp)
trap 'rm -f "$rendered"' EXIT
"$kustomize_bin" build "$kustomize_tree/base" >"$rendered"
PYTHONPATH="$root/scripts${PYTHONPATH:+:$PYTHONPATH}" "$python_bin" "$parser" "$rendered"
