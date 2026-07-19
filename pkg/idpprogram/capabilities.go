package idpprogram

// CapabilityRequirement names a host-supplied capability contract. Lambdas
// receive only capabilities declared both by the program and by their spec.
type CapabilityRequirement struct {
	ID      string `json:"id"`
	Version uint32 `json:"version"`
}

// EffectKind is a stable identifier for an observable or mutating native
// effect. These identifiers are contracts and therefore must not be generated
// from Go type names.
type EffectKind string

const (
	EffectRead                     EffectKind = "read"
	EffectCreateLocalIdentity      EffectKind = "createLocalIdentity"
	EffectAttachPasswordCredential EffectKind = "attachPasswordCredential"
	EffectConsumeInvitation        EffectKind = "consumeInvitation"
	EffectEstablishBrowserSession  EffectKind = "establishBrowserSession"
	EffectEstablishVirtualIdentity EffectKind = "establishVirtualIdentity"
	EffectSendEmailChallenge       EffectKind = "sendEmailChallenge"
)

// Valid reports whether e belongs to the design-03 effect vocabulary.
func (e EffectKind) Valid() bool {
	switch e {
	case EffectRead,
		EffectCreateLocalIdentity,
		EffectAttachPasswordCredential,
		EffectConsumeInvitation,
		EffectEstablishBrowserSession,
		EffectEstablishVirtualIdentity,
		EffectSendEmailChallenge:
		return true
	default:
		return false
	}
}
