# The strongest framing: an identity microkernel

This should not become “Keycloak, but smaller.” The differentiated product is an **identity microkernel**:

* Go owns protocols, cryptography, storage contracts, challenge state, replay protection, and token validation.
* JavaScript assembles those trusted primitives into an authentication graph.
* Small JavaScript functions handle application-specific routing, claims, risk decisions, and policy.
* Presets provide complete working configurations without requiring users to understand the graph.

That model fits the standards landscape. OpenID Connect separates authentication and identity claims from the underlying OAuth authorization machinery. WebAuthn provides relying-party-scoped public-key credentials. OAuth token exchange, Rich Authorization Requests, CIBA, and SPIFFE expose other naturally composable pieces: delegation, transaction details, out-of-band authentication, and workload identity. ([openid.net][1])

The main sweet spots would be:

* Embedded and application-local identity
* Ephemeral identity realms
* Application-specific claims and authorization
* Human and machine identity in one engine
* Protocol and migration bridges
* Authentication test infrastructure
* Transaction- and workflow-oriented authentication

A centralized workforce directory with extensive lifecycle management, hundreds of enterprise integrations, and a large delegated administration model remains better served by a full IdP.

---

# 1. Recommended execution model

Even in a “Goja-hosted” configuration, Go should remain the actual process host and trusted computing base. JavaScript can call `Auth.run()`, but that function should enter native Go code that starts the HTTP server, protocol handlers, storage, schedulers, and workers.

There are three useful deployment modes:

| Mode              | Shape                                        | Main use                                                        |
| ----------------- | -------------------------------------------- | --------------------------------------------------------------- |
| Embedded          | An `http.Handler`, middleware, or Go library | Single application, appliance, desktop backend                  |
| Sidecar           | Local HTTP or gRPC service                   | Several processes sharing one auth kernel                       |
| Script entrypoint | A JS file invokes native `run()`             | Low-code identity appliance or distributable auth configuration |

## Compile scripts into a graph

At startup:

1. Execute the configuration script.
2. Produce a serializable, immutable authentication graph.
3. Validate protocol configuration, redirect URIs, audiences, capability declarations, flow termination, and slot compatibility.
4. Run embedded policy tests.
5. Atomically activate the graph.

At request time, only explicitly registered lambdas should execute. Everything else should already be represented as native graph nodes.

## Design around Goja rather than pretending it is Node.js

Goja describes itself as an ES5.1-plus implementation, with much of ES6 still evolving. A runtime instance is not goroutine-safe, the embedding application must provide any event loop, and execution can be interrupted by the host. ([GitHub][2])

A practical model is therefore:

* One runtime per worker or a pool of single-owner runtimes
* The same script loaded independently into every runtime
* Short, synchronous request-time lambdas
* Native Go blocks for database, HTTP, email, cryptography, and other potentially blocking work
* Per-invocation execution deadlines using Goja interruption
* Bounded input and output sizes
* A process boundary for genuinely untrusted third-party scripts

The pseudocode below uses modern JavaScript syntax. A production framework should either publish a precise supported syntax profile or transpile scripts before Goja executes them.

---

# 2. Proposed fluent API

Assume the Go host injects a versioned global:

```js
const A = Auth.v1;
```

Avoid depending on Node-style module loading. Plugins can register against the injected API.

## Opinionated quick start

```js
const A = Auth.v1;

A.idp("notes")
  .use(A.preset.localWeb({
    origin: "https://notes.example",
    users: A.store.sqlite("./notes-auth.db"),

    client: A.client.sameOrigin(),

    login: "passkey-or-magic-link",

    sessions: {
      idle: "30m",
      absolute: "12h"
    }
  }))
  .mount("/auth")
  .validate("strict")
  .run();
```

`preset.localWeb()` would construct an entire graph:

* OIDC discovery, authorization, token, user-info, JWKS, and logout endpoints
* One first-party web client
* Authorization code flow
* User and session storage
* Login, registration, recovery, and consent UI
* Key rotation
* Rate limits and audit events
* Same-origin session cookie support

## Fully composed equivalent

```js
const A = Auth.v1;

const login = A.flow.login()
  .identify(
    A.identify.email({
      store: "users"
    })
  )

  .authenticate(
    A.choose(
      A.authn.passkey({
        userVerification: "preferred"
      }),

      A.authn.magicLink({
        send: A.notify.email("transactional-mail")
      }),

      A.seq(
        A.authn.password(),
        A.when(
          ctx => ctx.subject.mfaEnabled,
          A.authn.totp()
        )
      )
    )
  )

  .signals(
    A.parallel(
      A.signal.rateLimit({
        key: ctx => ctx.subject.id
      }),

      A.signal.custom("risk", ctx =>
        ctx.cap.risk.score({
          subject: ctx.subject.id,
          ip: ctx.request.ip,
          device: ctx.request.device
        })
      )
    )
  )

  .authorize(
    A.policy.decide(ctx => {
      if (
        ctx.signals.risk.score >= 80 &&
        !ctx.auth.amr.includes("passkey")
      ) {
        return A.decision.stepUp(
          A.authn.passkey({
            userVerification: "required"
          })
        );
      }

      return A.decision.allow();
    })
  )

  .claims(
    A.claims.merge(
      A.claims.oidcStandard(),

      A.claims.compute(ctx => {
        const membership = ctx.cap.appDB.membership(
          ctx.subject.id,
          ctx.tenant.id
        );

        return {
          tenant_id: ctx.tenant.id,
          roles: membership.roles,
          plan: membership.plan
        };
      })
    )
  )

  .issue(
    A.issue.oidc({
      accessTokenTTL: "10m",
      refreshTokenTTL: "30d"
    })
  );

A.idp("orders-auth")
  .issuer("https://orders.example/auth")

  .keys(
    A.keys.rotating(
      A.keys.handle("orders-signing-key"),
      {
        rotateEvery: "30d",
        verificationOverlap: "45d"
      }
    )
  )

  .store("users", A.store.sqlite("./orders-users.db"))

  .protocol(
    A.protocol.oidc({
      clients: [
        A.client.web("orders-web", {
          redirectURIs: [
            "https://orders.example/auth/callback"
          ]
        })
      ]
    })
  )

  .flow("login", login)

  .capabilities({
    stores: ["appDB"],
    services: ["risk.score", "transactional-mail.send"]
  })

  .hook(
    "login.after",
    A.effect(ctx =>
      ctx.cap.audit.emit("identity.login", {
        subject: ctx.subject.id,
        client: ctx.client.id,
        amr: ctx.auth.amr
      })
    )
  )

  .mount("/auth")
  .validate("strict")
  .run();
```

