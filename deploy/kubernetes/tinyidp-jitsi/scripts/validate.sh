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
grep -q -- '--theme-dir=/config' "${deploy_dir}/deployment.yaml"
grep -q 'cp /run/tinyidp-source-secrets/token-secret /run/tinyidp-runtime-secrets/token-secret' "${deploy_dir}/deployment.yaml"
grep -q 'chown 65532:65532 /run/tinyidp-runtime-secrets/token-secret /run/tinyidp-runtime-secrets/jitsi-shared-secret' "${deploy_dir}/deployment.yaml"
grep -q 'chmod 0400 /run/tinyidp-runtime-secrets/token-secret /run/tinyidp-runtime-secrets/jitsi-shared-secret' "${deploy_dir}/deployment.yaml"
grep -q -- '--token-secret-file=/run/tinyidp-runtime-secrets/token-secret' "${deploy_dir}/deployment.yaml"

# The init container repairs persisted state. It must reacquire ownership of
# /state itself before traversing a UID-65532, mode-0700 directory from a
# previous run. It must likewise reclaim an existing private audit child
# before the final recursive ownership handoff descends into it.
reacquire_line="$(rg -n '^\s*chown 0:0 /state$' "${deploy_dir}/deployment.yaml" | cut -d: -f1)"
audit_reacquire_line="$(rg -n '^\s*chown 0:0 /state/audit$' "${deploy_dir}/deployment.yaml" | cut -d: -f1)"
mkdir_line="$(rg -n 'mkdir -p /state/audit' "${deploy_dir}/deployment.yaml" | cut -d: -f1)"
chmod_line="$(rg -n 'chmod 0700 /state/audit' "${deploy_dir}/deployment.yaml" | tail -n1 | cut -d: -f1)"
chown_line="$(rg -n 'chown -R 65532:65532 /state' "${deploy_dir}/deployment.yaml" | cut -d: -f1)"
if [[ -z "${reacquire_line}" || -z "${audit_reacquire_line}" || -z "${mkdir_line}" || -z "${chmod_line}" || -z "${chown_line}" || "${reacquire_line}" -ge "${audit_reacquire_line}" || "${audit_reacquire_line}" -ge "${mkdir_line}" || "${mkdir_line}" -ge "${chmod_line}" || "${chmod_line}" -ge "${chown_line}" ]]; then
  echo "TinyIDP state permissions must reclaim private state and audit directories before handoff" >&2
  exit 1
fi
if ! rg -q 'add: \[CHOWN, FOWNER\]' "${deploy_dir}/deployment.yaml"; then
  echo "TinyIDP state permissions require CHOWN and FOWNER for existing PVC state" >&2
  exit 1
fi

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
