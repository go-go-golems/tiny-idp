# Coordinated Jitsi HS256 rotation

TinyIDP signs Jitsi room tokens with HS256. Prosody verifies them with the same
secret. The upstream Jitsi configuration accepts one active
`JWT_APP_SECRET`; therefore this deployment does not have an overlap window
where old and new keys are both valid.

The Vault KV v2 path is:

```text
kv/apps/tinyidp-jitsi/prod/runtime
```

It must contain:

```text
TINYIDP_TOKEN_SECRET   independent TinyIDP continuation/token key
JWT_APP_SECRET         shared only by TinyIDP and Prosody
JICOFO_AUTH_PASSWORD   independent printable 256-bit credential
JVB_AUTH_USER          jvb
JVB_AUTH_PASSWORD      independent printable 256-bit credential
```

Never reuse `TINYIDP_TOKEN_SECRET`, an OIDC client secret, or either XMPP
service credential as `JWT_APP_SECRET`.

## Rotation procedure

1. Announce a short meeting-admission maintenance window. Existing conferences
   can continue, but new admission is paused.
2. Scale the public TinyIDP deployment to zero. This prevents it from issuing
   tokens with the old secret while Prosody changes.
3. Write a new randomly generated, printable, at least 256-bit
   `JWT_APP_SECRET` as a new Vault KV v2 version. Do not change unrelated keys.
4. Wait until `VaultStaticSecret/tinyidp-jitsi-runtime` reports the new Vault
   version and its destination Secret has been updated.
5. VSO restarts TinyIDP and Prosody through `rolloutRestartTargets`. Because
   TinyIDP was scaled to zero, only Prosody becomes available at this point.
6. Wait for the Prosody StatefulSet rollout and readiness to complete.
7. Restore the TinyIDP replica count to one and wait for `/readyz`.
8. Run a synthetic browser join. Confirm that Prosody accepts the new token and
   that a malformed or previously captured old token is rejected.
9. Inspect logs, metrics, and audit output for secret bytes before ending the
   maintenance window.

Rollback means restoring the previous Vault KV version and repeating the same
ordered rollout. Do not run TinyIDP and Prosody concurrently with different
secret versions.