## Base block namespaces

| Namespace  | Representative blocks                                                                    |
| ---------- | ---------------------------------------------------------------------------------------- |
| `identify` | Session, email, username, invite, upstream subject, OAuth client, workload               |
| `authn`    | Password, passkey, magic link, TOTP, recovery code, upstream OIDC, mTLS, push approval   |
| `signal`   | Rate limits, device state, IP reputation, account state, geography, transaction risk     |
| `policy`   | RBAC, ABAC, scopes, consent, step-up, quorum, custom decision                            |
| `claims`   | OIDC standard claims, directory claims, computed claims, pairwise subject, minimization  |
| `issue`    | ID token, access token, refresh token, session, capability, X.509 identity, JWT identity |
| `protocol` | OIDC, OAuth device flow, CIBA, token exchange, introspection, revocation, JWKS           |
| `store`    | Memory, SQLite, PostgreSQL, encrypted file, remote store, application adapter            |
| `notify`   | Email, SMS, push, webhook                                                                |
| `audit`    | Structured events, metrics, traces, immutable receipts                                   |
| `attest`   | Kubernetes workload, Unix process, cloud instance, TPM-backed device                     |
| `bridge`   | Trusted headers, legacy sessions, API keys, upstream token validation                    |

## Block algebra

Every block should return one of five outcomes:

```js
A.outcome.ok(value, evidence);
A.outcome.challenge(view, continuation);
A.outcome.deny(code);
A.outcome.skip(reason);
A.outcome.error(code, { retryable: false });
```

The distinction is important:

* `ok` continues the flow.
* `challenge` suspends it. Go persists an opaque continuation, not JavaScript.
* `deny` is terminal.
* `skip` means the block is not applicable, allowing another branch.
* `error` is terminal unless an explicit recovery policy handles it.

The composition operators can then have unambiguous security semantics:

```js
A.seq(a, b, c);                 // Run in order
A.all(a, b, c);                 // All must succeed
A.choose(a, b, c);              // Present or select valid alternatives
A.firstAvailable(a, b, c);      // Continue only when a block returns skip
A.when(predicate, yes, no);     // Conditional branch
A.switch(selector, cases);      // Value-based routing
A.map(block, mapper);           // Transform successful output
A.tap(block, effect);           // Observe without changing the result
A.recover(block, rules);        // Explicit infrastructure recovery
A.timeout(block, "100ms");      // Execution budget
A.cache(block, options);        // Cache safe deterministic results
```

`choose()` must not silently try another authentication factor after a credential was rejected. A rejection is `deny`, not `skip`.

## Presets as patchable named graphs

Presets should not be opaque special cases. They should return normal graphs with stable, named slots.

```js
A.idp("shop")
  .use(A.preset.localWeb({
    origin: "https://shop.example",
    users: A.store.sqlite("./users.db"),
    client: A.client.sameOrigin()
  }))

  .replace(
    A.slot.login.authenticate,
    A.choose(
      A.authn.passkey(),
      A.authn.magicLink({
        send: A.notify.email("mail")
      })
    )
  )

  .append(
    A.slot.token.claims,
    A.claims.compute(ctx => ({
      customer_tier: ctx.cap.shop.customerTier(ctx.subject.id)
    }))
  )

  .wrap(
    A.slot.issue.accessToken,
    inner => A.issue.senderConstrained(inner, {
      method: "dpop"
    })
  )

  .run();
```

Sender-constrained OAuth tokens are supported through mechanisms such as mTLS and DPoP; these should be native issuance and validation wrappers rather than JavaScript implementations. ([RFC Editor][3])

## JavaScript plugin format

A plugin can register individual blocks, hooks, or complete presets.

```js
A.registerPlugin({
  name: "adaptive-risk-login",
  version: "1.2.0",
  apiVersion: 1,

  permissions: [
    "service:risk.score"
  ],

  configSchema: {
    threshold: "number",
    highAssuranceFactor: "string"
  },

  install(registry) {
    registry.block("signal.accountRisk", options =>
      A.signal.custom("accountRisk", ctx =>
        ctx.cap.risk.score({
          subject: ctx.subject.id,
          client: ctx.client.id,
          ip: ctx.request.ip,
          weights: options.weights
        })
      )
    );

    registry.preset("login.adaptive", options =>
      A.preset.compose(
        A.authn.passkey(),

        A.policy.decide(ctx =>
          ctx.signals.accountRisk.score >= options.threshold
            ? A.decision.stepUp(
                A.authn.named(options.highAssuranceFactor)
              )
            : A.decision.allow()
        )
      )
    );
  }
});
```

