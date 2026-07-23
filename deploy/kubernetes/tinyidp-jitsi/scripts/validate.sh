#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
deploy_dir="${root}/deploy/kubernetes/tinyidp-jitsi"
rendered="$(mktemp)"
trap 'rm -f "${rendered}"' EXIT
helm_render="${1:-}"

kubectl kustomize "${deploy_dir}" >"${rendered}"

grep -q 'kind: VaultStaticSecret' "${rendered}"
grep -q 'secretName: tinyidp-jitsi-runtime' "${rendered}"
grep -q 'path: jitsi-shared-secret' "${rendered}"
grep -q 'path: /readyz' "${rendered}"
grep -q 'path: /healthz' "${rendered}"
grep -q 'kind: NetworkPolicy' "${rendered}"

if grep -Eq 'JWT_APP_SECRET:[[:space:]]+[^[:space:]]' "${deploy_dir}/jitsi-values.yaml"; then
  echo "JWT_APP_SECRET must not be stored in Helm values" >&2
  exit 1
fi

if rg -n --glob '*.yaml' --glob '*.yml' \
  '(BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY|password:[[:space:]]+[^[:space:]]+|secret:[[:space:]]+[^[:space:]]+)' \
  "${deploy_dir}"; then
  echo "possible inline secret material found" >&2
  exit 1
fi

grep -q 'existingSecretName: tinyidp-jitsi-runtime' "${deploy_dir}/jitsi-values.yaml"
grep -q 'JWT_APP_ID: tinyidp-jitsi-prod' "${deploy_dir}/jitsi-values.yaml"
grep -q 'JWT_ACCEPTED_ISSUERS: tinyidp-jitsi-prod' "${deploy_dir}/jitsi-values.yaml"
grep -q 'JWT_ACCEPTED_AUDIENCES: tinyidp-jitsi-prod' "${deploy_dir}/jitsi-values.yaml"
grep -q 'JWT_ALLOW_EMPTY: "0"' "${deploy_dir}/jitsi-values.yaml"
grep -q 'JWT_SIGN_TYPE: HS256' "${deploy_dir}/jitsi-values.yaml"

if [[ -n "${helm_render}" ]]; then
  for target in \
    jitsi-jitsi-meet-prosody \
    jitsi-jitsi-meet-jicofo \
    jitsi-jitsi-meet-jvb-0; do
    grep -q "name: ${target}" "${helm_render}"
    grep -q "name: ${target}" "${deploy_dir}/runtime-secret.yaml"
  done
  grep -q 'name: tinyidp-jitsi-runtime' "${helm_render}"
  grep -q 'hostPort: 10000' "${helm_render}"
  grep -q 'TOKEN_AUTH_URL: "https://idp-jitsi.yolo.scapegoat.dev/integrations/jitsi/start?room={room}"' "${helm_render}"
fi

echo "OK: TinyIDP Kubernetes, VSO, and Jitsi shared-secret contracts are coherent"
