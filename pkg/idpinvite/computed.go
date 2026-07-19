package idpinvite

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

const (
	// EligibilityCapabilityID is the only host operation exposed to a computed
	// invitation provider. Its dotted name deliberately becomes ctx.cap.invitation.eligibility.
	EligibilityCapabilityID      = "invitation.eligibility"
	EligibilityCapabilityVersion = uint32(1)
)

// EligibilityProbe is the complete, bounded request a provider may send to a
// host eligibility service. It is data, not a database handle or a network
// client. The host chooses how (or whether) to answer it.
type EligibilityProbe struct {
	Email      string `json:"email"`
	InviteCode string `json:"inviteCode,omitempty"`
	Audience   string `json:"audience"`
}

// EligibilityDecision is the complete, bounded answer that returns to the
// provider. Evidence is an opaque stable reference for audit logs; it never
// gives JavaScript authority to fetch the underlying record.
type EligibilityDecision struct {
	Accepted   bool   `json:"accepted"`
	Reason     string `json:"reason,omitempty"`
	EvidenceID string `json:"evidenceId,omitempty"`
}

// EligibilityEvaluator is host code. It can call a directory or database,
// but receives only a validated value object and returns only a value object.
// A program cannot retain this callback or invoke it outside its capability
// budget and invocation lifetime.
type EligibilityEvaluator func(context.Context, EligibilityProbe) (EligibilityDecision, error)

// NewEligibilityCapability binds one computed-invitation evaluator to the
// scripting runtime. It validates both directions at the native seam, keeping
// the provider's observable authority smaller than the host implementation.
func NewEligibilityCapability(evaluate EligibilityEvaluator) (idpscript.CapabilityBinding, error) {
	if evaluate == nil {
		return idpscript.CapabilityBinding{}, errors.New("invitation eligibility evaluator is required")
	}
	return idpscript.CapabilityBinding{
		Requirement:    idpprogram.CapabilityRequirement{ID: EligibilityCapabilityID, Version: EligibilityCapabilityVersion},
		MaxInputBytes:  1024,
		MaxOutputBytes: 1024,
		Invoke: func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
			var probe EligibilityProbe
			if err := decodeExact(raw, &probe); err != nil {
				return nil, errors.New("computed invitation probe is invalid")
			}
			if err := validateProbe(probe); err != nil {
				return nil, err
			}
			decision, err := evaluate(ctx, probe)
			if err != nil {
				return nil, errors.Wrap(err, "evaluate computed invitation eligibility")
			}
			if err := validateDecision(decision); err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(decision)
			if err != nil {
				return nil, errors.Wrap(err, "encode computed invitation decision")
			}
			return encoded, nil
		},
	}, nil
}

func decodeExact(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("multiple JSON values")
	}
	return nil
}

func validateProbe(probe EligibilityProbe) error {
	if !validText(probe.Email) || !strings.Contains(probe.Email, "@") || !validText(probe.Audience) || len(probe.InviteCode) > 256 {
		return errors.New("computed invitation probe is invalid")
	}
	return nil
}

func validateDecision(decision EligibilityDecision) error {
	if len(decision.Reason) > 128 || len(decision.EvidenceID) > 512 {
		return errors.New("computed invitation decision is invalid")
	}
	return nil
}
