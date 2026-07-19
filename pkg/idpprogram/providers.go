package idpprogram

// ProviderKind names a typed virtual-resource family. A provider is not an
// ambient service locator: it names only the callbacks and native contract the
// host may invoke.
type ProviderKind string

const (
	ProviderKindIdentity   ProviderKind = "identity"
	ProviderKindInvitation ProviderKind = "invitation"
)

func (k ProviderKind) Valid() bool {
	return k == ProviderKindIdentity || k == ProviderKindInvitation
}

// ProviderState explains whether the provider has native durable state. It is
// descriptive activation metadata and is intentionally included in the
// fingerprinted program artifact so explain/operations cannot guess it.
type ProviderState string

const (
	ProviderStateVirtual ProviderState = "virtual"
	ProviderStateDurable ProviderState = "durable"
)

func (s ProviderState) Valid() bool {
	return s == ProviderStateVirtual || s == ProviderStateDurable
}

// ReplayProtection says where a provider prevents reuse of accepted evidence.
// Stateless signed evidence has expiry/audience checks but cannot promise
// one-use redemption without native durable state.
type ReplayProtection string

const (
	ReplayProtectionNone    ReplayProtection = "none"
	ReplayProtectionExpiry  ReplayProtection = "expiry"
	ReplayProtectionOneTime ReplayProtection = "one_time"
)

func (r ReplayProtection) Valid() bool {
	return r == ReplayProtectionNone || r == ReplayProtectionExpiry || r == ReplayProtectionOneTime
}

// RevocationMode documents whether accepted evidence can be withdrawn before
// expiry. The host must enforce the stated mode; JavaScript cannot override it.
type RevocationMode string

const (
	RevocationNone    RevocationMode = "none"
	RevocationKeyRoll RevocationMode = "key_rollover"
	RevocationDurable RevocationMode = "durable"
)

func (r RevocationMode) Valid() bool {
	return r == RevocationNone || r == RevocationKeyRoll || r == RevocationDurable
}

const (
	IdentityEstablishHandler  = "establish"
	InvitationValidateHandler = "validate"
)

// Provider is the immutable artifact contract for one identity or invitation
// resource. ID is a stable operator-selected identifier, normally
// "kind.name". Handlers reference named provider lambdas rather than storing
// Goja functions or host service handles.
type Provider struct {
	ID               string                     `json:"id"`
	Kind             ProviderKind               `json:"kind"`
	Version          uint32                     `json:"version"`
	State            ProviderState              `json:"state"`
	ReplayProtection ReplayProtection           `json:"replayProtection"`
	Revocation       RevocationMode             `json:"revocation"`
	Handlers         map[string]ProviderHandler `json:"handlers"`
}

// ProviderHandler pins both the JavaScript callback and its schema boundary.
// The duplicate schema names are deliberate: compilation can reject a handler
// changed independently of the provider contract before a request is served.
type ProviderHandler struct {
	ID           string `json:"id"`
	LambdaID     string `json:"lambdaId"`
	InputSchema  string `json:"inputSchema"`
	OutputSchema string `json:"outputSchema"`
}
