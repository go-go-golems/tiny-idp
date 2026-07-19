const A = require("tinyidp").v1;

module.exports = A.program("email-verified-signup", program => {
  const start = A.lambda("signup.start", {
    input: "signupStartInput", output: "signupResult", outcomes: ["present"], effects: [], capabilities: [], timeoutMs: 250, maxCapabilityCalls: 0, maxOutputBytes: 4096,
    run: ctx => ctx.present.form({title:"Create an account", resume:"submitted", fields:[A.field.displayName(),A.field.email()], actions:[A.action.submit(),A.action.deny()], carry:{}, expiresInSeconds:300}),
  });
  const submitted = A.lambda("signup.submitted", {
    input:"signupSubmittedInput", output:"signupResult", outcomes:["challenge"], effects:[], capabilities:[], timeoutMs:250,maxCapabilityCalls:0,maxOutputBytes:4096,
    run: ctx => ctx.challenge.emailCode({email:ctx.input.email,template:"signup-code",resume:"emailVerified",expiresInSeconds:900,maximumAttempts:5,maximumResends:2,carry:ctx.input}),
  });
  const emailVerified = A.lambda("signup.emailVerified", {
    input:"signupSubmittedInput", output:"signupResult", outcomes:["present"], effects:[], capabilities:[], timeoutMs:250,maxCapabilityCalls:0,maxOutputBytes:4096,
    run: ctx => { if (!ctx.evidence.email || ctx.evidence.email.address !== ctx.input.email) return A.result.error("email_evidence_required"); return ctx.present.form({title:"Choose a password",resume:"passwordSubmitted",fields:[A.field.password(),A.field.passwordConfirmation()],actions:[A.action.submit(),A.action.deny()],carry:ctx.input,expiresInSeconds:300}); },
  });
  const passwordSubmitted = A.lambda("signup.passwordSubmitted", {
    input:"signupSubmittedInput", output:"signupResult", outcomes:["commit"], effects:["createLocalIdentity","attachPasswordCredential"], capabilities:[], timeoutMs:250,maxCapabilityCalls:0,maxOutputBytes:4096,
    run: ctx => ctx.commit.signup({login:ctx.input.email,displayName:ctx.input.displayName,password:ctx.secret.password,passwordConfirmation:ctx.secret.passwordConfirmation}),
  });
  program.workflow("signup",{version:1,entry:"start",handlers:{start,submitted,emailVerified,passwordSubmitted},edges:[
    {from:"start",outcome:"present",to:"submitted",input:"signupSubmittedInput"},
    {from:"submitted",outcome:"challenge",to:"emailVerified",input:"signupSubmittedInput"},
    {from:"emailVerified",outcome:"present",to:"passwordSubmitted",input:"signupSubmittedInput"},
  ]});
});
