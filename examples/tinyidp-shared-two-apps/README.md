# Shared TinyIDP local HTTPS development stack

This Compose project runs the production-shaped two-application topology on a
developer workstation. One strict TinyIDP process serves Message Desk and the
go-go-goja auth-host demo through distinct OIDC client registrations and
distinct identity-page themes. Caddy terminates local HTTPS; every Go process
uses its trusted-proxy listener mode and validates the public origin.

The public endpoints are:

- `https://message.localhost:8443` — Message Desk, including open signup.
- `https://goja.localhost:8443` — the generated go-go-goja auth-host demo.
- `https://idp.localhost:8443` — TinyIDP's canonical issuer. Start login or
  signup from an application, not by opening `/authorize` without parameters.

The local-only, email-verified operator fixtures are:

- `admin@example.test` / `local-admin-password-2026!` — bootstrapped as the
  administrator of the goja demo organization.
- `invitee@example.test` / `local-invitee-password-2026!` — has no initial
  application membership and is used to prove existing-user invitation
  acceptance.

These credentials are generated only under the gitignored `runtime/secrets/`
directory and are deliberately unsuitable for any shared or production
environment.

## Start and verify

Run from this directory:

```sh
./scripts/00-init-secrets.sh
docker compose up --build -d
./scripts/01-export-browser-ca.sh
./scripts/02-smoke.sh
./scripts/03-browser-acceptance.py
docker compose ps -a
```

`ca-export` is expected to show `Exited (0)`. It is a successful one-shot job,
not a crashed server. The goja image is distroless, so the project validates
its readiness from `02-smoke.sh` through the public proxy instead of adding a
shell or HTTP client to the runtime image.

`03-browser-acceptance.py` uses independent cookie jars and the exported local
CA to exercise the complete HTTPS/OIDC behavior:

- Message Desk account creation without an invitation;
- goja account creation with a one-time TinyIDP signup invitation;
- preservation of an opaque application-invite continuation through OIDC;
- retryable rejection when that new password-only identity lacks a verified
  email claim;
- successful, atomic membership creation for the verified invitee fixture;
- rejection of both signup-invite and membership-invite replay; and
- tenant-queryable application audit plus TinyIDP issuance/redemption audit.

The script calls Docker Compose only for operator invitation issuance and
read-only database/audit assertions. Bearer codes are never written to disk.

## Trust the local CA in a browser

Caddy creates a development-only CA in its private data volume. The
`ca-export` job copies only the public root certificate to a separate volume;
the relying parties mount that public certificate read-only. The CA private key
never enters an application container.

`01-export-browser-ca.sh` copies the public root to:

```text
runtime/caddy-local-root.crt
```

Import that certificate as a trusted authority in the browser or operating
system you use for testing. On Debian/Ubuntu system trust, the explicit command
is:

```sh
sudo cp runtime/caddy-local-root.crt /usr/local/share/ca-certificates/tinyidp-local-caddy.crt
sudo update-ca-certificates
```

Firefox may use its own certificate store. Import the same public certificate
under Settings > Privacy & Security > Certificates > Authorities if Firefox
does not honor the operating-system store. Restart the browser after changing
trust. Installing a CA changes the workstation's trust policy, so the Compose
project never performs this step automatically.

To remove the Debian/Ubuntu trust entry later:

```sh
sudo rm /usr/local/share/ca-certificates/tinyidp-local-caddy.crt
sudo update-ca-certificates --fresh
```

## Why the CA export job exists

Caddy stores `root.crt` next to the local CA private key and deliberately uses
owner-only permissions. Mounting the entire Caddy data volume into an
application would both fail for a non-root/distroless process and expose
private signing material unnecessarily. The one-shot job implements a narrow
trust distribution contract:

```text
Caddy protected volume -- read public root --> ca-export
ca-export -- copy mode 0444 --> public trust volume
public trust volume -- read only --> TinyIDP health check, Message Desk, goja
```

Compose gates TinyIDP on `service_completed_successfully`; Message Desk and
goja then wait for TinyIDP readiness. This guarantees that TLS clients never
race the creation or publication of the local root.

## State, reset, and CA lifetime

Normal restarts preserve all named volumes and therefore preserve the same CA:

```sh
docker compose down
docker compose up -d
```

`docker compose down -v` destroys application databases **and** the local CA.
The next start creates a different root certificate, which must be exported and
trusted again. Do not use `-v` as a routine restart command.

The committed Postgres password is intentionally local and development-only.
The TinyIDP token secret is generated inside its owner-only state volume. No
production secret, Vault token, or live database is read by this project.

## Local go-go-goja iteration

This Phase 5 workspace intentionally builds `goja-auth` from the sibling
`../../../go-go-goja` checkout. Rebuild only that service after changing the
auth host or JavaScript routes:

```sh
docker compose up --build -d goja-auth
```

The generated host image is distroless. Inspect it through Compose logs and
the public readiness/acceptance scripts rather than adding debugging packages
to the runtime image.

## Logs and diagnosis

```sh
docker compose logs -f proxy idp message-desk goja-auth postgres
docker compose ps -a
./scripts/02-smoke.sh
```

The three application services should remain `Up`; Postgres, TinyIDP, and
Message Desk should become healthy. A direct visit to TinyIDP `/authorize`
without OAuth parameters is expected to return a protocol error. Use either
application's login or signup action so it supplies the registered client ID,
redirect URI, state, nonce, and PKCE challenge.