A plugin manifest should declare:

* API and block ABI version
* Configuration schema
* Required native blocks
* Required capabilities
* Whether hooks are pure or effectful
* Maximum execution time
* Whether the plugin is permitted in token issuance paths
* A content hash or signature for controlled deployments

## Security boundaries that should not be scriptable

JavaScript should never directly:

* Parse or validate JWT signatures
* Select arbitrary cryptographic algorithms
* Read signing key bytes
* Read password hashes
* Construct authorization codes
* Validate redirect URIs
* Implement PKCE, nonce, state, replay, or refresh-token rotation
* Mutate raw OAuth requests after native validation
* Decide whether an invalid signature should be ignored
* Access the filesystem or network without an explicit capability

Scripts should return structured decisions. Native Go code should enforce protocol invariants regardless of what the script asks for.

---

# 3. Examples, from ordinary to experimental

## Simple applications

### 1. Drop-in authentication for one web application

Runs authentication alongside the application without provisioning a separate identity service.

Use cases include self-hosted dashboards, internal utilities, appliances, and small single-tenant SaaS products.

```js
const A = Auth.v1;

A.idp("wiki")
  .use(A.preset.localWeb({
    origin: "https://wiki.example",
    users: A.store.sqlite("./wiki-auth.db"),
    client: A.client.sameOrigin(),
    login: "password-plus-totp"
  }))
  .mount("/auth")
  .run();
```

---

### 2. Authenticate against the application’s existing user table

The application remains the system of record. The auth kernel supplies secure protocol and authentication behavior without duplicating accounts.

```js
const A = Auth.v1;

const appUsers = A.store.adapter({
  findByLogin: (ctx, login) =>
    ctx.cap.appDB.userByEmail(login),

  loadSubject: (ctx, id) =>
    ctx.cap.appDB.userByID(id),

  verifyPassword: (ctx, id, candidate) =>
    ctx.cap.passwordVerifier.verify(id, candidate),

  updateLoginMetadata: (ctx, id, metadata) =>
    ctx.cap.appDB.recordLogin(id, metadata)
});

A.idp("shop")
  .store("users", appUsers)
  .use(A.preset.localWeb({
    origin: "https://shop.example",
    users: "users",
    client: A.client.sameOrigin()
  }))
  .run();
```

The JavaScript adapter never receives password hashes. It invokes an opaque native verifier.

---

### 3. Passkey-first passwordless authentication

Use passkeys as the primary factor, with magic-link or recovery-code fallback.

WebAuthn credentials are public-key credentials scoped to a relying party, making this a strong native block for an embedded IdP. ([W3C][4])

```js
const A = Auth.v1;

A.idp("admin-console")
  .use(A.preset.localWeb({
    origin: "https://admin.example",
    users: A.store.sqlite("./admin-users.db"),
    client: A.client.sameOrigin()
  }))

  .replace(
    A.slot.login.authenticate,
    A.choose(
      A.authn.passkey({
        discoverable: true,
        userVerification: "required"
      }),

      A.authn.magicLink({
        send: A.notify.email("security-mail"),
        ttl: "10m"
      }),

      A.authn.recoveryCode()
    )
  )

  .run();
```

---

### 4. Application-domain claims and authorization

The identity layer can understand the application’s own memberships instead of forcing all authorization data into a central directory.

```js
const A = Auth.v1;

A.idp("projects")
  .use(A.preset.localWeb({
    origin: "https://projects.example",
    users: A.store.sqlite("./users.db"),
    client: A.client.sameOrigin()
  }))

  .append(
    A.slot.login.authorize,
    A.policy.decide(ctx => {
      const membership = ctx.cap.projects.membership(
        ctx.subject.id,
        ctx.request.projectID
      );

      return membership
        ? A.decision.allow()
        : A.decision.deny("not_a_project_member");
    })
  )

  .append(
    A.slot.token.claims,
    A.claims.compute(ctx => {
      const membership = ctx.cap.projects.membership(
        ctx.subject.id,
        ctx.request.projectID
      );

      return {
        project_id: ctx.request.projectID,
        project_role: membership.role
      };
    })
  )

  .run();
```

---

### 5. Deterministic development and test IdP

A complete identity simulator can be embedded in integration tests, SDK test suites, local development, and CI.

```js
const A = Auth.v1;

A.idp("sdk-test-idp")
  .use(A.preset.testLab({
    issuer: "http://127.0.0.1:7777",

    clock: A.clock.fixed("2030-01-01T00:00:00Z"),
    keys: A.keys.deterministic("test-only-seed"),

    users: [
      {
        id: "alice",
        email: "alice@example.test",
        roles: ["admin"],
        factors: ["password", "totp"]
      },
      {
        id: "bob",
        email: "bob@example.test",
        roles: ["viewer"],
        factors: ["passkey"]
      }
    ],

    faults: {
      "token.once": "temporarily_unavailable",
      "refresh.after:3": "invalid_grant"
    }
  }))
  .run();
```

A strict production build should reject `testLab`, deterministic keys, and fixed clocks.

---

## More interesting application and platform uses

### 6. Resource- or transaction-driven step-up

A normal login may be sufficient for browsing, while exporting all customer data or transferring money requires recent high-assurance authentication.

RFC 9470 defines an interoperable way for a resource server to request stronger or more recent authentication. ([RFC Editor][5])

