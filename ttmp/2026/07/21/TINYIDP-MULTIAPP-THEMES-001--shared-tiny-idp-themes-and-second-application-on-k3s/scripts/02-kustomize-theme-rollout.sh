#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 /absolute/path/to/gitops/kustomize/tiny-message-desk" >&2
  exit 2
fi

source_dir=$1
if [[ ! -f "${source_dir}/kustomization.yaml" || ! -f "${source_dir}/themes/goja-auth.css" ]]; then
  echo "the argument is not the tiny-message-desk Kustomize source" >&2
  exit 2
fi

probe_dir=$(mktemp -d)
trap 'rm -rf "${probe_dir}"' EXIT
cp -a "${source_dir}/." "${probe_dir}/"

kubectl kustomize "${source_dir}" >"${probe_dir}/before.yaml"
printf '\n/* content-hash rollout probe */\n' >>"${probe_dir}/themes/goja-auth.css"
kubectl kustomize "${probe_dir}" >"${probe_dir}/after.yaml"

before_theme=$(sed -n 's/^  name: \(tinyidp-theme-catalog-[a-z0-9]*\)$/\1/p' "${probe_dir}/before.yaml" | sort -u)
after_theme=$(sed -n 's/^  name: \(tinyidp-theme-catalog-[a-z0-9]*\)$/\1/p' "${probe_dir}/after.yaml" | sort -u)
before_clients=$(sed -n 's/^  name: \(tinyidp-client-catalog-[a-z0-9]*\)$/\1/p' "${probe_dir}/before.yaml" | sort -u)
after_clients=$(sed -n 's/^  name: \(tinyidp-client-catalog-[a-z0-9]*\)$/\1/p' "${probe_dir}/after.yaml" | sort -u)

if [[ -z "${before_theme}" || -z "${after_theme}" || "${before_theme}" == "${after_theme}" ]]; then
  echo "theme content edit did not alter the generated ConfigMap reference" >&2
  exit 1
fi
if [[ -z "${before_clients}" || "${before_clients}" != "${after_clients}" ]]; then
  echo "theme-only edit unexpectedly altered the client catalog reference" >&2
  exit 1
fi

echo "PASS theme ConfigMap reference changed: ${before_theme} -> ${after_theme}"
echo "PASS independent client catalog reference remained: ${before_clients}"
