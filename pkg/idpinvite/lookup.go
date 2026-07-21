package idpinvite

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

const (
	// LookupCapabilityID is the only native read operation granted to a durable
	// invitation provider lambda. Its namespace becomes ctx.cap.invitation.lookup.
	LookupCapabilityID      = "invitation.lookup"
	LookupCapabilityVersion = uint32(1)
)

type lookupRequest struct {
	Code string `json:"code"`
}

// LookupDecision is intentionally coarse. Expected lifecycle failures all
// become Valid=false so JavaScript cannot turn TinyIDP into a token-state
// oracle. Valid records expose only redacted evidence.
type LookupDecision struct {
	Valid         bool       `json:"valid"`
	InvitationID  string     `json:"invitationId,omitempty"`
	PolicyVersion string     `json:"policyVersion,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
}

// NewLookupCapability binds one client audience and one native clock to a
// durable service. Audience is never accepted from JavaScript.
func NewLookupCapability(service *DurableService, audience string, now func() time.Time) (idpscript.CapabilityBinding, error) {
	if service == nil || !validText(audience) {
		return idpscript.CapabilityBinding{}, errors.New("durable invitation lookup service and audience are required")
	}
	if now == nil {
		now = time.Now
	}
	return idpscript.CapabilityBinding{
		Requirement:    idpprogram.CapabilityRequirement{ID: LookupCapabilityID, Version: LookupCapabilityVersion},
		MaxInputBytes:  1024,
		MaxOutputBytes: 1024,
		Invoke: func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
			var request lookupRequest
			if err := decodeExact(raw, &request); err != nil || !validText(request.Code) || len(request.Code) > 512 {
				return nil, errors.New("durable invitation lookup request is invalid")
			}
			inspection, err := service.Inspect(ctx, request.Code, audience, now().UTC())
			if err != nil {
				if stderrors.Is(err, idpstore.ErrNotFound) || stderrors.Is(err, idpstore.ErrExpired) || stderrors.Is(err, idpstore.ErrInvitationRevoked) || stderrors.Is(err, idpstore.ErrAlreadyConsumed) {
					return json.Marshal(LookupDecision{Valid: false})
				}
				return nil, errors.Wrap(err, "inspect durable invitation")
			}
			expiresAt := inspection.ExpiresAt
			return json.Marshal(LookupDecision{Valid: true, InvitationID: inspection.InvitationID, PolicyVersion: inspection.PolicyVersion, ExpiresAt: &expiresAt})
		},
	}, nil
}
