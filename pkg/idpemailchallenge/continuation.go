package idpemailchallenge

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
)

const EvidenceKind = "verified_email"

// BindingsFromContinuation derives the complete verification binding only
// from an already-loaded continuation. Browser input cannot select a client,
// generation, resume handler, or binding hash.
func BindingsFromContinuation(c idpcontinuation.WorkflowContinuation) VerificationBindings {
	return VerificationBindings{WorkflowID: c.WorkflowID, ResumeHandlerID: c.ResumeHandlerID, ProgramFingerprint: c.ProgramFingerprint, ClientID: c.ClientID, ClientGeneration: c.ClientGeneration, BrowserBindingHash: append([]byte(nil), c.BrowserBindingHash...)}
}

// EvidenceProjection creates the only JSON value passed to ctx.evidence.email.
func EvidenceProjection(e VerifiedEmailEvidence) (map[string]json.RawMessage, error) {
	if e.Version != RecordVersionV1 || !valid(e.ChallengeID) || !valid(e.Address) || e.Method != "email_code" || e.VerifiedAt.IsZero() {
		return nil, errors.New("verified email evidence is invalid")
	}
	raw, err := json.Marshal(map[string]any{"address": e.Address, "verified": true, "method": e.Method, "verifiedAt": e.VerifiedAt.UTC().Format("2006-01-02T15:04:05Z07:00")})
	if err != nil {
		return nil, errors.Wrap(err, "encode verified email evidence")
	}
	return map[string]json.RawMessage{"email": raw}, nil
}
