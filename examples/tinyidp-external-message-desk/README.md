# Standalone tiny-idp + Message Desk

This development demo runs a standalone tiny-idp and Message Desk as distinct
containers. The browser uses `localhost:8081` as the canonical issuer and
`localhost:8080` as the relying party. Only discovery, JWKS, and token
requests use the private Docker address `idp:8081`; the public issuer remains
unchanged for authorization responses and ID-token validation.

Run it from this directory:

```sh
docker compose up --build
```

Open `http://localhost:8080`. The committed seed is intentionally public and
development-only: it supplies `amelie` and `wesen` with the documented demo
password in `demo-seed.json`. Replace the file with an operator-controlled
secret mount before using any non-demo environment. Do not reuse a seeded
demo password.

The two named volumes are independent durable boundaries. Reset only the
demo with:

```sh
docker compose down -v
```

The provider owns accounts, its browser session, consent, chooser state, and
logout. Message Desk owns only messages and its own opaque application session.
`Log out of Message Desk` clears the latter; `Log out everywhere` also sends
the browser to the provider end-session endpoint.
