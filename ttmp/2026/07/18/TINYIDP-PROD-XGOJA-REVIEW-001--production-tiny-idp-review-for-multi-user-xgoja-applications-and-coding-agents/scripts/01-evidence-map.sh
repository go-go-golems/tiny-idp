#!/usr/bin/env bash
set -euo pipefail

# Run from the tiny-idp repository root. The output is intentionally a compact,
# line-addressable index rather than a generated copy of source code.
rg -n \
  'func (NewInitializedApplication|InitializeState|composeApplication)|func \(o Options\) Validate|func \(s \*Service\) (Create|AuthenticatePassword)|func \(p \*Provider\) (Handler|deviceAuthorization|deviceVerification)|func \(a \*Authenticator\) Authenticate' \
  cmd/tinyidp-xapp pkg/embeddedidp pkg/idpaccounts internal/fositeadapter

rg -n \
  'app\.(get|post|patch|delete)|\.auth\(|\.public\(|\.resource\(|\.csrf\(|\.allow\(|\.audit\(|\.handle\(' \
  cmd/tinyidp-xapp/app/routes ../go-go-goja/examples/xgoja/21-generated-host-auth/verbs

rg -n \
  'type RoutePlan|type SecuritySpec|type Authenticator interface|func \(e \*Enforcer\) Enforce|func \(b \*routeBuilder\)' \
  ../go-go-goja/pkg/gojahttp ../go-go-goja/modules/express

