package idpprogram_test

import (
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExplainProvidersStatesMaterializationReplayAndRevocation(t *testing.T) {
	explained := idpprogram.ExplainProviders(idpprogram.Program{Providers: map[string]idpprogram.Provider{
		"invitation.durable": {ID: "invitation.durable", Kind: idpprogram.ProviderKindInvitation, Version: 1, State: idpprogram.ProviderStateDurable, ReplayProtection: idpprogram.ReplayProtectionOneTime, Revocation: idpprogram.RevocationDurable},
		"identity.email":     {ID: "identity.email", Kind: idpprogram.ProviderKindIdentity, Version: 1, State: idpprogram.ProviderStateVirtual, ReplayProtection: idpprogram.ReplayProtectionExpiry, Revocation: idpprogram.RevocationKeyRoll},
	}})
	assert.Equal(t, []string{"identity.email", "invitation.durable"}, []string{explained[0].ID, explained[1].ID})
	assert.Equal(t, "no local user row", explained[0].Materialization)
	assert.Equal(t, idpprogram.ReplayProtectionOneTime, explained[1].ReplayProtection)
	assert.Equal(t, idpprogram.RevocationDurable, explained[1].Revocation)
	assert.Equal(t, []idpprogram.EffectKind{idpprogram.EffectConsumeInvitation}, explained[1].NativeEffects)
}
