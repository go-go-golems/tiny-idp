# Tiny-IDP provider patterns

These examples are deliberately split between JavaScript program declarations
and native host bindings. A script may declare a workflow or provider and call
only declared capabilities. It does not open a database, read a signing key,
send arbitrary email, or create a user/session directly. Native code validates
and commits those operations.

## Open signup

Use the ready-to-run baseline in
[`pkg/idpsignup/open_signup.js`](../../pkg/idpsignup/open_signup.js). It uses
the `signup` workflow and requests only the native `createLocalIdentity` and
`attachPasswordCredential` effects. The host owns the eventual account,
credential, session, and authorization-code transaction.

## Allowed email domain

Put this decision in the submitted signup lambda. The value was normalized by
the provider-owned email field before the lambda sees it.

```javascript
if (!ctx.input.email.endsWith("@example.test")) {
  return ctx.deny("signup.email_domain_not_allowed");
}
return ctx.commit.signup({
  login: ctx.input.email,
  displayName: ctx.input.displayName,
  password: ctx.secret.password,
  passwordConfirmation: ctx.secret.passwordConfirmation,
});
```

The denial code is a stable public category, not a dynamic explanation of why
an account was or was not eligible.

## Signed invitation

The host verifies the signed invitation with `idpinvite.KeyRing.Verify` before
it grants an invitation evidence reference. The token includes audience,
issuer, policy version, issue/not-before/expiry times, and optional subject or
email restrictions. JavaScript receives only the verified result, never the
key ring:

```go
verified, err := keyRing.Verify(inviteCode, idpinvite.VerifyRequest{
    Audience: "message-app", Email: normalizedEmail, Now: clock.Now(),
})
// Bind verified.ID as native evidence before the signup commit transaction.
```

## Computed eligibility

The JavaScript declaration requests one narrow capability. It cannot obtain the
directory client or make arbitrary network calls:

```javascript
program.capabilities({"invitation.eligibility": {version: 1}});
const validate = A.lambda("community.validate", {
  kind: "provider", input: "probe", output: "decision",
  outcomes: ["complete"], effects: [], capabilities: ["invitation.eligibility"],
  timeoutMs: 250, maxCapabilityCalls: 1, maxOutputBytes: 1024,
  run: async ctx => A.result.complete(
    await ctx.cap.invitation.eligibility(ctx.input)
  ),
});
program.provider("invitation", "community", {
  version: 1, state: "virtual", replayProtection: "none", revocation: "none",
  handlers: {validate},
});
```

Bind the capability with `idpinvite.NewEligibilityCapability`; its input and
output codecs reject undeclared fields and unbounded results.

## Durable one-time invitation

For invitations that must be consumed once, use `idpinvite.DurableService`.
Issue and revoke them from native administration code; at signup commit,
`RedeemInTransaction` performs the keyed-hash lookup and atomic consumption.

```go
service.Issue(ctx, idpinvite.DurableIssue{
    ID: "invite-42", Code: operatorGeneratedCode,
    Audience: "message-app", PolicyVersion: "v1", ExpiresAt: expiry,
})
// During the one native signup transaction:
evidence, err := service.RedeemInTransaction(ctx, tx, code, "message-app", now)
```

The raw code is neither persisted nor emitted in audit/metric data.

## Virtual identity

Use a virtual identity only after native evidence has established its seed. The
subject is pairwise and stable for the namespace but no local user row is
created:

```go
candidate, err := idpidentity.NewVirtual(deriver, idpidentity.VirtualRequest{
    Namespace: "message-app", Seed: verifiedExternalSubject,
    Email: verifiedEmail, EmailVerified: true, DisplayName: displayName,
})
```

`candidate.ProfileClaims()` intentionally excludes protocol-owned claims such
as `sub`, `iss`, and `aud`.

## Local stored identity

Use the standard signup commit path when Tiny-IDP must own a password-backed
account. The JavaScript outcome requests effects; `SignupCommitter` performs
the identity and credential writes atomically, consumes invitation evidence,
and creates the browser session only after all native validation succeeds.

See [`pkg/idpsignup/email_verified_signup.js`](../../pkg/idpsignup/email_verified_signup.js)
for the multi-request version that verifies email before collecting a password.
