package fositeadapter

import (
	"encoding/json"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

func TestSignupInvitationInputRequiresMatchingConsumptionEffect(t *testing.T) {
	tests := []struct {
		name               string
		input              json.RawMessage
		effects            []idpprogram.EffectPlan
		wantRequiresInvite bool
		wantValid          bool
	}{
		{name: "open signup", input: json.RawMessage(`{"email":"open@example.test"}`), effects: signupEffects(false), wantValid: true},
		{name: "invited signup consumes", input: json.RawMessage(`{"email":"invited@example.test","inviteCode":"one-time-code"}`), effects: signupEffects(true), wantRequiresInvite: true, wantValid: true},
		{name: "invited signup cannot omit consumption", input: json.RawMessage(`{"email":"invited@example.test","inviteCode":"one-time-code"}`), effects: signupEffects(false), wantRequiresInvite: true, wantValid: false},
		{name: "open signup cannot invent consumption", input: json.RawMessage(`{"email":"open@example.test"}`), effects: signupEffects(true), wantValid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requiresInvite, err := signupInputRequiresInvitationConsumption(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if requiresInvite != tt.wantRequiresInvite {
				t.Fatalf("requires invitation=%v want %v", requiresInvite, tt.wantRequiresInvite)
			}
			expectedCount := 2
			if requiresInvite {
				expectedCount = 3
			}
			if got := validSignupEffectSequence(tt.effects, expectedCount, requiresInvite); got != tt.wantValid {
				t.Fatalf("valid effect sequence=%v want %v", got, tt.wantValid)
			}
		})
	}
}

func signupEffects(withInvitation bool) []idpprogram.EffectPlan {
	effects := []idpprogram.EffectPlan{{Kind: idpprogram.EffectCreateLocalIdentity}, {Kind: idpprogram.EffectAttachPasswordCredential}}
	if withInvitation {
		effects = append(effects, idpprogram.EffectPlan{Kind: idpprogram.EffectConsumeInvitation})
	}
	return effects
}
