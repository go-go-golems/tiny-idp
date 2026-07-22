const A = require("tinyidp").v1;

module.exports = A.program("shared-verified-open-and-invite-signup", program => {
  program.capabilities({
    "identity.displayName.lookup": {version: 1},
    "invitation.lookup": {version: 1},
  });

  const validateInvitation = A.lambda("invitation.signup.validate", {
    kind: "provider",
    input: "signupSubmittedInput", output: "signupResult",
    outcomes: ["complete", "deny"], effects: [], capabilities: ["invitation.lookup"],
    timeoutMs: 250, maxCapabilityCalls: 1, maxOutputBytes: 1024,
    run: async ctx => {
      const decision = await ctx.cap.invitation.lookup({code: ctx.input.inviteCode || ""});
      return decision.valid ? A.result.complete() : A.result.deny("invitation.rejected");
    },
  });

  program.provider("invitation", "signup", {
    version: 1,
    state: "durable",
    replayProtection: "one_time",
    revocation: "durable",
    handlers: {validate: validateInvitation},
  });

  const start = A.lambda("signup.start", {
    input: "signupStartInput", output: "signupResult",
    outcomes: ["present"], effects: [], capabilities: [],
    timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => {
      const fields = [A.field.displayName(), A.field.email()];
      if (ctx.input.clientId === "goja-auth-host-demo") fields.push(A.field.inviteCode());
      return ctx.present.form({
        title: "Create an account",
        resume: "submitted",
        fields,
        actions: [A.action.submit(), A.action.deny()],
        carry: {},
        expiresInSeconds: 300,
      });
    },
  });

  const submitted = A.lambda("signup.submitted", {
    input: "signupSubmittedInput", output: "signupResult",
    outcomes: ["present", "challenge"], effects: [], capabilities: ["identity.displayName.lookup"],
    timeoutMs: 250, maxCapabilityCalls: 1, maxOutputBytes: 4096,
    run: async ctx => {
      const result = await ctx.cap.identity.displayName.lookup({displayName: ctx.input.displayName});
      if (!result.available) {
        return ctx.present.form({
          title: "Create an account",
          resume: "submitted",
          fields: [A.field.displayName(), A.field.email()],
          actions: [A.action.submit(), A.action.deny()],
          values: {displayName: ctx.input.displayName, email: ctx.input.email},
          errors: [{field: A.field.displayName(), code: "rejected"}],
          carry: ctx.input.inviteCode ? {inviteCode: ctx.input.inviteCode} : {},
          expiresInSeconds: 300,
        });
      }
      return ctx.challenge.emailCode({
        email: ctx.input.email,
        template: "signup-code",
        resume: "emailVerified",
        expiresInSeconds: 900,
        maximumAttempts: 5,
        maximumResends: 2,
        carry: ctx.input,
      });
    },
  });

  const emailVerified = A.lambda("signup.emailVerified", {
    input: "signupSubmittedInput", output: "signupResult",
    outcomes: ["present"], effects: [], capabilities: [],
    timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => {
      if (!ctx.evidence.email || ctx.evidence.email.address !== ctx.input.email) {
        return A.result.error("email_evidence_required");
      }
      return ctx.present.form({
        title: "Choose a password",
        resume: "passwordSubmitted",
        fields: [A.field.password(), A.field.passwordConfirmation()],
        actions: [A.action.submit(), A.action.deny()],
        carry: ctx.input,
        expiresInSeconds: 300,
      });
    },
  });

  const passwordSubmitted = A.lambda("signup.passwordSubmitted", {
    input: "signupSubmittedInput", output: "signupResult",
    outcomes: ["commit"],
    effects: ["createLocalIdentity", "attachPasswordCredential", "consumeInvitation"],
    capabilities: [], timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => ctx.commit.signup({
      login: ctx.input.email,
      displayName: ctx.input.displayName,
      uniqueDisplayName: true,
      password: ctx.secret.password,
      passwordConfirmation: ctx.secret.passwordConfirmation,
      inviteCode: ctx.input.inviteCode || "",
    }),
  });

  program.workflow("signup", {
    version: 1,
    entry: "start",
    handlers: {start, submitted, emailVerified, passwordSubmitted},
    edges: [
      {from: "start", outcome: "present", to: "submitted", input: "signupSubmittedInput"},
      {from: "submitted", outcome: "present", to: "submitted", input: "signupSubmittedInput"},
      {from: "submitted", outcome: "challenge", to: "emailVerified", input: "signupSubmittedInput"},
      {from: "emailVerified", outcome: "present", to: "passwordSubmitted", input: "signupSubmittedInput"},
    ],
  });

  program.test("valid-invitation-provider", {
    lambda: "invitation.signup.validate",
    input: {inviteCode: "test-code"},
    expectedKind: "complete",
    fakes: {"invitation.lookup": {valid: true}},
  });
  program.test("invalid-invitation-provider", {
    lambda: "invitation.signup.validate",
    input: {inviteCode: "test-code"},
    expectedKind: "deny",
    fakes: {"invitation.lookup": {valid: false}},
  });
  program.test("message-desk-starts-open-verified-signup", {
    lambda: "signup.start",
    input: {clientId: "tinyidp-message-app", redirectUri: "https://message.example/callback", requestedScope: "openid", interactionId: "message", hasBrowserSession: false},
    expectedKind: "present",
  });
  program.test("display-name-collision-represents-signup", {
    lambda: "signup.submitted",
    input: {displayName: "Taken", email: "taken@example.test"},
    expectedKind: "present",
    fakes: {"identity.displayName.lookup": {available: false}},
  });
});
