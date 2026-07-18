# Embedded provider example

This directory is a minimal, complete, self-contained web application. One Go
process and one public origin serve both the relying party and the embedded IdP:

- `pkg/sqlitestore` for durable identity state;
- `pkg/idpaccounts` for account creation and password authentication;
- `embeddedidp.Bootstrap` for the browser client and initial signing key;
- `embeddedidp.New` for the provider handler mounted at `/idp/`;
- an Authorization Code + PKCE relying party at `/`;
- the implemented callback at `/auth/callback`;
- bounded in-process discovery, token, JWKS, and UserInfo requests;
- RS256 ID-token signature and issuer, audience, expiry, and nonce validation;
- independent application sessions and coordinated RP/IdP logout.

Run manually from the repository root while developing the API:

```bash
go run ./examples/embedded
```

Open `http://127.0.0.1:5556/` and choose **Sign in with the embedded IdP**.
The issuer is `http://127.0.0.1:5556/idp`. The example creates the development
account `alice` with password `correct horse battery staple` and is idempotent
when rerun against the same database.

The browser still traverses ordinary OIDC front-channel redirects. Back-channel
requests never leave the process: `NewInProcessIssuerTransport` dispatches only
exact issuer-origin URLs to the provider handler and rejects network fallback.

This is a development composition. Production mode additionally requires an HTTPS issuer, secure cookies, a durable audit reporter, a production-ready rate limiter and client-address resolver, a strong token secret, maintenance-capable persistent storage, and the other checks documented in [`../../docs/embedding-foundations.md`](../../docs/embedding-foundations.md).
