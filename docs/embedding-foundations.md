# Embedding tiny-idp in a Go application

This guide describes the supported public composition boundary for a Go application that embeds tiny-idp. It covers browser applications today and the client/bootstrap preparation for a later device-authorization application. It does not claim that the strict embedded provider currently implements the device authorization grant.

The runnable development host is [`../examples/embedded/main.go`](../examples/embedded/main.go). Executable external-package examples are in [`../pkg/embeddedidp/example_test.go`](../pkg/embeddedidp/example_test.go). The production xapp composition is a larger reference in [`../cmd/tinyidp-xapp/state.go`](../cmd/tinyidp-xapp/state.go), [`../cmd/tinyidp-xapp/production_app.go`](../cmd/tinyidp-xapp/production_app.go), and [`../cmd/tinyidp-xapp/development_app.go`](../cmd/tinyidp-xapp/development_app.go).

## 1. Supported package boundary

An embedding application should use these packages:

| Package | Responsibility |
| --- | --- |
| `pkg/sqlitestore` | Persistent identity, credential, protocol, consent, session, client, and signing-key state |
| `pkg/idpstore` | Store interfaces and domain records |
| `pkg/idpaccounts` | Account creation, password replacement, password authentication, lockout, and password-work reporting |
| `pkg/embeddedidp` | Client/key bootstrap, provider construction, lifecycle, readiness, maintenance, and in-process issuer HTTP |
| `pkg/idp` | Password policy, audit, rate-limit, client-address, consent, readiness, and maintenance contracts |
| `pkg/idpui` | Optional login and consent renderer contract |

Application code must not import `internal/authn`, `internal/admin`, `internal/passwordhash`, `internal/keys`, or `internal/fositeadapter`. Those packages are implementation and operator-command details. The public account service deliberately does not expose the Argon2id hasher type. The public bootstrap deliberately does not expose private signing-key bytes.

The runtime ownership graph is:

```text
application composition root
    |
    +-- sqlitestore.Store ------------------------------+
    |                                                   |
    +-- idpaccounts.Service                             |
    |       +-- account creation                       |
    |       +-- password replacement                   |
    |       +-- password authentication                |
    |       +-- lockout and bounded password work      |
    |                                                   |
    +-- embeddedidp.Bootstrap                           |
    |       +-- browser/device client declarations     |
    |       +-- semantic drift detection               |
    |       +-- initial signing key                    |
    |                                                   |
    +-- embeddedidp.Provider <--------------------------+
    |       +-- authorize/token/userinfo/end-session
    |       +-- discovery/JWKS
    |       +-- login/consent interactions
    |       +-- readiness/maintenance
    |
    +-- InProcessIssuerTransport
            +-- exact issuer only
            +-- no network fallback
            +-- bounded handler response
```

## 2. Composition order

Build the identity subsystem in this order:

1. Open one persistent store.
2. Open the durable audit sink and other production controls.
3. Construct `idpaccounts.Service` from the store and audit sink.
4. Reconcile declared clients and the initial signing key with `embeddedidp.Bootstrap`.
5. Create or reconcile the first account through `idpaccounts.Service`.
6. Construct `embeddedidp.Provider` with the same store, account authenticator, and production controls.
7. Construct `embeddedidp.InProcessIssuerTransport` if the relying party shares the process.
8. Mount `provider.Handler()` on the host's `http.ServeMux`.
9. Schedule periodic `provider.RunMaintenance` calls in a host-owned lifecycle loop.
10. Close provider, audit sink, and store during graceful shutdown.

Do not construct a provider against an empty store and then mutate clients behind it. The Fosite client view is constructed during provider startup. Bootstrap prerequisites before calling `embeddedidp.New`.

Pseudocode:

```text
store = openSQLite(identityDatabase)
audit = openDurableAudit(auditLog)

accounts = idpaccounts.NewService(store, audit)
report = embeddedidp.Bootstrap(store, declaredClients, signingKeyID, audit)
reconcileInitialAccount(accounts)

provider = embeddedidp.New(
    issuer,
    production mode,
    store,
    accounts as authenticator,
    secure cookies,
    token secret,
    audit,
    rate limiter,
    client address resolver)

transport = embeddedidp.NewInProcessIssuerTransport(issuer, provider.Handler)
mount(provider.Handler)
scheduleMaintenance(provider)
```

## 3. Account service

Construct the public service with:

```go
accounts, err := idpaccounts.NewService(store, idpaccounts.Options{
    LoginPolicy:   idpaccounts.DefaultLoginPolicy(),
    PasswordPolicy: idp.DefaultPasswordAcceptancePolicy(),
    PasswordWork:   idp.PasswordWorkConfig{MaxConcurrent: 2},
    Audit:          auditSink,
})
```

