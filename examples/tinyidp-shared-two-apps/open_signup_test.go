package sharedtwoapps

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestDisplayNameCollisionPreservesInvitationInContinuation(t *testing.T) {
	source, err := os.ReadFile("open-signup.js")
	if err != nil {
		t.Fatal(err)
	}
	executor, err := idpsignup.New(context.Background(), string(source), 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = executor.Close(context.Background()) })
	requirement := idpprogram.CapabilityRequirement{ID: idpaccounts.DisplayNameLookupCapabilityID, Version: idpaccounts.DisplayNameLookupCapabilityVersion}
	capability := idpscript.CapabilityBinding{
		Requirement:    requirement,
		MaxInputBytes:  1024,
		MaxOutputBytes: 128,
		Invoke: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"available":false}`), nil
		},
	}
	input := json.RawMessage(`{"displayName":"Taken","email":"taken@example.test","inviteCode":"ONE-TIME-CODE"}`)
	outcome, err := executor.InvokeSubmissionWithCapabilities(context.Background(), idpsignup.SubmittedHandler, input, map[string]idpscript.CapabilityBinding{requirement.ID: capability}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	presentation, err := idpworkflow.DecodePresentation(outcome.Presentation)
	if err != nil {
		t.Fatal(err)
	}
	validated, err := idpworkflow.ValidatePresentation(executor.Program(), idpsignup.WorkflowID, idpsignup.SubmittedHandler, presentation, idpworkflow.DefaultRegistry(), idpworkflow.DefaultMaximumContinuationTTL)
	if err != nil {
		t.Fatal(err)
	}
	var carry struct {
		InviteCode string `json:"inviteCode"`
	}
	if err := json.Unmarshal(validated.Presentation.Carry, &carry); err != nil {
		t.Fatal(err)
	}
	if carry.InviteCode != "ONE-TIME-CODE" {
		t.Fatalf("continuation invite code = %q", carry.InviteCode)
	}
}
