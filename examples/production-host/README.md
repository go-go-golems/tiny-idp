# Production host example

`tinyidp serve-production` is the executable example for embedding tiny-idp in
a production-shaped process. Unlike `tinyidp serve`, it uses the public
`pkg/embeddedidp` API, durable SQLite, a synchronous fsync audit sink, bounded
login work, explicit proxy trust, scheduled retention, HTTPS, request limits,
HTTP timeouts, liveness/readiness endpoints, and graceful SIGINT/SIGTERM
shutdown.

Provision clients, users, credentials, and a signing key first:

```bash
go run ./cmd/tinyidp admin --db ./var/idp.db init --generate-signing-key --kid initial
go run ./cmd/tinyidp admin --db ./var/idp.db client create \
  --id example-spa --public \
  --redirect-uri https://app.example.test/callback \
  --scope openid --scope profile --scope email
printf '%s' 'a production password with enough length' | \
  go run ./cmd/tinyidp admin --db ./var/idp.db user create \
    --login alice --password-from-stdin
```

Create the host-only token secret and protect it. The host does not read this
secret from an environment variable or accept it directly on the command line.

```bash
mkdir -p ./var
openssl rand -out ./var/token-secret 32
chmod 600 ./var/token-secret
```

Use a certificate whose SAN covers the issuer hostname, then run:

```bash
go run ./cmd/tinyidp serve-production \
  --addr :8443 \
  --issuer https://idp.example.test:8443 \
  --db ./var/idp.db \
  --audit-path ./var/audit.jsonl \
  --token-secret-file ./var/token-secret \
  --tls-cert ./var/tls.crt \
  --tls-key ./var/tls.key
```

The liveness endpoint answers when the process is functioning. Readiness also
checks SQLite/schema access, active signing-key validity, token-secret policy,
audit delivery, rate limiting, and maintenance recency. A reverse proxy may
route only to a ready instance; it should restart on liveness failure, not on a
temporary readiness failure.

The supported SQLite topology is one active tiny-idp process, one local
filesystem database, and exactly one open database connection. Network
filesystems, shared-volume multi-writer deployments, and active/active replicas
are unsupported. Use verified online backups for recovery and a planned
single-writer failover procedure.
