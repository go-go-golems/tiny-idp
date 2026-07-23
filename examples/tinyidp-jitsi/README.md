# TinyIDP Jitsi plugin local stack

This example runs a production-shaped TinyIDP Jitsi integration on one
workstation. TinyIDP authenticates browser users and mints short-lived,
room-bound Jitsi JWTs. Prosody validates those JWTs locally. Jicofo coordinates
the conference and Jitsi Videobridge transports media.

The stack uses the pinned official Jitsi `stable-10978` images and exposes:

- `https://idp.localhost:8443` for TinyIDP.
- `https://meet.localhost:8443` for Jitsi Meet.
- `127.0.0.1:10000/udp` for local JVB media.

The TinyIDP administration listener, Prosody, Jicofo, and JVB are not published
to the host network.

## Start the stack

Initialize local-only secrets and the persistent Caddy development CA:

```bash
./examples/tinyidp-jitsi/scripts/00-init-secrets.sh
```

Start the services in tmux:

```bash
tmux new-session -s tinyidp-jitsi \
  'docker compose -f examples/tinyidp-jitsi/compose.yaml up --build'
```

Export the public CA certificate:

```bash
./examples/tinyidp-jitsi/scripts/01-export-browser-ca.sh
```

The CA private key remains inside the manually retained Docker volume
`tinyidp-local-caddy-pki`. Only the public certificate is copied into
`runtime/`. Deleting the volume invalidates every local certificate issued by
that CA. Do not copy or commit the volume contents.

Run the HTTP and configuration smoke checks:

```bash
./examples/tinyidp-jitsi/scripts/02-smoke.sh
```

Run the browser and conference checks:

```bash
./examples/tinyidp-jitsi/scripts/03-browser-tests.sh
```

## Local identity flows

The deterministic local administrator is:

```text
login:    admin@example.test
password: local-jitsi-admin-password-2026!
```

The browser suite also provisions an identity with no email claim so the Goja
policy-denial path can be tested deterministically:

```text
login:    denied@example.test
password: local-jitsi-policy-denied-password-2026!
```

Jitsi sends an unauthenticated participant to:

```text
/integrations/jitsi/start?room=<room>
```

The plugin supports two additional, explicit browser intents for exercising
TinyIDP-owned identity workflows:

```text
/integrations/jitsi/start?room=<room>&intent=signup
/integrations/jitsi/start?room=<room>&prompt=select_account
```

These values are parsed into typed broker fields. Arbitrary OAuth parameters
are not forwarded.

## Shared contract

TinyIDP and Prosody receive the same Jitsi-only values:

```text
application/issuer/audience: tinyidp-jitsi-local
XMPP domain / JWT subject:   meet.localhost
algorithm:                   HS256
secret:                      runtime/secrets/jitsi-shared-secret.key
```

TinyIDP reads the secret through a Compose secret and owner-only runtime copy.
Prosody reads the same Compose secret into `JWT_APP_SECRET`. Neither service
prints it. Jicofo and JVB use separate printable 256-bit passwords because
their generated HOCON configuration cannot safely contain arbitrary binary
bytes.

## Expected authentication sequence

```text
Browser -> Jitsi prejoin
Browser -> Prosody without JWT
Prosody -> Browser: token required
Jitsi -> TinyIDP plugin /start
Plugin -> TinyIDP /authorize (state, nonce, PKCE)
Browser -> login, signup, or account chooser
TinyIDP -> plugin /callback?code=...&state=...
Plugin -> policy -> short-lived Jitsi JWT
Plugin -> Browser: /room?jwt=...
Browser -> Prosody with JWT
Prosody -> Browser: authenticated
Browser -> Jicofo -> JVB conference
```

## State and cleanup

Database state, audit records, generated Jitsi configuration, and Caddy state
use named volumes. `docker compose down` preserves them. To remove the example
state while retaining the shared CA:

```bash
docker compose -f examples/tinyidp-jitsi/compose.yaml down -v
```

The external `tinyidp-local-caddy-pki` volume is deliberately not removed by
that command.
