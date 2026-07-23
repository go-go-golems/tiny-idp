# TinyIDP and Jitsi Kubernetes contract

This directory is the production-shaped deployment contract corresponding to
the locally validated stack in `examples/tinyidp-jitsi`. It intentionally
separates:

- Kustomize-managed TinyIDP, persistent state, ingress, network policy, and
  Vault Secrets Operator resources;
- the pinned upstream `jitsi-contrib/jitsi-helm` chart;
- reviewed non-secret JavaScript and OIDC client configuration;
- Vault-owned secret bytes.

It is not valid to apply the placeholder TinyIDP image tag. Replace
`sha-REPLACE_WITH_MERGED_COMMIT` with the immutable GHCR tag produced after the
plugin branch is merged.

## Required Vault policy

The Kubernetes auth role `tinyidp-jitsi` binds only service account
`tinyidp-jitsi` in namespace `tinyidp-jitsi`. Its policy grants read access to:

```text
kv/data/apps/tinyidp-jitsi/prod/runtime
kv/data/apps/tinyidp-jitsi/prod/image-pull
```

The source Vault record must contain the five keys documented in
`rotation-runbook.md`. VSO writes one Kubernetes Secret named
`tinyidp-jitsi-runtime`. Both TinyIDP and the Jitsi chart reference that Secret;
secret bytes never enter Git or Helm values.

The image-pull record follows the cluster's existing `server`, `username`,
`password`, and `auth` schema. VSO transforms it into the separate
`kubernetes.io/dockerconfigjson` Secret referenced by the TinyIDP service
account.

## Render and inspect

```bash
kubectl kustomize deploy/kubernetes/tinyidp-jitsi \
  > /tmp/tinyidp-jitsi-kustomize.yaml

helm template jitsi oci://ghcr.io/jitsi-contrib/jitsi-meet \
  --version 2.22.0 \
  --namespace tinyidp-jitsi \
  --values deploy/kubernetes/tinyidp-jitsi/jitsi-values.yaml \
  > /tmp/tinyidp-jitsi-helm.yaml

./deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh \
  /tmp/tinyidp-jitsi-helm.yaml
```

When access to the target API is available, validate all built-in and installed
custom-resource schemas without creating resources:

```bash
./deploy/kubernetes/tinyidp-jitsi/scripts/server-dry-run.sh \
  /tmp/tinyidp-jitsi-kustomize.yaml \
  /tmp/tinyidp-jitsi-helm.yaml
```

The script validates namespaced objects against the existing `default`
namespace because Kubernetes does not persist a dry-run Namespace before
evaluating later documents. It always passes `--dry-run=server`; it does not
apply the resources.

The chart's JVB uses host UDP port 10000 and advertises the node's public IP.
The Hetzner firewall and host firewall must permit `10000/udp`. Only one JVB
replica is configured because a single fixed host port cannot be shared by two
pods on the same node.

## GitOps ordering

1. Vault policy and Kubernetes auth role.
2. Namespace, service account, `VaultConnection`, and `VaultAuth`.
3. `VaultStaticSecret`, ConfigMap, and PVC.
4. TinyIDP and the pinned Jitsi Helm release.
5. Public ingresses.
6. Synthetic authentication and two-browser media checks.

TinyIDP's administration service has no Ingress. NetworkPolicy permits its
admin port only from the monitoring namespace. Traefik is the only public
caller allowed to reach the provider port.
