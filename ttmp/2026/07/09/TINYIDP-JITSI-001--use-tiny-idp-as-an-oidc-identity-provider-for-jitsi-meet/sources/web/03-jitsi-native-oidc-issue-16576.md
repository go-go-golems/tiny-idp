(disclaimer: I'm no expert regarding OIDC, JWT, IdPs etc, so there could be something I missed that would make this use-case a lot easier)

### What problem are you trying to solve?

When using Jitsi Meet with an OpenID Connect (OIDC) identity provider such as Keycloak, Auth0, or Azure AD, Jitsi cannot consume the tokens returned by the IdP directly.

Integrating Jitsi Meet with standard OpenID Connect (OIDC) identity providers (e.g., Keycloak, Auth0, Azure AD) currently requires an intermediate page or service to handle the token redirect.

Today, most setups need a “bridge” (such as static/oauth.html or [https://github.com/nordeck/jitsi-keycloak-adapter](https://github.com/nordeck/jitsi-keycloak-adapter)) because:

- The OIDC IdP returns an id\_token in the fragment (#id\_token=...) or via form\_post — which Jitsi doesn’t read.
- Jitsi expects a JWT in the query string as ?jwt=....
- Since fragments aren’t sent to the server, reverse proxies like Nginx can’t rewrite the response.
- Integrators must inject a custom static page or microservice that reads the fragment and redirects to /room?jwt=.

This complicates SSO integrations, adds latency, and breaks deep-linking for mobile or embedded Jitsi clients.

# Summary

By letting Jitsi Meet directly consume OIDC tokens (from fragments or POSTs) and recognize configurable token parameter names, deployments could:

- Integrate cleanly with any standards-based IdP (Keycloak, Auth0, Azure AD, etc.)
- Simplify web and mobile SSO
- Stay fully compatible with modern flows like Authorization Code + PKCE

This feature would make Jitsi a first-class OIDC client instead of requiring every deployer to maintain a “glue” layer between their IdP and Jitsi.

### What solution would you like to see?

Add native OIDC redirect handling to Jitsi Meet, so it can consume IdP responses directly without requiring an intermediary.

Specifically:

1. Parse tokens from URL fragments  
	If a user lands on a URL like /#id\_token=...&room=..., Jitsi should automatically extract the id\_token and join the room as if it were ?jwt=....
2. Support alternate token parameter names  
	Allow configuration in config.js, e.g.:
```js
config.jwtParamNames = ['jwt', 'id_token', 'access_token'];
config.roomParamName = 'room';
```

This allows direct compatibility with OIDC-compliant IdPs that return id\_token instead of jwt.

3. Optionally handle `response_mode=form_post`  
	When an IdP POSTs an id\_token to the redirect URI, Jitsi should parse it and continue the join flow automatically.

This would make Jitsi natively compatible with the OIDC Authorization Code (with PKCE) or Implicit flows — no more glue pages or hacks.

### Is there an alternative?

1. Custom bridge page (e.g. /static/oauth.html)  
	A JavaScript file that reads the id\_token from the URL and redirects to Jitsi with ?jwt=....

Works, but requires static injection and extra redirect.

2. Custom backend service  
	An external service that handles the OIDC redirect, exchanges the code or token, then redirects back to Jitsi.

Secure and standards-based, but adds complexity and operational overhead.

3. Pre-signed JWT links  
	Mint tokens on a server and share URLs like /room?jwt=.

Works well for service integrations or bots, but not for human users doing interactive SSO.