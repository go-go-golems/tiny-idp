---
Title: Seeded-user Password Semantics Guide
Ticket: TINYIDP-PASSWORDS-001
Status: active
Topics:
    - oidc
    - testing
    - identity
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/scenario/scenario.go
      Note: |-
        Scenario registry is the runtime lookup path for seeded users
        Scenario model password metadata target
    - Path: internal/scenario/seeded_users.go
      Note: |-
        Current seeded-user schema and conversion to scenarios
        Seeded-user schema extension target
    - Path: internal/server/authorize.go
      Note: |-
        Current authorize POST accepts login and ignores password
        Authorize POST password validation insertion point
    - Path: internal/server/static/login.html
      Note: |-
        Login form already has a password field marked ignored
        Login form password field copy
ExternalSources: []
Summary: Design and implementation guide for optional password checks on tinyidp seeded users while keeping default test-mode login permissive.
LastUpdated: 2026-07-05T17:45:00-04:00
WhatFor: Use when implementing optional password fields for seeded users, login-form validation behavior, and tests that mimic Keycloak demo credentials.
WhenToUse: Read before changing seeded-user schema, authorize POST login validation, login page wording, or docs around password behavior.
---


# Seeded-user Password Semantics Guide

## Executive summary

`tinyidp` currently treats the login field as the user/scenario selector and ignores the password field. That is a good default for local OIDC client testing: it keeps the mock simple, fast, and focused on relying-party behavior rather than account security. However, some demos and smoke tests are easier to understand when they can use familiar credentials such as `alice / alice-password` and `bob / bob-password`, especially when replacing Keycloak tutorials.

This ticket proposes optional password semantics for seeded users only. If a seeded user defines a password, `tinyidp` validates the submitted password for that user. If no password is configured, login remains permissive. Built-in scenarios continue to work without passwords. This makes `tinyidp` more realistic for tutorials without turning it into a production account system.

The key rule is:

> Passwords are test fixture selectors, not security credentials.

They should be plain-text fixture values by default, with optional later support for hashes only if a real test need appears.

## Problem statement and scope

The current login page contains a password input, but the UI says the password is ignored. `authorize.go` normalizes the posted `login`, looks up a scenario, and creates a session without checking `password`. This is clear for a mock IdP, but it creates two usability problems:

1. Keycloak replacement tutorials usually document `alice-password` and `bob-password`. Users expect those passwords to matter.
2. Negative login tests cannot currently assert “wrong password produces a login failure.”

This ticket covers optional password validation for seeded users. It does not add production authentication, password hashing requirements, lockouts, reset flows, MFA, or persistent account storage.

## Current-state analysis

### Seeded users

`internal/scenario/seeded_users.go` defines `SeededUser` with fields for login, subject, email, name, email verification, extra claims, omitted claims, and category. `LoadSeededUsers` accepts JSON or YAML and converts users into normal scenarios. Seeded users can override builtins such as `alice` and `bob`.

This is the right place to add fixture password metadata because seeded users already represent deterministic test principals.

### Authorization POST

`internal/server/authorize.go` currently handles POST by:

1. parsing the submitted form;
2. validating OIDC authorize params;
3. normalizing `login`;
4. looking up a scenario;
5. applying any auth-error scenario;
6. creating an IdP session;
7. issuing an authorization code.

The posted `password` is not read. The implementation can add password validation between scenario lookup and session creation.

### Login form

`internal/server/static/login.html` already contains a password field with placeholder `ignored`. Once optional passwords exist, that text should become conditional or more precise. If the template remains static, change it to “required only for seeded users that define one” or “optional unless configured.”

## Proposed model

### Data model

Extend `SeededUser`:

```go
type SeededUser struct {
    Login string `json:"login" yaml:"login"`
    Sub   string `json:"sub" yaml:"sub"`
    Email string `json:"email" yaml:"email"`
    Name  string `json:"name" yaml:"name"`

    Password string `json:"password" yaml:"password"`

    EmailVerified      *bool `json:"email_verified" yaml:"email_verified"`
    EmailVerifiedKebab *bool `json:"email-verified" yaml:"email-verified"`
    Claims             map[string]any `json:"claims" yaml:"claims"`
    OmitClaims         []string       `json:"omit_claims" yaml:"omit_claims"`
    Category           string         `json:"category" yaml:"category"`
}
```

Add password metadata to `Scenario` rather than doing a separate lookup table:

```go
type Scenario struct {
    Name        string
    Description string
    Category    string
    User        user.User

    Password string // optional test fixture password

    ExtraClaims map[string]any
    OmitClaims  []string
    AuthError   string
    TokenError  string
    // ... existing fields
}
```

The scenario registry remains the lookup path. This avoids parallel maps keyed by login.

### Semantics

- If `Scenario.Password == ""`, password validation is skipped.
- If `Scenario.Password != ""`, the submitted password must match exactly.
- Empty submitted password fails when a password is configured.
- Wrong password returns an HTTP 401 or 400 from the authorize POST without issuing an authorization code.
- Auth-error scenarios still work. Password validation should happen before scenario `AuthError` redirection for password-protected seeded users, so a wrong password does not trigger the scenario's normal auth error.

Suggested status code:

- Use `401 Unauthorized` for wrong password.
- Use `400 Bad Request` for missing login or malformed form.

### YAML example

