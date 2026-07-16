package oidcmeta

type Discovery struct {
	Issuer                                    string   `json:"issuer"`
	AuthorizationEndpoint                     string   `json:"authorization_endpoint"`
	DeviceAuthorizationEndpoint               string   `json:"device_authorization_endpoint"`
	TokenEndpoint                             string   `json:"token_endpoint"`
	UserinfoEndpoint                          string   `json:"userinfo_endpoint"`
	IntrospectionEndpoint                     string   `json:"introspection_endpoint"`
	IntrospectionEndpointAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported"`
	JWKSURI                                   string   `json:"jwks_uri"`
	EndSessionEndpoint                        string   `json:"end_session_endpoint,omitempty"`
	ResponseTypesSupported                    []string `json:"response_types_supported"`
	GrantTypesSupported                       []string `json:"grant_types_supported"`
	SubjectTypesSupported                     []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported          []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                           []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported         []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported             []string `json:"code_challenge_methods_supported"`
	ClaimsSupported                           []string `json:"claims_supported"`
}

func ProductionDiscovery(issuer string) (Discovery, error) {
	iss, err := ParseIssuer(issuer)
	if err != nil {
		return Discovery{}, err
	}
	return Discovery{
		Issuer:                      iss.String(),
		AuthorizationEndpoint:       iss.Endpoint("/authorize"),
		DeviceAuthorizationEndpoint: iss.Endpoint("/device_authorization"),
		TokenEndpoint:               iss.Endpoint("/token"),
		UserinfoEndpoint:            iss.Endpoint("/userinfo"),
		IntrospectionEndpoint:       iss.Endpoint("/introspect"),
		IntrospectionEndpointAuthMethodsSupported: []string{"client_secret_basic"},
		JWKSURI:                           iss.Endpoint("/jwks"),
		EndSessionEndpoint:                iss.Endpoint("/end-session"),
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{"RS256"},
		ScopesSupported:                   []string{"openid", "profile", "email", "offline_access"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		ClaimsSupported:                   []string{"iss", "sub", "aud", "exp", "iat", "auth_time", "nonce", "email", "email_verified", "name", "preferred_username", "groups", "roles", "tenant", "locale"},
	}, nil
}
