package domain

// ClaimsForScopes returns user claims permitted by granted scopes. Protocol
// claims such as iss/aud/exp/iat are built by the token layer; this function
// covers user/profile claims.
func ClaimsForScopes(u User, scopes []string) map[string]any {
	claims := map[string]any{"sub": u.Sub}
	if HasScope(scopes, "email") {
		if u.Email != "" {
			claims["email"] = u.Email
		}
		claims["email_verified"] = u.EmailVerified
	}
	if HasScope(scopes, "profile") {
		if u.Name != "" {
			claims["name"] = u.Name
		}
		if u.PreferredUsername != "" {
			claims["preferred_username"] = u.PreferredUsername
		}
		if len(u.Groups) > 0 {
			claims["groups"] = u.Groups
		}
		if len(u.Roles) > 0 {
			claims["roles"] = u.Roles
		}
		if u.Tenant != "" {
			claims["tenant"] = u.Tenant
		}
		if u.Locale != "" {
			claims["locale"] = u.Locale
		}
	}
	return claims
}
