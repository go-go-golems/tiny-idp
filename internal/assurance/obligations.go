package assurance

import (
	"sort"

	"github.com/pkg/errors"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

var obligationActions = map[ObligationID]idpstore.InteractionRequiredAction{
	ObligationLogin:            idpstore.InteractionRequireLogin,
	ObligationFreshLogin:       idpstore.InteractionRequireFreshLogin,
	ObligationConsent:          idpstore.InteractionRequireConsent,
	ObligationStepUp:           idpstore.InteractionRequireStepUp,
	ObligationAccountSelection: idpstore.InteractionRequireAccountSelection,
	ObligationRegistration:     idpstore.InteractionRequireRegistration,
}

const knownInteractionActions = idpstore.InteractionRequireLogin |
	idpstore.InteractionRequireFreshLogin |
	idpstore.InteractionRequireConsent |
	idpstore.InteractionRequireStepUp |
	idpstore.InteractionRequireAccountSelection |
	idpstore.InteractionRequireRegistration

// ObligationsFromActions converts the compact persisted representation into a
// deterministic set of stable IDs. It rejects future/unknown bits so a newer
// store cannot silently weaken an older assurance runtime.
func ObligationsFromActions(actions idpstore.InteractionRequiredAction) ([]ObligationID, error) {
	if actions&^knownInteractionActions != 0 {
		return nil, errors.Errorf("unknown interaction required-action bits %d", actions&^knownInteractionActions)
	}
	result := make([]ObligationID, 0, len(obligationActions))
	for obligation, action := range obligationActions {
		if actions.Has(action) {
			result = append(result, obligation)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

// ActionsFromObligations is the inverse codec. Duplicate and unknown IDs are
// rejected rather than normalized away, making malformed scenario/trace input
// observable to callers.
func ActionsFromObligations(obligations []ObligationID) (idpstore.InteractionRequiredAction, error) {
	var actions idpstore.InteractionRequiredAction
	seen := map[ObligationID]struct{}{}
	for _, obligation := range obligations {
		action, ok := obligationActions[obligation]
		if !ok {
			return 0, errors.Errorf("unknown interaction obligation %q", obligation)
		}
		if _, duplicate := seen[obligation]; duplicate {
			return 0, errors.Errorf("duplicate interaction obligation %q", obligation)
		}
		seen[obligation] = struct{}{}
		actions |= action
	}
	return actions, nil
}
