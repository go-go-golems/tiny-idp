// Keep this deployable example in sync with pkg/idpsignup/invite_required_signup.js.
// Message Desk remains open-signup; the goja client requires a durable code.
const A = require("tinyidp").v1;

module.exports = A.program("shared-open-and-invite-signup", program => {
  program.capabilities({"invitation.lookup": {version: 1}});
  const validateInvitation = A.lambda("invitation.signup.validate", {
    kind: "provider", input: "signupSubmittedInput", output: "signupResult",
    outcomes: ["complete", "deny"], effects: [], capabilities: ["invitation.lookup"],
    timeoutMs: 250, maxCapabilityCalls: 1, maxOutputBytes: 1024,
    run: async ctx => {
      const decision = await ctx.cap.invitation.lookup({code: ctx.input.inviteCode || ""});
      return decision.valid ? A.result.complete() : A.result.deny("invitation.rejected");
    },
  });
  program.provider("invitation", "signup", {version: 1, state: "durable", replayProtection: "one_time", revocation: "durable", handlers: {validate: validateInvitation}});

  const start = A.lambda("signup.start", {
    input: "signupStartInput", output: "signupResult", outcomes: ["present"], effects: [], capabilities: [], timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => {
      const fields = [A.field.displayName(), A.field.email(), A.field.password(), A.field.passwordConfirmation()];
      if (ctx.input.clientId === "goja-auth-host-demo") fields.push(A.field.inviteCode());
      return ctx.present.form({title: "Create an account", resume: "submitted", fields, actions: [A.action.submit(), A.action.deny()], carry: {}, expiresInSeconds: 300});
    },
  });
  const submitted = A.lambda("signup.submitted", {
    input: "signupSubmittedInput", output: "signupResult", outcomes: ["commit", "deny"], effects: ["createLocalIdentity", "attachPasswordCredential", "consumeInvitation"], capabilities: [], timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => {
      const inviteCode = ctx.input.inviteCode || "";
      if (inviteCode && !(ctx.evidence["invitation.signup"] || {}).accepted) return A.result.deny("invitation.rejected");
      return ctx.commit.signup({login: ctx.input.email, displayName: ctx.input.displayName, password: ctx.secret.password, passwordConfirmation: ctx.secret.passwordConfirmation, inviteCode});
    },
  });
  program.workflow("signup", {version: 1, entry: "start", handlers: {start, submitted}, edges: [{from: "start", outcome: "present", to: "submitted", input: "signupSubmittedInput"}]});
  program.test("valid-invitation-provider", {lambda: "invitation.signup.validate", input: {inviteCode: "test-code"}, expectedKind: "complete", fakes: {"invitation.lookup": {valid: true}}});
  program.test("invalid-invitation-provider", {lambda: "invitation.signup.validate", input: {inviteCode: "test-code"}, expectedKind: "deny", fakes: {"invitation.lookup": {valid: false}}});
});
