package idpprogram

import "sort"

// ProviderExplanation is a stable operator-facing projection of a compiled
// provider contract. It contains no callbacks, data, or credentials.
type ProviderExplanation struct {
	ID               string
	Kind             ProviderKind
	Version          uint32
	State            ProviderState
	ReplayProtection ReplayProtection
	Revocation       RevocationMode
	Materialization  string
	NativeEffects    []EffectKind
}

// ExplainProviders produces deterministic descriptions suitable for CLI and
// activation diagnostics. A virtual identity has no local account row; all
// other provider state is described directly from the fingerprinted contract.
func ExplainProviders(program Program) []ProviderExplanation {
	ids := make([]string, 0, len(program.Providers))
	for id := range program.Providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]ProviderExplanation, 0, len(ids))
	for _, id := range ids {
		provider := program.Providers[id]
		explanation := ProviderExplanation{ID: provider.ID, Kind: provider.Kind, Version: provider.Version, State: provider.State, ReplayProtection: provider.ReplayProtection, Revocation: provider.Revocation, Materialization: "native durable state"}
		if provider.Kind == ProviderKindIdentity && provider.State == ProviderStateVirtual {
			explanation.Materialization = "no local user row"
			explanation.NativeEffects = []EffectKind{EffectEstablishVirtualIdentity}
		}
		if provider.Kind == ProviderKindIdentity && provider.State == ProviderStateDurable {
			explanation.Materialization = "local stored identity"
			explanation.NativeEffects = []EffectKind{EffectCreateLocalIdentity}
		}
		if provider.Kind == ProviderKindInvitation && provider.State == ProviderStateDurable {
			explanation.NativeEffects = []EffectKind{EffectConsumeInvitation}
		}
		result = append(result, explanation)
	}
	return result
}
