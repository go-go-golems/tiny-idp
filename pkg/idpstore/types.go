package idpstore

import "time"

// Mode controls validation strictness. Development mode permits loopback HTTP
// issuers and memory stores; production mode fails closed.
type Mode string

const (
	DevMode        Mode = "dev"
	ProductionMode Mode = "production"
)

// Client is a configured relying party/OAuth client.
type Client struct {
	ID                     string
	SecretHash             []byte
	Public                 bool
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	AllowedScopes          []string
	RequirePKCE            bool
	AccessTokenTTL         time.Duration
	IDTokenTTL             time.Duration
	RefreshTokenTTL        time.Duration
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Disabled               bool
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
	MustChangeAtLogin bool
	Disabled          bool
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