The zero values select production-oriented defaults. `PasswordPolicy` controls establishment and normalization. `LoginPolicy` controls lockout and the development-only passwordless option. `PasswordWork` bounds concurrent Argon2id operations so unauthenticated login traffic cannot create unbounded memory pressure.

The service implements:

```go
idp.PasswordAuthenticator
idp.PasswordWorkReporter
idp.ProductionReadyReporter
```

This lets the same service establish accounts and satisfy `embeddedidp.Options.Authenticator`.

### 3.1 Create an account

```go
created, err := accounts.Create(ctx, idpaccounts.CreateRequest{
    Login:             "alice",
    Password:          passwordBytes,
    Email:             "alice@example.test",
    EmailVerified:     true,
    Name:              "Alice",
    PreferredUsername: "alice",
})
```

`Create` performs these operations:

```text
normalize login
validate password under the acceptance policy
choose or validate opaque user ID
choose subject = explicit subject or user ID
construct and validate profile
derive Argon2id credential under the bounded work semaphore
atomically create login + user + credential
emit identity.account.created after commit
```

Duplicate login or explicit ID returns `idpstore.ErrDuplicate`. Store errors fail closed. Password rejection wraps `idp.ErrPasswordRejected`.

If persistence succeeds and audit delivery fails, the method returns the committed user and an error wrapping `idp.ErrAuditDelivery`. An HTTP handler or CLI must not translate every non-nil error into “nothing changed.” It should reconcile by login or ID before retrying.

### 3.2 Replace a password

```go
err := accounts.SetPassword(ctx, idpaccounts.SetPasswordRequest{
    Login:    "alice",
    Password: replacement,
})
```

Replacement derives a new credential and atomically replaces the credential plus account security state. It emits `identity.account.password_changed` after commit. Account enable/disable remains an operational administrative action, not part of the public password service.

### 3.3 Authenticate

The provider normally calls:

```go
result, err := accounts.AuthenticatePassword(ctx, login, password, idp.LoginMetadata{
    ClientID:   clientID,
    RemoteAddr: remoteAddress,
    UserAgent:  userAgent,
})
```

Security-relevant outcomes include:

| Error | Meaning |
| --- | --- |
| `idpaccounts.ErrInvalidCredentials` | Unknown login, missing credential, or password mismatch |
| `idpaccounts.ErrAccountDisabled` | User or credential is disabled |
| `idpaccounts.ErrAccountLocked` | Durable lockout is active |
| `idpaccounts.ErrAuthenticationUnavailable` | Credential or security-state persistence failed |
| `idpaccounts.ErrPasswordWorkRejected` | Context ended while waiting for bounded password work |

Unknown users receive dummy Argon2 work. Authentication errors do not identify whether the login exists. Successful authentication resets durable failed-login state and may rehash a credential when parameters change.

## 4. Declarative bootstrap

Bootstrap reconciles clients and the first active signing key. It is not a migration system, a key-rotation command, or a drift repair operation.

```go
report, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{
    Mode: embeddedidp.ProductionMode,
    Clients: []embeddedidp.ClientSpec{
        embeddedidp.BrowserClient(
            "message-app",
            []string{"https://messages.example.test/auth/callback"},
            []string{"https://messages.example.test/"},
            []string{"openid", "profile", "email"},
        ),
    },
    SigningKeyID: "initial-rs256",
    Audit:        auditSink,
})
```

Bootstrap validates every declaration and duplicate ID before the first write. It orders clients by normalized ID. For each client it either creates the absent record or validates semantic equivalence with the existing record.

It normalizes declaration-only variation:

- surrounding whitespace and empty list entries;
- duplicate scopes and URIs;
- list order;
- zero token lifetimes to documented defaults;
- creation and update timestamps for comparison.

It compares every security-relevant stored field:

- public versus confidential status;
- secret hash bytes;
- redirect and post-logout redirect sets;
- scope set;
- PKCE requirement;
- access, ID, and refresh token lifetimes;
- disabled status.

Drift returns `*embeddedidp.ClientConflictError`, which unwraps to `embeddedidp.ErrBootstrapConflict`. `Fields` contains names such as `redirect_uris` and `allowed_scopes`, never secret contents.

### 4.1 Partial commit and retry

The store interface does not define one transaction spanning an arbitrary set of clients and a generated RSA key. Bootstrap therefore reports committed work:

```go
type BootstrapReport struct {
    ClientsCreated    []string
    ClientsValidated  []string
    SigningKeyCreated bool
    ActiveSigningKey  string
}
```

If a later client, key operation, or audit emission fails, the report describes earlier commits. Retrying the same declaration converges because created objects validate as equivalent.