```js
const A = Auth.v1;

const sensitiveActionPolicy = A.policy.decide(ctx => {
  const tx = ctx.authorizationDetails;

  const requiresStepUp =
    tx.type === "payment" &&
    tx.amount > 1000 &&
    (
      ctx.auth.ageSeconds > 300 ||
      !ctx.auth.amr.includes("passkey")
    );

  if (requiresStepUp) {
    return A.decision.stepUp(
      A.authn.passkey({
        userVerification: "required"
      }),
      {
        acr: "urn:example:high",
        maxAge: "5m"
      }
    );
  }

  return A.decision.allow();
});

A.idp("payments")
  .use(A.preset.localWeb({
    origin: "https://payments.example",
    users: A.store.sqlite("./users.db"),
    client: A.client.sameOrigin()
  }))
  .append(A.slot.login.authorize, sensitiveActionPolicy)
  .run();
```

---

### 7. Multi-tenant white-label identity

Resolve the tenant from the hostname or client, then provide tenant-specific issuer configuration, keys, UI, upstream IdP, and claim mapping.

```js
const A = Auth.v1;

A.idp("b2b-platform")
  .use(A.preset.multiTenant({
    resolve: ctx => ctx.request.host,

    load: (ctx, host) =>
      ctx.cap.tenants.byHost(host),

    configure: tenant => ({
      issuer: `https://${tenant.host}/auth`,

      keys: A.keys.handle(tenant.signingKeyRef),

      users: A.store.named(tenant.userStoreRef),

      authenticate: tenant.upstream
        ? A.authn.upstreamOIDC(tenant.upstream)
        : A.choose(
            A.authn.passkey(),
            A.authn.magicLink({
              send: A.notify.email(tenant.mailerRef)
            })
          ),

      claims: A.claims.compute(ctx => ({
        tenant_id: tenant.id,
        tenant_roles: ctx.cap.tenants.roles(
          tenant.id,
          ctx.subject.id
        )
      })),

      theme: tenant.theme
    })
  }))
  .run();
```

The tenant graph can be cached and rebuilt atomically when tenant configuration changes.

---

### 8. Small identity broker with claim normalization

Route users to different upstream providers and present one stable identity contract to the application.

Use cases include B2B SaaS, acquisitions with multiple identity systems, and a gradual move from social login to enterprise SSO.

```js
const A = Auth.v1;

const upstreamLogin = A.flow.login()
  .identify(A.identify.email())

  .authenticate(
    A.switch(
      ctx => ctx.identity.emailDomain,
      {
        "acme.example":
          A.authn.upstreamOIDC("acme-workforce"),

        "partner.example":
          A.authn.upstreamOIDC("partner-idp"),

        default:
          A.choose(
            A.authn.upstreamOIDC("consumer-login"),
            A.authn.passkey()
          )
      }
    )
  )

  .claims(
    A.claims.mapUpstream(ctx => ({
      email: ctx.upstream.email,
      name: ctx.upstream.displayName,
      groups: normalizeGroups(ctx.upstream),
      source_idp: ctx.upstream.issuerID
    }))
  )

  .issue(A.issue.oidc());

A.idp("identity-broker")
  .protocol(A.protocol.oidc({
    clients: A.clientStore("clients")
  }))
  .flow("login", upstreamLogin)
  .run();
```

A separate, explicitly enabled local break-glass flow is safer than automatically falling back after an upstream authentication failure.

---

### 9. CLI, television, and constrained-device login

The OAuth device authorization grant allows a constrained client to show a code while the user completes authentication on another device. ([RFC Editor][6])

```js
const A = Auth.v1;

A.idp("developer-cli")
  .protocol(
    A.protocol.deviceAuthorization({
      clients: [
        A.client.public("acme-cli")
      ],

      userCode: {
        alphabet: "BCDFGHJKLMNPQRSTVWXYZ23456789",
        length: 8,
        ttl: "10m"
      },

      polling: {
        initialInterval: "5s",
        rateLimit: true
      }
    })
  )

  .flow(
    "device-approval",
    A.flow.deviceApproval()
      .authenticate(A.authn.passkey())
      .authorize(
        A.policy.consent({
          describe: ctx => ({
            application: ctx.client.displayName,
            requestedScopes: ctx.request.scopes
          })
        })
      )
      .issue(
        A.issue.accessToken({
          audience: "developer-api",
          ttl: "30m"
        })
      )
  )

  .run();
```

---

### 10. Out-of-band approval from another device

A kiosk, call-center terminal, checkout, or physical device initiates a request. The user authenticates and approves it on a trusted phone.

CIBA defines this general model: the client initiates authentication directly with the provider, and the user interaction occurs out of band. ([openid.net][7])

```js
const A = Auth.v1;

A.idp("terminal-approval")
  .protocol(
    A.protocol.ciba({
      delivery: "ping",
      clients: A.clientStore("terminals")
    })
  )

  .flow(
    "backchannel-login",
    A.flow.backchannel()
      .identify(A.identify.loginHint({
        store: "users"
      }))

      .authenticate(
        A.authn.pushApproval({
          channel: "mobile-authenticator",

          bindingMessage: ctx =>
            ctx.request.terminalCode,

          display: ctx => ({
            title: "Approve terminal sign-in",
            terminal: ctx.client.displayName,
            location: ctx.request.location
          })
        })
      )

      .authorize(A.policy.consent())
      .issue(A.issue.oidc())
  )

  .run();
```

---

### 11. Per-preview-environment authentication

Create a short-lived realm for each pull request or preview deployment. Only project members can enter, and the realm disappears with the environment.

```js
const A = Auth.v1;