```yaml
users:
  - login: alice
    password: alice-password
    sub: user-alice-fixed
    email: alice@example.test
    name: Alice Inbox
    claims:
      groups: [inbox-users]
      tenant: personal
  - login: bob
    password: bob-password
    sub: user-bob-fixed
    email: bob@example.test
    name: Bob Inbox
    claims:
      groups: [inbox-users]
      tenant: personal
```

### Pseudocode

```go
func (s *Server) authorizePost(w http.ResponseWriter, r *http.Request, ar authorizeRequest) {
    login := user.Normalize(r.PostForm.Get("login"))
    if login == "" {
        http.Error(w, "login is required", http.StatusBadRequest)
        return
    }

    sc, _ := s.registry.Lookup(login)

    if sc.Password != "" {
        submitted := r.PostForm.Get("password")
        if submitted != sc.Password {
            http.Error(w, "invalid login or password", http.StatusUnauthorized)
            return
        }
    }

    if sc.AuthError != "" {
        redirectOAuthError(w, r, ar.RedirectURI, ar.State, sc.AuthError, "simulated "+sc.AuthError)
        return
    }

    sess := newSession(login, sc.User, &sc)
    s.setSessionCookie(w, sess)
    s.issueCodeAndRedirect(w, r, ar, sc.User, &sc, sess.AuthTime)
}
```

Use a generic error message such as `invalid login or password`. Even though this is a mock, it avoids teaching bad habits in example code.

## Testing plan

### Unit tests for seeded users

Add tests in `internal/scenario`:

1. YAML user with password loads into a scenario with `Password` set.
2. YAML user without password keeps permissive behavior.
3. JSON user with password also works.
4. Existing seeded-user claims behavior is unchanged.

### Server tests

Add tests in `internal/server`:

1. Password-protected seeded user succeeds with correct password.
2. Password-protected seeded user fails with wrong password.
3. Password-protected seeded user fails with missing password.
4. Seeded user without password still succeeds with any password.
5. Built-in scenario still succeeds with ignored password.
6. Wrong password does not create an IdP session.
7. Wrong password does not create an authorization code.

### xgoja smoke tests

After the xgoja seeded users file includes passwords:

- update smoke helpers to submit `alice-password` and `bob-password`;
- add one negative smoke only if it remains fast and readable.

## Decision records

### Decision: Passwords are optional and seeded-user-only

- **Context:** The current mock intentionally ignores passwords. Some demos need password-shaped behavior, but production auth is out of scope.
- **Options considered:** Keep ignoring passwords; require passwords for all users; validate passwords only when configured.
- **Decision:** Validate only configured seeded-user passwords.
- **Rationale:** This preserves the fast scenario-selector workflow while supporting Keycloak-like demos.
- **Consequences:** Built-in scenarios remain permissive. Docs must clearly state that passwords are optional test fixtures.
- **Status:** proposed

### Decision: Store fixture passwords in plain text initially

- **Context:** Config files are local test fixtures, not production account stores.
- **Options considered:** Plain text; bcrypt hashes; both plain text and hash fields.
- **Decision:** Start with plain text `password` only.
- **Rationale:** Plain text is clear, copy/pasteable, and matches Keycloak demo credentials in tutorials. Hashing would imply a security model tinyidp does not have.
- **Consequences:** Documentation must warn not to use real credentials.
- **Status:** proposed

### Decision: Validate before scenario auth-error hooks

- **Context:** Scenarios can intentionally return auth errors. Password-protected seeded users should still reject wrong credentials first.
- **Options considered:** Run scenario hooks before password validation; validate password first.
- **Decision:** Validate password first when a scenario has a configured password.
- **Rationale:** A wrong password should not exercise the scenario's normal success or failure path.
- **Consequences:** Auth-error scenarios with passwords require correct credentials before the configured auth error fires.
- **Status:** proposed

## Implementation phases

### Phase 1: schema and registry

- Add `Password string` to `scenario.SeededUser`.
- Add `Password string` to `scenario.Scenario`.
- Copy `su.Password` into the scenario during `seededUserToScenario`.
- Add tests for YAML/JSON loading.

### Phase 2: authorize validation

- Read `r.PostForm.Get("password")` in authorize POST.
- Add helper:

```go
func scenarioPasswordMatches(sc scenario.Scenario, submitted string) bool
```

- Return `401` for wrong configured password.
- Ensure no code/session is created on wrong password.

### Phase 3: docs and examples

- Update `README.md` seeded users section.
- Update `cmd/tinyidp/doc/pages/reference.md`.
- Update login form copy.
- Update `examples/users/personal-inbox-users.yaml` if the config ticket has added it.

### Phase 4: xgoja smoke adoption

- Update personal-inbox smoke helpers to submit configured passwords.
- Keep existing tests permissive until the users file is updated.

## Risks and alternatives

- If password validation becomes too prominent, users may mistake `tinyidp` for a real IdP. Documentation must keep the non-production warning clear.
- Exact string matching is enough for test fixtures but may surprise users expecting hashing. That is acceptable for first implementation.
- Returning the login form with an error message would be nicer for browsers than plain `401`. Plain `401` is simpler and easier for smoke tests; a later UX pass can add template errors.

## References

- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/scenario/seeded_users.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/authorize.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/static/login.html`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/tinyidp-users.yaml`
