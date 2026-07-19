package idpstore

import "time"

// Mode controls validation strictness. Development mode permits loopback HTTP
// issuers and memory stores; production mode fails closed.
type Mode string

const (
	DevMode        Mode = "dev"
	ProductionMode Mode = "production"
)

// OAuth grant types supported by tiny-idp. A client must declare every grant
// type it is permitted to use; there is intentionally no implicit default.
const (
	GrantAuthorizationCode = "authorization_code"
	GrantRefreshToken      = "refresh_token"
	GrantDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
)

// Client is a configured relying party/OAuth client.
type Client struct {
	ID                     string
	SecretHash             []byte
	Public                 bool
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	AllowedScopes          []string
	AllowedGrantTypes      []string
	// AllowedAudiences is the set of OAuth resource indicators this client may
	// request. Empty means this client cannot obtain resource-server tokens.
	AllowedAudiences []string
	// CanIntrospect permits this confidential client to authenticate as a
	// resource server at the RFC 7662 introspection endpoint.
	CanIntrospect   bool
	RequirePKCE     bool
	AccessTokenTTL  time.Duration
	IDTokenTTL      time.Duration
	RefreshTokenTTL time.Duration
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Disabled        bool
}

// User is an OIDC subject known to the provider. It intentionally carries
// profile/account state only; password hashes and credential lifecycle metadata
// live in PasswordCredential so they are never exposed through userinfo/profile
// claim paths by accident.
type User struct {
	ID                string
	Sub               string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	Groups            []string
	Roles             []string
	Tenant            string
	Locale            string
	Disabled          bool
	LockedUntil       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// PasswordCredential is the durable password verifier for one user login. The
// PasswordHash field contains a self-describing encoded hash, never a plaintext
// password. Keeping credentials separate from User keeps OIDC profile data and
// credential secrets on different access paths.
type PasswordCredential struct {
	UserID            string
	Login             string
	PasswordHash      []byte
	HashAlgorithm     string
	HashParams        PasswordHashParams
	CreatedAt         time.Time
	UpdatedAt         time.Time
	PasswordChangedAt time.Time
	Disabled          bool
}

// DurableInvitation is a one-time invitation whose browser-visible code is
// represented only by CodeHash. The state transition to RedeemedAt is made by
// the store, never by JavaScript, so a signup transaction can consume it
// without a check-then-write race.
type DurableInvitation struct {
	CodeHash         []byte
	ID               string
	Audience         string
	PolicyVersion    string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	RedeemedAt       *time.Time
	RedeemedEvidence string
}

// PasswordHashParams records the parameters used to derive PasswordHash. The
// encoded hash is authoritative for verification; these fields make admin and
// diagnostics output possible without parsing hashes everywhere.
type PasswordHashParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// AccountSecurityState tracks password-login failure and lockout state. It is
// separated from PasswordCredential so a password reset can preserve or clear
// security counters deliberately.
type AccountSecurityState struct {
	UserID                string
	FailedLoginCount      int
	FirstFailedLoginAt    *time.Time
	LastFailedLoginAt     *time.Time
	LockedUntil           *time.Time
	LastSuccessfulLoginAt *time.Time
}

// Grant records a user/client/scope authorization relationship.
type Grant struct {
	ID        string
	UserID    string
	ClientID  string
	Scope     []string
	AuthTime  time.Time
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// AuthorizationCode is the persisted representation of a one-time code. The
// raw code is never stored, only CodeHash.
type AuthorizationCode struct {
	CodeHash      []byte
	ClientID      string
	UserID        string
	GrantID       string
	RedirectURI   string
	Scope         []string
	Nonce         string
	PKCEChallenge string
	PKCEMethod    string
	AuthTime      time.Time
	ExpiresAt     time.Time
	ConsumedAt    *time.Time
}

// AccessToken is an opaque access token record. The raw bearer value is never
// stored, only TokenHash.
type AccessToken struct {
	TokenHash []byte
	GrantID   string
	ClientID  string
	UserID    string
	Scope     []string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// RefreshToken is a rotating refresh token record. TokenHash, ParentTokenHash,
// and ReplacedByHash are hashes, never raw token strings.
type RefreshToken struct {
	TokenHash       []byte
	GrantID         string
	ClientID        string
	UserID          string
	Scope           []string
	ParentTokenHash []byte
	ReplacedByHash  []byte
	CreatedAt       time.Time
	ExpiresAt       time.Time
	RevokedAt       *time.Time
	ReuseDetectedAt *time.Time
}

// Consent records a user's approval for a client and a normalized set of
// scopes. Consent is intentionally server-side state so prompt/consent behavior
// survives provider restarts and does not depend on browser-controlled data.
type Consent struct {
	UserID    string
	ClientID  string
	Scope     []string
	GrantedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Session is a server-side IdP browser session. The browser cookie carries a
// random handle; storage keeps only IDHash.
type Session struct {
	IDHash     []byte
	UserID     string
	AuthTime   time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	ACR        string
	AMR        []string
	RevokedAt  *time.Time
}

// BrowserContext is an opaque, browser-profile-scoped container for remembered
// IdP sessions. The browser receives only the random handle whose keyed hash is
// IDHash; it never receives a user ID, a session handle, or a list of accounts.
// A context is not authentication evidence on its own.
type BrowserContext struct {
	IDHash     []byte
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

// RememberedBrowserSession associates one browser context with one valid IdP
// session. DisplayLabel is deliberately host-selected, already-sanitized
// presentation data. It must not be treated as identity evidence: activation
// revalidates the linked session and user atomically.
//
// Raw browser-context, remembered-entry, and IdP-session handles are never
// persisted. Every handle field here is a keyed hash.
type RememberedBrowserSession struct {
	IDHash        []byte
	ContextIDHash []byte
	SessionIDHash []byte
	UserID        string
	DisplayLabel  string
	CreatedAt     time.Time
	LastUsedAt    time.Time
	RemovedAt     *time.Time
}

// InteractionRequiredAction is a bit set of native actions that must be
// satisfied before an authorization interaction may be consumed.
type InteractionRequiredAction uint32

const (
	InteractionRequireLogin InteractionRequiredAction = 1 << iota
	InteractionRequireFreshLogin
	InteractionRequireConsent
	InteractionRequireStepUp
	InteractionRequireAccountSelection
	// InteractionRequireRegistration marks a provider-owned account-creation
	// continuation. It is only created after the authorization request has been
	// parsed and validated, and it remains subject to the normal one-use,
	// expiry, client-generation, and browser-binding checks.
	InteractionRequireRegistration
)

func (a InteractionRequiredAction) Has(want InteractionRequiredAction) bool { return a&want != 0 }

// InteractionOutcome is the terminal reason recorded by an atomic consume.
// Pending is represented by an empty outcome and a nil ConsumedAt.
type InteractionOutcome string

const (
	InteractionOutcomeApproved InteractionOutcome = "approved"
	InteractionOutcomeDenied   InteractionOutcome = "denied"
	InteractionOutcomeRejected InteractionOutcome = "rejected"
)

func (o InteractionOutcome) Valid() bool {
	switch o {
	case InteractionOutcomeApproved, InteractionOutcomeDenied, InteractionOutcomeRejected:
		return true
	default:
		return false
	}
}

// InteractionRecord is the server-owned continuation for one validated
// authorization request. CanonicalRequest contains validated public protocol
// parameters, never credentials or raw browser/session/interaction handles.
type InteractionRecord struct {
	IDHash             []byte
	CanonicalRequest   map[string][]string
	RequestDigest      []byte
	ClientID           string
	RedirectURI        string
	RequiredActions    InteractionRequiredAction
	BrowserBindingHash []byte
	SessionIDHash      []byte
	BrowserContextHash []byte
	GenerationHash     []byte
	// DeviceUserCodeHash binds an RFC 8628 browser verification continuation to
	// a pending device grant. It is empty for ordinary OAuth authorization
	// interactions; raw user codes never enter this record.
	DeviceUserCodeHash []byte
	CreatedAt          time.Time
	ExpiresAt          time.Time
	ConsumedAt         *time.Time
	Outcome            InteractionOutcome
}

// DeviceGrantStatus is the durable state of an RFC 8628 authorization. Expiry
// is derived from DeviceGrant.ExpiresAt at the transaction clock, rather than
// persisted as a mutable status.
type DeviceGrantStatus string

const (
	DeviceGrantPending  DeviceGrantStatus = "pending"
	DeviceGrantApproved DeviceGrantStatus = "approved"
	DeviceGrantDenied   DeviceGrantStatus = "denied"
	DeviceGrantConsumed DeviceGrantStatus = "consumed"
)

func (s DeviceGrantStatus) Valid() bool {
	switch s {
	case DeviceGrantPending, DeviceGrantApproved, DeviceGrantDenied, DeviceGrantConsumed:
		return true
	default:
		return false
	}
}

// DeviceGrant holds only hashed protocol credentials. DeviceCodeHash and
// UserCodeHash are domain-separated keyed hashes; raw device and user codes
// must never enter this type, persistence, audit, or metrics.
type DeviceGrant struct {
	ID              string
	DeviceCodeHash  []byte
	UserCodeHash    []byte
	ClientID        string
	RequestedScopes []string
	ApprovedScopes  []string
	// RequestedAudiences and ApprovedAudiences are the resource indicators
	// accepted at device authorization and later approved with the device
	// grant. Keeping both makes the durable consent decision explicit.
	RequestedAudiences    []string
	ApprovedAudiences     []string
	Status                DeviceGrantStatus
	UserID                string
	Subject               string
	AuthTime              time.Time
	AuthenticationMethods []string
	CreatedAt             time.Time
	ExpiresAt             time.Time
	PollInterval          time.Duration
	NextPollAt            time.Time
	SlowDownCount         uint32
	DecidedAt             *time.Time
	ConsumedAt            *time.Time
	Version               uint64
}

// ValidateForCreate ensures the store is given a complete pending record. It
// deliberately does not accept any terminal state from a caller; named store
// transitions own all later status changes.
func (g DeviceGrant) ValidateForCreate() error {
	if g.ID == "" || len(g.DeviceCodeHash) == 0 || len(g.UserCodeHash) == 0 || g.ClientID == "" || g.Status != DeviceGrantPending || g.CreatedAt.IsZero() || g.ExpiresAt.IsZero() || g.PollInterval <= 0 || g.NextPollAt.IsZero() {
		return ErrInvalidDeviceGrant
	}
	if !g.CreatedAt.Before(g.ExpiresAt) {
		return ErrInvalidDeviceGrant
	}
	return nil
}

// DevicePollOutcome is deliberately protocol-neutral. The endpoint and Fosite
// handler map these durable outcomes to RFC 8628 error responses later.
type DevicePollOutcome string

const (
	DevicePollPending  DevicePollOutcome = "pending"
	DevicePollSlowDown DevicePollOutcome = "slow_down"
	DevicePollApproved DevicePollOutcome = "approved"
	DevicePollDenied   DevicePollOutcome = "denied"
	DevicePollExpired  DevicePollOutcome = "expired"
	DevicePollConsumed DevicePollOutcome = "consumed"
)

type DevicePollRequest struct {
	DeviceCodeHash []byte
	ClientID       string
	Now            time.Time
}

type DevicePollResult struct {
	Outcome DevicePollOutcome
	Grant   DeviceGrant
}

type DeviceGrantDecision string

const (
	DeviceGrantApprove DeviceGrantDecision = "approve"
	DeviceGrantDeny    DeviceGrantDecision = "deny"
)

func (d DeviceGrantDecision) Valid() bool {
	return d == DeviceGrantApprove || d == DeviceGrantDeny
}

type DeviceDecisionRequest struct {
	UserCodeHash          []byte
	Decision              DeviceGrantDecision
	UserID                string
	Subject               string
	AuthTime              time.Time
	AuthenticationMethods []string
	ApprovedScopes        []string
	ApprovedAudiences     []string
	Now                   time.Time
}

type DeviceConsumeRequest struct {
	DeviceCodeHash []byte
	ClientID       string
	Now            time.Time
}

// SigningKey is a persisted signing key plus lifecycle metadata.
type SigningKey struct {
	ID            string
	Algorithm     string
	PrivateKeyPEM []byte
	CreatedAt     time.Time
	NotBefore     time.Time
	NotAfter      time.Time
	Active        bool
}