A.idp(`preview-${env.PREVIEW_ID}`)
  .use(A.preset.preview({
    origin: env.PREVIEW_URL,
    expiresAt: env.PREVIEW_EXPIRES_AT,

    upstream: A.authn.upstreamOIDC("company-workforce"),

    allow: ctx =>
      ctx.subject.groups.includes("engineering") ||
      ctx.subject.groups.includes(
        `project:${env.PROJECT_ID}`
      ),

    client: A.client.sameOrigin(),

    keys: A.keys.ephemeral({
      destroyAt: env.PREVIEW_EXPIRES_AT
    })
  }))
  .run();
```

This is useful for review applications, temporary customer demos, training environments, and isolated security testing.

---

### 12. Legacy-authentication strangler

A trusted legacy gateway provides an authenticated identity. The new kernel converts that identity into modern sessions and tokens while applications migrate incrementally.

```js
const A = Auth.v1;

const legacyLogin = A.flow.login()
  .identify(
    A.identify.trustedHeader({
      userHeader: "X-Legacy-User",
      groupsHeader: "X-Legacy-Groups",

      requireTransport: A.transport.mtls({
        peer: "legacy-auth-gateway"
      }),

      signatureHeader: "X-Legacy-Identity-Signature",
      signatureKey: A.keys.handle("legacy-header-key")
    })
  )

  .authorize(
    A.policy.rule(
      "known-legacy-account",
      ctx => ctx.cap.directory.exists(ctx.subject.id)
    )
  )

  .claims(
    A.claims.compute(ctx => ({
      migration_source: "legacy",
      groups: ctx.identity.groups
    }))
  )

  .issue(A.issue.oidc());

A.idp("migration-bridge")
  .protocol(A.protocol.oidc({
    clients: A.clientStore("new-apps")
  }))
  .flow("legacy-login", legacyLogin)
  .run();
```

A raw internet-facing identity header must never be trusted without authenticated transport and integrity protection.

---

## Distributed-system and edge uses

### 13. Token exchange and downscoped delegation

A frontend token should not necessarily be forwarded unchanged through every backend. A service can exchange it for a shorter-lived token with a narrower audience and reduced scopes.

RFC 8693 explicitly supports an authorization server acting as a security token service, including exchanges for tokens intended for downstream services. ([RFC Editor][8])

```js
const A = Auth.v1;

A.idp("service-sts")
  .protocol(A.protocol.tokenExchange())

  .flow(
    "exchange",
    A.flow.tokenExchange()
      .authenticate(A.authn.subjectToken())

      .authorize(
        A.policy.decide(ctx => {
          const allowed = ctx.cap.delegation.canDelegate({
            subject: ctx.subject.id,
            actor: ctx.client.id,
            target: ctx.request.resource,
            scopes: ctx.request.scopes
          });

          return allowed
            ? A.decision.allow()
            : A.decision.deny("delegation_not_allowed");
        })
      )

      .issue(
        A.issue.accessToken({
          audience: ctx => ctx.request.resource,

          scopes: ctx =>
            intersection(
              ctx.subjectToken.scopes,
              ctx.request.scopes,
              ctx.cap.delegation.maxScopes(ctx.client.id)
            ),

          ttl: "2m",

          actor: ctx => ({
            client_id: ctx.client.id
          })
        })
      )
  )

  .run();
```

---

### 14. Tiny workload-identity issuer

Attest a pod, Unix process, cloud instance, or device and issue short-lived machine identity.

SPIFFE uses URI-form workload identifiers within a trust domain and supports X.509 and JWT identity documents. ([spiffe.io][9])

```js
const A = Auth.v1;

A.idp("workload-trust")
  .use(A.preset.workloadTrust({
    trustDomain: "acme.internal",

    attest: A.choose(
      A.attest.kubernetes({
        cluster: "production"
      }),

      A.attest.unixProcess({
        allowedExecutables: [
          "/opt/acme/bin/worker"
        ]
      })
    ),

    identity: ctx =>
      `spiffe://acme.internal/${ctx.attestation.namespace}/${ctx.attestation.service}`,

    issue: [
      A.issue.x509SVID({
        ttl: "10m"
      }),

      A.issue.jwtSVID({
        ttl: "2m",
        requireAudience: true
      })
    ]
  }))
  .run();
```

This would not need to replace a large SPIFFE deployment. It could serve appliances, small clusters, development environments, or applications needing one local workload issuer.

---

### 15. Offline edge identity realm

Run a local realm in a factory, vessel, store, vehicle, or field installation where connectivity is intermittent.

```js
const A = Auth.v1;

A.idp("factory-edge")
  .use(A.preset.edgeRealm({
    root: A.keys.tpm("edge-device-root"),

    upstream: {
      issuer: "central-identity",
      sync: [
        "authorized-users",
        "device-inventory",
        "revocations"
      ]
    },

    offline: {
      maximumDuration: "72h",
      failClosedAfter: "72h",
      allowNewEnrollment: false
    },

    localTokens: {
      audiences: [
        "factory-control",
        "factory-observability"
      ],
      ttl: "10m"
    },

    authenticate: A.choose(
      A.authn.passkey(),
      A.authn.badge({
        requireLocalPIN: true
      })
    )
  }))
  .run();
```

The local issuer should use short token lifetimes, bounded offline operation, and explicit behavior when revocation information becomes stale.

---

### 16. Transaction-bound authorization

Instead of asking for a broad `payments.write` scope, ask for permission to perform one specific operation.

Rich Authorization Requests define structured JSON authorization details for cases such as a particular payment or file operation. ([RFC Editor][10])

```js
const A = Auth.v1;

