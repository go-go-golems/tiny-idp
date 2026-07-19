package fositeadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

type recordingPresentationPolicy struct {
	input  idp.PresentationInput
	output idp.PresentationOutput
	err    error
}

func (p *recordingPresentationPolicy) Present(_ context.Context, input idp.PresentationInput) (idp.PresentationOutput, error) {
	p.input = input.Clone()
	return p.output, p.err
}

func TestPresentationPolicyDecoratesOnlyProviderOwnedInteractionMetadata(t *testing.T) {
	policy := &recordingPresentationPolicy{output: idp.PresentationOutput{DocumentTitle: "Choose your workspace account"}}
	provider := &Provider{presentation: policy}
	page := &idpui.InteractionPage{DocumentTitle: "Choose an account", ClientID: "message-app", AccountChooser: &idpui.AccountChooserPrompt{AccountField: idpui.AccountFieldName, Entries: []idpui.AccountChooserEntry{{Value: "opaque-entry", Label: "Ada"}}}}
	require.NoError(t, provider.decorateInteractionPage(context.Background(), page))
	assert.Equal(t, "Choose your workspace account", page.DocumentTitle)
	assert.Equal(t, idp.PresentationAccountSelection, policy.input.Kind)
	assert.Equal(t, "message-app", policy.input.ClientID)
	assert.Equal(t, 1, policy.input.AccountCount)
	assert.Empty(t, policy.input.RequestedScope)
}

func TestPresentationPolicyDecoratesConsentAndDeviceWithPublicContextOnly(t *testing.T) {
	policy := &recordingPresentationPolicy{output: idp.PresentationOutput{DocumentTitle: "Review access"}}
	provider := &Provider{presentation: policy}
	consent := &idpui.InteractionPage{DocumentTitle: "Approve", Consent: &idpui.ConsentPrompt{ClientID: "message-app", Scopes: []idpui.Scope{{Name: "openid"}, {Name: "profile"}}}}
	require.NoError(t, provider.decorateInteractionPage(context.Background(), consent))
	assert.Equal(t, idp.PresentationConsent, policy.input.Kind)
	assert.Equal(t, []string{"openid", "profile"}, policy.input.RequestedScope)

	device := &idpui.DeviceVerificationPage{DocumentTitle: "Approve device", Confirmation: &idpui.DeviceConfirmationPrompt{ClientID: "coding-agent", Scopes: []idpui.Scope{{Name: "openid"}}}}
	require.NoError(t, provider.decorateDeviceVerificationPage(context.Background(), device))
	assert.Equal(t, "Review access", device.DocumentTitle)
	assert.Equal(t, idp.PresentationDeviceVerify, policy.input.Kind)
	assert.Equal(t, "coding-agent", policy.input.ClientID)
	assert.Equal(t, []string{"openid"}, policy.input.RequestedScope)
}

func TestPresentationPolicyFailureIsReturnedToNativeRenderer(t *testing.T) {
	provider := &Provider{presentation: &recordingPresentationPolicy{err: errors.New("synthetic failure")}}
	page := &idpui.InteractionPage{DocumentTitle: "Approve", Consent: &idpui.ConsentPrompt{ClientID: "message-app"}}
	assert.Error(t, provider.decorateInteractionPage(context.Background(), page))
}
