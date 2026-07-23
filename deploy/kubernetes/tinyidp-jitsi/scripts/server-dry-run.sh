#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <kustomize-render.yaml> <helm-render.yaml>" >&2
  exit 2
fi

for source in "$@"; do
  if [[ ! -s "${source}" ]]; then
    echo "rendered manifest is missing or empty: ${source}" >&2
    exit 2
  fi
done

temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

# Server-side dry-run does not persist a Namespace document before evaluating
# later documents in the same stream. Validate namespaced schemas against the
# already-existing default namespace while leaving the Namespace object's name
# and every workload spec unchanged.
sed 's/namespace: tinyidp-jitsi/namespace: default/g' \
  "$1" >"${temporary_directory}/kustomize.yaml"
sed 's/namespace: tinyidp-jitsi/namespace: default/g' \
  "$2" >"${temporary_directory}/helm.yaml"

kubectl apply --dry-run=server \
  -f "${temporary_directory}/kustomize.yaml"
kubectl apply --dry-run=server \
  -f "${temporary_directory}/helm.yaml"

echo "OK: live API accepted TinyIDP, VSO, and Jitsi resources in server-side dry-run"
