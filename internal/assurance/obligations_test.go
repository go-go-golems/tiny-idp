package assurance

import (
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestInteractionObligationCodecIsLossless(t *testing.T) {
	input := idpstore.InteractionRequireLogin | idpstore.InteractionRequireConsent | idpstore.InteractionRequireRegistration
	obligations, err := ObligationsFromActions(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []ObligationID{ObligationLogin, ObligationConsent, ObligationRegistration}
	if len(obligations) != len(want) {
		t.Fatalf("obligations=%v", obligations)
	}
	for index := range want {
		if obligations[index] != want[index] {
			t.Fatalf("obligations=%v want=%v", obligations, want)
		}
	}
	roundTrip, err := ActionsFromObligations(obligations)
	if err != nil || roundTrip != input {
		t.Fatalf("round trip actions=%d err=%v", roundTrip, err)
	}
}

func TestInteractionObligationCodecFailsClosed(t *testing.T) {
	if _, err := ObligationsFromActions(idpstore.InteractionRequiredAction(1 << 31)); err == nil {
		t.Fatal("unknown action bit accepted")
	}
	if _, err := ActionsFromObligations([]ObligationID{ObligationLogin, ObligationLogin}); err == nil {
		t.Fatal("duplicate obligation accepted")
	}
	if _, err := ActionsFromObligations([]ObligationID{"unknown.required@v1"}); err == nil {
		t.Fatal("unknown obligation accepted")
	}
}
