# `tinyidp/verify` compile-only module

This package registers the only CommonJS module available to the isolated
verification-plan compiler. It converts lower-camel JavaScript data into the
versioned `verifyplan.Plan` schema. It exposes no provider, store, key, secret,
clock, filesystem, network, process, or assertion capability.

```javascript
const V = require("tinyidp/verify").v1;

module.exports = V.plan({
  suites: [{
    name: "fresh authentication",
    scenarios: [{
      name: "prompt login",
      steps: [{
        kind: "authorize.begin",
        parameters: {prompt: "login"},
      }],
      assertions: [{
        id: "freshAuthenticationBeforeIssuance",
        version: "v1",
      }],
    }],
  }],
});
```

The host compiles the source with `gojaverify.Compile`, then supplies a native
`verifyplan.Driver` and a versioned registry of native assertion functions to
`verifyplan.Runner`. JavaScript never executes a scenario or an assertion.

This separation is a security boundary, not only an API convention. Adding a
new JavaScript-visible module or live object expands the verifier's trusted
computing base and requires a threat-model update, negative capability tests,
and an explicit design decision.