Audit failure is also post-commit. Check `errors.Is(err, idp.ErrAuditDelivery)` and inspect the report before deciding how to recover.

### 4.2 Signing keys

If a valid active RS256 key exists, bootstrap retains it. If no active key exists, bootstrap generates and persists one 2048-bit RSA key. The report exposes only its identifier.

Bootstrap refuses to:

- overwrite a client;
- repair a corrupt or expired active key;
- rotate an existing active key;
- retire verification keys;
- return private key bytes.

Use the administrative key lifecycle for rotation and repair.

## 5. Browser client profile

`BrowserClient` creates a public authorization-code client:

```text
Public = true
RequirePKCE = true
RedirectURIs = exact caller declarations
PostLogoutRedirectURIs = exact caller declarations
AccessTokenTTL = 1 hour
IDTokenTTL = 1 hour
RefreshTokenTTL = 24 hours
```

At least one redirect URI and the `openid` scope are required. URI validation uses `idpstore.Client.Validate`. Production permits HTTPS redirects and loopback HTTP exceptions. Matching during authorization remains exact.

The browser sequence is:

```text
browser -> relying party: GET protected page
relying party -> browser: redirect to issuer /authorize with PKCE
browser -> embedded provider: login and consent interaction
embedded provider -> browser: exact callback with code
browser -> relying party: callback
relying party -> in-process transport: POST issuer /token
in-process transport -> provider handler: same request, no network
provider -> relying party: tokens
relying party -> browser: application session
```

## 6. Device client preparation and current gap

`DeviceClient` declares a public no-redirect client:

```go
embeddedidp.DeviceClient("example-cli", []string{"openid", "profile"})
```

The stored record has:

```text
Public = true
RequirePKCE = true
RedirectURIs = empty
PostLogoutRedirectURIs = empty
```

`RequirePKCE` preserves the current invariant that every stored public client sets the flag. It is dormant for a pure device grant, which has no authorization-code redirect.

Important limitation: the strict Fosite-backed embedded provider does not yet expose native `/device_authorization`, browser verification, approval, or device-code polling endpoints. Device grant code currently exists in the mock engine for local testing. Bootstrap support means a later implementation can declare the correct client shape without changing this API. It does not make the strict device flow usable by itself.

Before shipping the third example, implement and test:

- explicit per-client allowed grant capabilities;
- durable hashed device and user codes;
- verification URI and complete verification URI;
- polling interval, `authorization_pending`, and `slow_down` behavior;
- approval, denial, expiry, single consumption, and client binding;
- production audit and rate-limit events;
- discovery metadata for the device endpoint;
- conformance and adversarial tests.

The later sequence will be:

```text
device -> provider: POST /device_authorization
provider -> device: device_code + user_code + verification URI
user browser -> provider: open verification URI
user browser -> provider: authenticate and approve
device -> in-process or public HTTP: poll /token
provider -> device: token only after atomic approval and consumption
```

## 7. In-process issuer transport

Construct the transport only for a provider in the same process:

```go
transport, err := embeddedidp.NewInProcessIssuerTransport(
    issuer,
    provider.Handler(),
    embeddedidp.InProcessTransportOptions{
        MaxResponseBytes: embeddedidp.DefaultInProcessResponseLimit,
    },
)
oidcClient := &http.Client{Transport: transport, Timeout: 10 * time.Second}
```

The transport is intentionally not a general HTTP client. It has no fallback transport. It accepts only:

- an absolute request URL;
- the exact issuer scheme and host;
- the issuer path itself or a segment descendant;
- a canonical decoded path;
- no user information, fragment, opaque URL, backslash, encoded slash, encoded backslash, or encoded dot ambiguity.

These requests are distinct:

```text
issuer https://id.example.test/idp

accepted: https://id.example.test/idp
accepted: https://id.example.test/idp/token
rejected: https://id.example.test/idp-other
rejected: https://other.example.test/idp/token
rejected: http://id.example.test/idp/token
rejected: https://id.example.test/idp/%2e%2e/admin
rejected: https://id.example.test/idp%2ftoken
```

The response is buffered up to 1 MiB by default. A handler that attempts to write more causes `RoundTrip` to return an error even if the handler ignores the writer's short-write error. There is no partial successful response.

Cancellation is propagated through the cloned server request context. Because handler execution is synchronous, handlers must observe request cancellation as ordinary `net/http` handlers should.

Do not use this transport for:

- arbitrary application HTTP;
- third-party identity providers;
- fallback to public network traffic;
- streaming or unbounded responses;
- requests whose URL is chosen by an untrusted caller.

## 8. Production provider construction

`embeddedidp.Options.Validate` fails closed on the production contract. A typical composition supplies:

```go
provider, err := embeddedidp.New(ctx, embeddedidp.Options{
    Issuer:        "https://messages.example.test/idp",
    Mode:          embeddedidp.ProductionMode,
    Store:         store,
    Authenticator: accounts,
    Cookie: embeddedidp.CookieConfig{
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        Path:     "/idp",
    },
    Token:         embeddedidp.TokenConfig{SecretKey: tokenSecret},
    Audit:         auditSink,
    RateLimiter:   limiter,
    ClientAddress: resolver,
    PasswordPolicy: idp.DefaultPasswordAcceptancePolicy(),
    PasswordWork:   idp.PasswordWorkConfig{MaxConcurrent: 2},
})
```

Production validation requires:

- a canonical HTTPS issuer;
- secure supported cookie settings;
- a token secret of at least 32 bytes;
- a persistent store with the supported schema;
- maintenance support;
- valid configured clients;
- exactly one usable active RS256 signing key and valid verification keys;
- a durable ready audit reporter;
- production-ready rate limiting and client-address resolution;
- NIST-aligned password acceptance;
- bounded password work;
- a production-ready authenticator when supplied.

The provider handler speaks HTTP inside the process. Terminate public TLS at a correctly configured reverse proxy and preserve the issuer host and scheme contract.

## 9. Lifecycle and maintenance

The host owns startup and shutdown. It must not leak the store, audit file, or provider goroutines.

Recommended shutdown order:

```text
stop accepting public traffic
cancel maintenance context
wait for host goroutines
provider.Close
audit.Close
store.Close
```

Use `errgroup` for host goroutines. Avoid starting an unsupervised maintenance goroutine. Call `provider.Readiness(ctx)` during startup and expose the provider health endpoints through the issuer path.

## 10. Review checklist

Before releasing an embedding application, verify:

- [ ] application packages import only supported public identity packages;
- [ ] bootstrap runs before provider construction;
- [ ] every browser redirect is exact and intentional;
- [ ] the device profile is not presented as strict device-grant support;
- [ ] account creation handles post-commit audit errors;
- [ ] bootstrap handles partial reports and post-commit audit errors;
- [ ] production controls use durable implementations;
- [ ] token and binding secrets are generated, owner-only, and not logged;
- [ ] active and verification signing keys pass readiness;
- [ ] the in-process transport has no network fallback;
- [ ] the public listener has timeouts and graceful shutdown;
- [ ] maintenance is scheduled and monitored;
- [ ] focused, race, full, lint, static-analysis, and security checks pass;
- [ ] backup and restore procedures cover identity, audit, and application state consistently.

## 11. Verification commands

From the repository root:

```bash
go test ./pkg/idpaccounts ./pkg/embeddedidp ./cmd/tinyidp-xapp
go test -race ./pkg/idpaccounts ./pkg/embeddedidp ./cmd/tinyidp-xapp
go test ./...
go build ./...
make lint
make gosec
make auditlint
```

The repository-specific analyzer is not a replacement for `go vet`, `golangci-lint`, `gosec`, or dependency vulnerability scanning. It encodes tiny-idp-specific boundaries that general analyzers do not know.

## 12. Source map

Read these files in order when onboarding:

1. [`../pkg/idpaccounts/accounts.go`](../pkg/idpaccounts/accounts.go) — account mutation.
2. [`../pkg/idpaccounts/password.go`](../pkg/idpaccounts/password.go) — authentication, lockout, and bounded password work.
3. [`../pkg/idpstore/interfaces.go`](../pkg/idpstore/interfaces.go) — persistence and atomic invariants.
4. [`../pkg/embeddedidp/bootstrap.go`](../pkg/embeddedidp/bootstrap.go) — client/key reconciliation.
5. [`../pkg/embeddedidp/inprocess_transport.go`](../pkg/embeddedidp/inprocess_transport.go) — exact-issuer HTTP boundary.
6. [`../pkg/embeddedidp/options.go`](../pkg/embeddedidp/options.go) — production startup checks.
7. [`../pkg/embeddedidp/provider.go`](../pkg/embeddedidp/provider.go) — provider lifecycle, readiness, and maintenance.
8. [`../cmd/tinyidp-xapp/state.go`](../cmd/tinyidp-xapp/state.go) — durable initialization consumer.
9. [`../cmd/tinyidp-xapp/production_app.go`](../cmd/tinyidp-xapp/production_app.go) — complete production composition.
10. [`../pkg/embeddedidp/example_test.go`](../pkg/embeddedidp/example_test.go) — executable public-package examples.

The complete design rationale, rejected alternatives, phase plan, and implementation diary are under ticket `TINYIDP-EMBED-FOUND-001` in `ttmp/2026/07/13/`.