A.idp("transaction-auth")
  .protocol(
    A.protocol.richAuthorization({
      types: {
        payment: A.schema.object({
          creditor: A.schema.string(),
          amount: A.schema.money(),
          reference: A.schema.string()
        })
      }
    })
  )

  .append(
    A.slot.login.authorize,
    A.policy.decide(ctx => {
      const payment = ctx.authorizationDetails;

      if (payment.type !== "payment") {
        return A.decision.deny("unsupported_transaction");
      }

      if (payment.amount.value > 5000) {
        return A.decision.stepUp(
          A.authn.passkey({
            userVerification: "required"
          })
        );
      }

      return A.decision.consent({
        display: {
          creditor: payment.creditor,
          amount: payment.amount,
          reference: payment.reference
        }
      });
    })
  )

  .append(
    A.slot.token.claims,
    A.claims.authorizationDetails()
  )

  .run();
```

---

### 17. Privacy-preserving personas per client

Give each relying party a different subject identifier and release only the claims it requires.

OIDC pairwise identifiers are intended to prevent clients from correlating a user across relying parties without permission. ([openid.net][1])

```js
const A = Auth.v1;

A.idp("plugin-ecosystem")
  .use(A.preset.oidcProvider({
    clients: A.clientStore("plugins")
  }))

  .append(
    A.slot.token.claims,
    A.claims.compose(
      A.claims.pairwiseSubject({
        sector: ctx => ctx.client.sectorID,
        key: A.keys.handle("pairwise-subject-key")
      }),

      A.claims.minimize({
        release: (ctx, requested) =>
          requested.filter(claim =>
            ctx.client.allowedClaims.includes(claim)
          )
      }),

      A.claims.compute(ctx => ({
        persona: ctx.cap.personas.forClient(
          ctx.subject.id,
          ctx.client.id
        )
      }))
    )
  )

  .run();
```

---

### 18. Data residency and key routing per tenant

The policy graph can route storage and signing operations to opaque regional capabilities without exposing key material to JavaScript.

```js
const A = Auth.v1;

A.idp("regional-saas")
  .tenant(
    A.tenant.resolve(ctx =>
      ctx.cap.tenants.resolve(
        ctx.request.host,
        ctx.client.id
      )
    )
  )

  .store(
    "users",
    A.store.route(
      ctx => ctx.tenant.region,
      {
        eu: A.store.named("eu-user-store"),
        us: A.store.named("us-user-store"),
        apac: A.store.named("apac-user-store")
      }
    )
  )

  .keys(
    A.keys.route(
      ctx => ctx.tenant.region,
      {
        eu: A.keys.handle("eu-signing-key"),
        us: A.keys.handle("us-signing-key"),
        apac: A.keys.handle("apac-signing-key")
      }
    )
  )

  .use(A.preset.localWeb({
    users: "users",
    client: A.client.fromTenant()
  }))

  .run();
```

---

## More novel uses

### 19. Ephemeral identity realm for an incident

An incident response system creates a realm for the lifetime of an incident. Humans, bots, temporary vendors, and emergency tooling receive narrowly scoped identities that automatically expire.

```js
const A = Auth.v1;

A.idp(`incident-${env.INCIDENT_ID}`)
  .use(A.preset.ephemeralRealm({
    createdAt: env.INCIDENT_STARTED_AT,
    expiresAfter: "12h",

    bootstrap: A.enrollment.signedInvite({
      signer: A.keys.handle("incident-controller"),
      audiences: ["incident-room"]
    }),

    subjects: [
      A.subject.human(),
      A.subject.workload(),
      A.subject.automation()
    ],

    defaultScopes: [
      "incident:read",
      "timeline:append"
    ],

    privilegedScopes: {
      "production:change":
        A.authn.quorum(2, [
          A.authn.approvalByRole("incident-commander"),
          A.authn.approvalByRole("service-owner"),
          A.authn.approvalByRole("security")
        ])
    },

    destroyOnExpiry: true
  }))
  .run();
```

The realm itself becomes a short-lived security boundary.

---

### 20. Human-to-AI-agent delegation

A human authorizes an agent to perform one bounded task rather than handing it a broad personal access token.

```js
const A = Auth.v1;

A.idp("agent-delegation")
  .protocol(A.protocol.tokenExchange())

  .flow(
    "delegate-to-agent",
    A.flow.tokenExchange()
      .authenticate(
        A.all(
          A.authn.subjectToken(),
          A.authn.dpopProof()
        )
      )

      .authorize(
        A.policy.decide(ctx => {
          const task = ctx.request.authorizationDetails;

          if (task.type !== "agent_task") {
            return A.decision.deny("invalid_task");
          }

          if (task.maximumCost > 50) {
            return A.decision.stepUp(
              A.authn.passkey({
                userVerification: "required"
              })
            );
          }

          return A.decision.consent({
            display: {
              agent: ctx.client.displayName,
              objective: task.objective,
              resources: task.resources,
              maximumCost: task.maximumCost,
              expiresIn: task.expiresIn
            }
          });
        })
      )

      .issue(
        A.issue.capability({
          audience: ctx =>
            ctx.request.authorizationDetails.resources,

          actions: ctx =>
            ctx.request.authorizationDetails.actions,

          constraints: ctx => ({
            objectiveHash:
              ctx.request.authorizationDetails.objectiveHash,

            maximumCost:
              ctx.request.authorizationDetails.maximumCost
          }),

          confirmation: A.confirmation.dpop(),
          ttl: "15m"
        })
      )
  )

  .run();
