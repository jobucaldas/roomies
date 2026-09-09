#!/usr/bin/env bash
# Static checks for the bounded workload security-context contract.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
kustomize_bin=${KUSTOMIZE:-kustomize}
kustomize_tree=${KUSTOMIZE_TREE:-"$root/deploy/kustomize"}

for overlay in base overlays/production; do
    rendered=$(mktemp)
    trap 'rm -f "$rendered"' EXIT
    "$kustomize_bin" build "$kustomize_tree/$overlay" >"$rendered"

    # The base has four Roomies workload containers: backend, worker, bundle,
    # and Caddy. Keep these checks count-based so a new workload cannot silently
    # omit the required controls without changing this bounded contract.
    test "$(grep -c '^ *runAsNonRoot: true$' "$rendered")" -ge 4
    test "$(grep -c '^ *allowPrivilegeEscalation: false$' "$rendered")" -eq 4
    test "$(grep -c '^ *readOnlyRootFilesystem: true$' "$rendered")" -eq 4
    test "$(grep -c '^ *runAsUser:' "$rendered")" -eq 4
    test "$(grep -c '^ *runAsGroup:' "$rendered")" -eq 4
    test "$(grep -c '^ *type: RuntimeDefault$' "$rendered")" -eq 3
    test "$(grep -c '^ *- ALL$' "$rendered")" -eq 4
    ! grep -q '^ *privileged: true$' "$rendered"
    ! grep -q '^ *hostPath:' "$rendered"

    test "$(grep -c '^ *:8080$' "$rendered")" -eq 1
    test "$(grep -c 'containerPort: 8080$' "$rendered")" -eq 2
    test "$(grep -c 'containerPort: 8081$' "$rendered")" -eq 1
    test "$(grep -c '^ *port: 80$' "$rendered")" -eq 1
    test "$(grep -c '^ *targetPort: http$' "$rendered")" -eq 2
    test "$(grep -c '^ *port: health$' "$rendered")" -eq 2
    ! grep -q 'containerPort: 80$' "$rendered"

    rm -f "$rendered"
    trap - EXIT
 done

printf '%s\n' 'Kubernetes workload security-context checks passed for base and production.'
