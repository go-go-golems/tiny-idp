# Embedded provider example

This directory is a runnable development host that composes only public tiny-idp packages:

- `pkg/sqlitestore` for durable identity state;
- `pkg/idpaccounts` for account creation and password authentication;
- `embeddedidp.Bootstrap` for the browser client and initial signing key;
- `embeddedidp.New` for the provider handler.

Run manually from the repository root while developing the API:

```bash
go run ./examples/embedded
```

The issuer listens at `http://127.0.0.1:5556/idp`. The example creates the development account `alice` with password `correct horse battery staple` and is idempotent when rerun against the same database.

This is a development composition. Production mode additionally requires an HTTPS issuer, secure cookies, a durable audit reporter, a production-ready rate limiter and client-address resolver, a strong token secret, maintenance-capable persistent storage, and the other checks documented in [`../../docs/embedding-foundations.md`](../../docs/embedding-foundations.md).