```

This combines delegation, transaction details, short lifetimes, explicit user consent, and proof-of-possession rather than relying on a reusable bearer credential. ([RFC Editor][8])

---

### 21. Multi-party or quorum authorization

Require two or more distinct identities to approve a high-impact action. Each approval is bound to the exact transaction, not merely to a generic session.

```js
const A = Auth.v1;

const productionChange = A.policy.quorum({
  required: 2,

  candidates: [
    A.approver.role("service-owner"),
    A.approver.role("security"),
    A.approver.role("on-call-operator")
  ],

  distinctSubjects: true,

  bindTo: ctx =>
    A.hash.canonical({
      service: ctx.authorizationDetails.service,
      version: ctx.authorizationDetails.version,
      environment: "production",
      changeID: ctx.authorizationDetails.changeID
    }),

  authenticateApprover: A.authn.passkey({
    userVerification: "required"
  }),

  expiresAfter: "20m"
});

A.idp("production-control")
  .use(A.preset.transactionAuthorization())
  .policy("production-change", productionChange)
  .run();
```

Use cases include production deploys, large refunds, cryptographic key operations, destructive administrative actions, and release approvals.

---

### 22. Object- and action-bound capability mint

Issue a credential that is useful for exactly one object and action, rather than giving the holder a general session or broad API token.

```js
const A = Auth.v1;

A.idp("document-capabilities")
  .flow(
    "grant-document-access",
    A.flow.authorization()
      .authenticate(A.authn.existingSession())

      .authorize(
        A.policy.decide(ctx =>
          ctx.cap.documents.canRead(
            ctx.subject.id,
            ctx.request.documentID
          )
            ? A.decision.allow()
            : A.decision.deny("document_access_denied")
        )
      )

      .issue(
        A.issue.capability({
          audience: "document-service",

          resource: ctx =>
            `document:${ctx.request.documentID}`,

          actions: ["read"],

          constraints: {
            maximumReads: 1
          },

          confirmation: A.confirmation.dpop(),
          ttl: "2m"
        })
      )
  )

  .run();
```

This is useful for document access, one-time exports, upload authorization, recovery actions, and delegated tool calls.

---

### 23. Identity-aware workflow engine

Generalize authentication into a resumable state machine involving several actors, devices, or channels.

```js
const A = Auth.v1;

A.idp("release-workflow")
  .flow(
    "production-release",
    A.flow.transaction("production-release")

      .stage(
        "requester",
        A.all(
          A.authn.existingSession(),
          A.policy.role("release-engineer")
        )
      )

      .stage(
        "service-owner",
        A.authn.approval({
          role: "service-owner",
          channel: "mobile-or-web"
        })
      )

      .stage(
        "security-review",
        A.when(
          ctx => ctx.transaction.risk >= 70,
          A.authn.approval({
            role: "security",
            require: A.authn.passkey({
              userVerification: "required"
            })
          })
        )
      )

      .stage(
        "execution-agent",
        A.authn.workload({
          audience: "deployment-controller"
        })
      )

      .issue(
        A.issue.capability({
          audience: "deployment-controller",
          resource: ctx =>
            `release:${ctx.transaction.releaseID}`,
          actions: ["deploy"],
          ttl: "10m"
        })
      )
  )

  .run();
```

The same primitive can model account recovery, insurance claims, procurement, regulated approvals, and human-in-the-loop automation.

---

### 24. Portable self-issued identity capsule

A desktop or mobile application can run a personal issuer controlled by the user and present different identifiers to different applications.

This is experimental. In a self-issued model, claims directly asserted by the user are not automatically trustworthy as third-party-attested attributes. ([openid.net][11])

```js
const A = Auth.v1;

A.idp("personal-identity")
  .use(A.preset.selfIssued({
    keys: A.keys.secureElement("personal-identity-key"),

    subject: A.subject.keyControlled(),

    pairwise: {
      enabled: true,
      derivePerClient: true
    },

    authenticateUser: A.authn.deviceBiometric(),

    disclose: A.claims.userSelected({
      default: [],
      neverRelease: [
        "recovery_material",
        "device_inventory"
      ]
    }),

    externallyAttestedClaims:
      A.claims.credentials({
        trustedIssuers:
          A.trustStore("credential-issuers")
      })
  }))

  .run();
```

Possible uses include local-first applications, personal developer identity, peer-to-peer tools, and devices that should not depend continuously on a cloud identity provider.

---

### 25. Ad hoc micro-federation

Two applications or organizations establish temporary, tightly constrained trust through a signed invitation rather than permanent federation configuration.

```js
const A = Auth.v1;

A.idp("temporary-collaboration")
  .federation(
    A.federation.invitation({
      verifyWith: A.trustStore("partner-roots"),

      maximumLifetime: "24h",

      require: invitation =>
        invitation.allowedClients.length <= 5 &&
        invitation.allowedScopes.every(scope =>
          [
            "workspace:read",
            "workspace:comment"
          ].includes(scope)
        ),

      mapSubject: ctx => ({
        external_issuer: ctx.upstream.issuer,
        external_subject: ctx.upstream.subject,
        collaboration_id: ctx.invitation.id
      }),

      issue: A.issue.accessToken({
        audience: "temporary-workspace",
        ttl: "15m"
      })
    })
  )

  .run();
```

Use cases include temporary vendor access, incident collaboration, local device clusters, conferences, workshops, and short-lived inter-company projects.

---

### 26. Authentication-policy digital twin

Run a candidate policy against real, redacted requests without affecting production decisions. Compare authorization results, claims, authentication requirements, and token lifetimes.

```js
const A = Auth.v1;

