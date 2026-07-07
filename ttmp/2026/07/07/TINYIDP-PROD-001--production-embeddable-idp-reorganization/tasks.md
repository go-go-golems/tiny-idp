# Tasks

## TODO

- [x] Download protocol and security source material into sources/ <!-- t:6rgc -->
- [x] Analyze current tiny-idp architecture and file-level behavior <!-- t:o8ei -->
- [x] Write production reorganization design and implementation guide <!-- t:0qfd -->
- [x] Write OIDC intern textbook <!-- t:9tsa -->
- [x] Validate ticket with docmgr doctor <!-- t:1hua -->
- [x] Upload deliverables to reMarkable <!-- t:86ft -->
- [ ] Phase 1.1: Add internal/domain package with client, user, claim, grant, token, session, and key models <!-- t:96fv -->
- [ ] Phase 1.2: Add production/dev validation for issuer, clients, redirect URIs, PKCE, and scopes <!-- t:j7mf -->
- [ ] Phase 1.3: Add domain unit tests for exact redirect matching, wildcard rejection, scope filtering, and stable subjects <!-- t:od2h -->
- [ ] Phase 2.1: Add internal/storage interfaces and reusable store test suite <!-- t:ifvd -->
- [ ] Phase 2.2: Implement concurrency-safe internal/store/memory for clients, users, codes, refresh tokens, sessions, grants, and keys <!-- t:t5iv -->
- [ ] Phase 2.3: Add memory store tests for one-time code consume and refresh-token reuse detection <!-- t:241z -->
- [ ] Phase 3.1: Add internal/oidcmeta issuer/discovery/JWKS helpers <!-- t:ddix -->
- [ ] Phase 3.2: Add internal/keys helpers for RSA signing keys and JWKS conversion <!-- t:gv0a -->
- [ ] Phase 3.3: Add metadata/key tests for path issuers, strict discovery, and public-only JWKS <!-- t:sae7 -->
- [ ] Phase 4.1: Add internal/fositeadapter strict adapter seam with explicit supported production handlers <!-- t:l26u -->
- [ ] Phase 4.2: Add strict authorize/token handler spike with S256 PKCE and no mock debug behavior <!-- t:xr0r -->
- [ ] Phase 4.3: Add end-to-end strict authorization-code flow test <!-- t:o0yt -->
- [ ] Phase 5.1: Add pkg/embeddedidp public Options and Provider API <!-- t:uzcc -->
- [ ] Phase 5.2: Add production-mode validation for HTTPS issuer, secure cookies, persistent keys/stores, PKCE, and debug exclusion <!-- t:kyh4 -->
- [ ] Phase 5.3: Add embedded provider tests and example wiring <!-- t:4gkw -->
- [ ] Phase 6.1: Add internal/store/sqlite migrations and schema <!-- t:kzxg -->
- [ ] Phase 6.2: Implement SQLite store for clients, users, sessions, grants, authorization codes, refresh tokens, and signing keys <!-- t:08mh -->
- [ ] Phase 6.3: Run reusable store suite against SQLite and verify restart-stable signing keys <!-- t:bzzh -->
- [ ] Phase 7.1: Add tinyidp serve --engine mock|fosite flag with mock default <!-- t:obc6 -->
- [ ] Phase 7.2: Wire strict engine into CLI shared issuer/client/users config <!-- t:0nuc -->
- [ ] Phase 7.3: Add dual-engine smoke/config tests and update docs <!-- t:4cgi -->
