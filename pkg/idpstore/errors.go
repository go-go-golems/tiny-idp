package idpstore

import "errors"

var (
	ErrEmptyClientID             = errors.New("client id is required")
	ErrEmptyRedirectURI          = errors.New("redirect URI is required")
	ErrInvalidRedirectURI        = errors.New("redirect URI is invalid")
	ErrWildcardRedirectURI       = errors.New("redirect URI must not contain wildcards")
	ErrRedirectURIFragment       = errors.New("redirect URI must not contain a fragment")
	ErrProductionRedirectHTTP    = errors.New("production redirect URI must use https except loopback development clients")
	ErrPublicClientRequiresPKCE  = errors.New("public clients require PKCE")
	ErrPublicClientHasSecret     = errors.New("public clients must not have a secret")
	ErrConfidentialMissingSecret = errors.New("confidential clients require a secret hash")
	ErrEmptySubject              = errors.New("user subject is required")
	ErrSubjectUsesEmail          = errors.New("user subject must not equal email")
)