const currentPolicy = A.policy.bundle("access-v1");
const candidatePolicy = A.policy.bundle("access-v2");

A.idp("production-auth")
  .use(A.preset.localWeb({
    origin: "https://app.example",
    users: A.store.named("production-users"),
    client: A.client.sameOrigin()
  }))

  .policy("access", currentPolicy)

  .shadowPolicy("access-v2", {
    candidate: candidatePolicy,

    sample: 0.10,

    redact: ctx => ({
      client: ctx.client.id,
      subjectClass: ctx.subject.accountClass,
      scopes: ctx.request.scopes,
      signalBuckets: bucketSignals(ctx.signals),
      transactionType:
        ctx.authorizationDetails?.type
    }),

    compare: [
      "decision",
      "required_acr",
      "issued_scopes",
      "claim_names",
      "token_ttl"
    ],

    onDifference: diff =>
      A.audit.emit("policy.shadow-difference", diff)
  })

  .run();
```

This makes policy changes testable in the same way application code is tested, without exposing a policy explanation endpoint to attackers.

---

# 4. Features that would make the framework unusually useful

## Authentication graph inspection

Provide development-only tools such as:

```js
const result = app.explain({
  client: "orders-web",
  subject: "alice",
  action: "refund",
  amount: 2000
});

print(result.path);
print(result.requiredFactors);
print(result.issuedClaims);
```

Production explanations should be redacted and access-controlled.

## Policy tests inside the script

```js
A.test("large refunds require recent passkey", {
  given: {
    subject: {roles: ["support"]},
    auth: {
      amr: ["password", "totp"],
      ageSeconds: 600
    },
    authorizationDetails: {
      type: "refund",
      amount: 5000
    }
  },

  expect: {
    outcome: "step_up",
    factor: "passkey",
    maxAge: "5m"
  }
});
```

A graph should not activate if its required tests fail.

## Hot reload with atomic activation

```js
A.deployment({
  reload: "on-file-change",

  activation: {
    validate: true,
    runTests: true,
    warmRuntimePool: true,
    atomicSwap: true
  },

  rollbackOn: [
    "script_error",
    "test_failure",
    "missing_capability",
    "invalid_protocol_graph"
  ]
});
```

## Compile-time feature selection

To preserve the tiny-binary property, native blocks should be explicitly registered in Go:

```go
kernel := auth.New(
    auth.WithOIDC(),
    auth.WithSQLite(),
    auth.WithPasskeys(),
    auth.WithMagicLinks(),
    auth.WithGoja(),
)
```

A script referring to an unavailable block should fail validation. There should be no requirement to link workload attestation, CIBA, PostgreSQL, SMS, or federation code into a binary that does not use them.

---

# 5. A practical release sequence

## First release

Build the graph engine, immutable context model, outcome algebra, runtime pool, capabilities, hooks, test runner, and named preset slots. Include:

* Local web preset
* OIDC provider
* Sessions
* SQLite and application-store adapter
* Password, passkey, TOTP, magic link, and recovery codes
* Computed claims
* Simple RBAC and custom policy
* Audit events
* Deterministic test preset

## Second release

Add the components that establish the “identity Lego” positioning:

* Multi-tenancy
* Upstream OIDC brokering
* Device authorization
* Dynamic step-up
* Token exchange
* Transaction authorization
* Preview and ephemeral realms
* Policy shadowing

## Experimental packages

Keep the more ambitious areas outside the minimal core until their abstractions stabilize:

* Workload identity
* Edge/offline realms
* Quorum authorization
* Agent delegation
* Capability issuance
* Self-issued identity
* Temporary federation
* Multi-actor identity workflows

The most important abstraction is not the JavaScript hook. It is the **typed, resumable authentication graph**. Hooks then become one kind of graph node, presets become preassembled graphs, and the same small native blocks can support ordinary login, delegated service calls, workload identity, approval workflows, and temporary trust realms without turning the product into a monolithic IdP.

[1]: https://openid.net/specs/openid-connect-core-1_0-final.html "https://openid.net/specs/openid-connect-core-1_0-final.html"
[2]: https://github.com/dop251/goja "https://github.com/dop251/goja"
[3]: https://www.rfc-editor.org/rfc/rfc9700.txt "https://www.rfc-editor.org/rfc/rfc9700.txt"
[4]: https://www.w3.org/TR/webauthn-3/ "https://www.w3.org/TR/webauthn-3/"
[5]: https://www.rfc-editor.org/rfc/rfc9470.pdf "https://www.rfc-editor.org/rfc/rfc9470.pdf"
[6]: https://www.rfc-editor.org/rfc/rfc8628.txt "https://www.rfc-editor.org/rfc/rfc8628.txt"
[7]: https://openid.net/specs/openid-client-initiated-backchannel-authentication-core-1_0.html "https://openid.net/specs/openid-client-initiated-backchannel-authentication-core-1_0.html"
[8]: https://www.rfc-editor.org/rfc/rfc8693.txt "https://www.rfc-editor.org/rfc/rfc8693.txt"
[9]: https://spiffe.io/docs/latest/spiffe-about/spiffe-concepts/ "https://spiffe.io/docs/latest/spiffe-about/spiffe-concepts/"
[10]: https://www.rfc-editor.org/rfc/rfc9396.txt "https://www.rfc-editor.org/rfc/rfc9396.txt"
[11]: https://openid.net/specs/openid-connect-self-issued-v2-1_0-12.html "https://openid.net/specs/openid-connect-self-issued-v2-1_0-12.html"

