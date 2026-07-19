const A = require("tinyidp").v1;

module.exports = A.program("open-signup", program => {
  const start = A.lambda("signup.start", {
    input: "signupStartInput", output: "signupResult",
    outcomes: ["present"], effects: [], capabilities: [],
    timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => ctx.present.form({
      title: "Create an account",
      resume: "submitted",
      fields: [A.field.displayName(), A.field.email(), A.field.password(), A.field.passwordConfirmation()],
      actions: [A.action.submit(), A.action.deny()],
      carry: {}, expiresInSeconds: 300,
    }),
  });
  const submitted = A.lambda("signup.submitted", {
    input: "signupSubmittedInput", output: "signupResult",
    outcomes: ["commit"], effects: ["createLocalIdentity", "attachPasswordCredential"], capabilities: [],
    timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => ctx.commit.signup({
      login: ctx.input.email,
      displayName: ctx.input.displayName,
      password: ctx.secret.password,
      passwordConfirmation: ctx.secret.passwordConfirmation,
    }),
  });
  program.workflow("signup", {
    version: 1, entry: "start", handlers: {start, submitted},
    edges: [{from: "start", outcome: "present", to: "submitted", input: "signupSubmittedInput"}],
  });
});
