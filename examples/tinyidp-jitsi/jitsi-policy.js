const A = require("tinyidp").v1;

module.exports = A.program("jitsi-policy", program => {
  const decide = A.lambda("integration.jitsi.authorize@v1", {
    kind: "provider",
    input: "integration.jitsi.authorize.input.v1",
    output: "integration.jitsi.authorize.output.v1",
    outcomes: ["complete", "deny"],
    effects: [],
    capabilities: [],
    timeoutMs: 50,
    maxCapabilityCalls: 0,
    maxOutputBytes: 4096,
    run: ctx => {
      if (!ctx.input.identity.email) {
        return A.result.deny("verified_email_required");
      }
      return A.result.complete({
        kind: "complete",
        claims: {
          displayName: ctx.input.identity.displayName,
          includeEmail: true,
          moderator: false,
        },
      });
    },
  });

  program.provider("authorization", "jitsi", {
    version: 1,
    state: "virtual",
    replayProtection: "none",
    revocation: "none",
    handlers: {decide},
  });
});
