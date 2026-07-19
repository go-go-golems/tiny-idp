package fositeadapter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/ory/fosite"
	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func (p *Provider) beginScriptedSignup(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, client idpstore.Client, interactionHandle, csrfToken string) error {
	if p.scriptedSignup == nil || p.workflowContinuations == nil {
		return errors.New("scripted signup is unavailable")
	}
	presentation, err := p.scriptedSignup.Start(r.Context())
	if err != nil {
		return err
	}
	record, err := p.store.GetInteraction(r.Context(), idpstore.HashSecret(p.csrfKey, interactionHandle))
	if err != nil {
		return errors.Wrap(err, "load signup authorization interaction")
	}
	workflow := p.scriptedSignup.Program().Workflows[idpsignup.WorkflowID]
	publicValues, err := json.Marshal(presentation.Presentation.PublicValues)
	if err != nil {
		return errors.Wrap(err, "encode signup public values")
	}
	continuationHandle, _, err := p.workflowContinuations.Create(r.Context(), idpcontinuation.WorkflowContinuation{
		WorkflowID: idpsignup.WorkflowID, ResumeHandlerID: presentation.Presentation.ResumeHandler,
		ProgramFingerprint: p.scriptedSignup.Fingerprint(), SchemaVersion: "v1", WorkflowVersion: workflow.Version,
		RequestDigest: record.RequestDigest, ClientID: record.ClientID, RedirectURI: record.RedirectURI,
		ClientGeneration: hex.EncodeToString(record.GenerationHash), BrowserBindingHash: record.BrowserBindingHash,
		SessionIDHash: record.SessionIDHash, BrowserContextHash: record.BrowserContextHash,
		Presentation: idpcontinuation.PresentationState{ID: "signup", Fields: fieldIDs(presentation.Fields), AllowedActions: actionIDs(presentation.Actions), PublicValues: publicValues},
		InputSchema:  presentation.InputSchema, Carry: presentation.Presentation.Carry,
		ExpiresAt: p.now().Add(presentation.Presentation.ExpiresIn),
	})
	if err != nil {
		return errors.Wrap(err, "create signup workflow continuation")
	}
	p.renderWorkflow(w, r, http.StatusOK, workflowPage(p, record, interactionHandle, csrfToken, continuationHandle, presentation.Fields, presentation.Actions, presentation.Presentation.PublicValues, nil))
	return nil
}

func (p *Provider) resumeScriptedSignup(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, client idpstore.Client, record idpstore.InteractionRecord, interactionHandle, clientAddress string) {
	continuationHandle := r.PostForm.Get(idpui.WorkflowContinuationFieldName)
	continuation, err := p.workflowContinuations.Load(r.Context(), continuationHandle, p.signupBindings(record, r))
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "workflow.signup.resume_rejected", ClientID: record.ClientID, Result: "rejected", Reason: "continuation_unavailable"})
		http.Error(w, "registration request was not accepted", http.StatusBadRequest)
		return
	}
	fields, actions, err := workflowDescriptors(continuation.Presentation)
	if err != nil {
		http.Error(w, "registration request was not accepted", http.StatusBadRequest)
		return
	}
	submission, err := idpworkflow.ParseSubmission(fields, actions, r.PostForm)
	if err != nil || submission.Interaction != interactionHandle || submission.Continuation != continuationHandle {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, nil)
		return
	}
	defer submission.DestroySecrets()
	if submission.Action == idpworkflow.ActionDeny {
		if _, err := p.workflowContinuations.Consume(r.Context(), continuationHandle, continuation.Revision, p.signupBindings(record, r), idpcontinuation.TerminalOutcome{Kind: idpcontinuation.TerminalDeny}); err != nil {
			http.Error(w, "registration request was not accepted", http.StatusBadRequest)
			return
		}
		if _, err := p.store.ConsumeInteraction(r.Context(), record.IDHash, p.now(), idpstore.InteractionOutcomeDenied); err != nil {
			http.Error(w, "authorization interaction already completed", http.StatusBadRequest)
			return
		}
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrAccessDenied)
		return
	}
	input, err := json.Marshal(map[string]string{"displayName": submission.PublicValues[idpworkflow.FieldDisplayName], "email": submission.PublicValues[idpworkflow.FieldEmail]})
	if err != nil || p.workflowContinuations.ValidateResumeInput(r.Context(), continuation, input) != nil {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	outcome, err := p.scriptedSignup.Submit(r.Context(), submission.PublicValues, map[string]idpworkflow.SecretHandle{
		"password": submission.Secrets[idpworkflow.FieldPassword], "passwordConfirmation": submission.Secrets[idpworkflow.FieldPasswordConfirmation],
	})
	if err != nil || outcome.Kind != idpprogram.OutcomeCommit {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	registered, err := p.commitScriptedSignup(r.Context(), outcome, submission, clientAddress, record.ClientID)
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "account.self_registration", ClientID: record.ClientID, Result: "rejected", Reason: "registration_rejected"})
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	if _, err := p.workflowContinuations.Consume(r.Context(), continuationHandle, continuation.Revision, p.signupBindings(record, r), idpcontinuation.TerminalOutcome{Kind: idpcontinuation.TerminalComplete}); err != nil {
		http.Error(w, "registration request was not accepted", http.StatusBadRequest)
		return
	}
	p.completeScriptedSignup(w, r, ar, client, record, registered)
}

func (p *Provider) completeScriptedSignup(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, client idpstore.Client, record idpstore.InteractionRecord, registered idpstore.User) {
	sessionHash, err := p.createBrowserSession(w, r, registered, p.now())
	if err != nil {
		http.Error(w, "create session failed", http.StatusInternalServerError)
		return
	}
	p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "account.self_registration", ClientID: record.ClientID, Subject: registered.Sub, Result: "accepted"})
	requireConsent, err := p.consent.RequireConsent(r.Context(), registered, client, []string(ar.GetRequestedScopes()))
	if err != nil {
		http.Error(w, "consent policy failed", http.StatusInternalServerError)
		return
	}
	if _, err := p.store.ConsumeInteraction(r.Context(), record.IDHash, p.now(), idpstore.InteractionOutcomeApproved); err != nil {
		http.Error(w, "authorization interaction already completed", http.StatusBadRequest)
		return
	}
	if requireConsent {
		consentHandle, consentCSRF, err := p.createInteractionForSession(w, r, ar, idpstore.InteractionRequireConsent, sessionHash)
		if err != nil {
			http.Error(w, "create consent interaction failed", http.StatusInternalServerError)
			return
		}
		page := p.newInteractionPage(consentHandle, consentCSRF, idpstore.InteractionRequireConsent, nil, true, client.ID, []string(ar.GetRequestedScopes()), "", nil)
		p.renderInteraction(w, r, http.StatusOK, page)
		return
	}
	p.finishAuthorize(w, r, ar, registered, p.now(), false, nil)
}

func (p *Provider) commitScriptedSignup(ctx context.Context, outcome idpprogram.Outcome, submission idpworkflow.Submission, clientAddress, clientID string) (idpstore.User, error) {
	if len(outcome.Effects) != 2 || outcome.Effects[0].Kind != idpprogram.EffectCreateLocalIdentity || outcome.Effects[1].Kind != idpprogram.EffectAttachPasswordCredential {
		return idpstore.User{}, errors.New("signup script emitted an invalid effect sequence")
	}
	var identity struct {
		Login       string `json:"login"`
		DisplayName string `json:"displayName"`
	}
	var credential struct {
		PasswordHandle             string `json:"passwordHandle"`
		PasswordConfirmationHandle string `json:"passwordConfirmationHandle"`
	}
	if err := json.Unmarshal(outcome.Effects[0].Payload, &identity); err != nil {
		return idpstore.User{}, errors.Wrap(err, "decode signup identity effect")
	}
	if err := json.Unmarshal(outcome.Effects[1].Payload, &credential); err != nil {
		return idpstore.User{}, errors.Wrap(err, "decode signup credential effect")
	}
	password, confirmation, ok := submissionSecrets(submission, credential.PasswordHandle, credential.PasswordConfirmationHandle)
	if !ok || len(password) == 0 || !equalBytes(password, confirmation) || !p.allowRegistration(ctx, clientID, clientAddress, identity.Login) {
		return idpstore.User{}, errors.New("signup effects are not acceptable")
	}
	defer clearBytes(password)
	defer clearBytes(confirmation)
	return p.registration.Create(ctx, idpaccounts.CreateRequest{Login: identity.Login, Name: identity.DisplayName, Password: password, Email: identity.Login})
}

func submissionSecrets(submission idpworkflow.Submission, passwordToken, confirmationToken string) ([]byte, []byte, bool) {
	var password, confirmation []byte
	for _, handle := range submission.Secrets {
		if handle.Token() == passwordToken {
			password, _ = submission.ResolveSecret(handle)
		}
		if handle.Token() == confirmationToken {
			confirmation, _ = submission.ResolveSecret(handle)
		}
	}
	return password, confirmation, password != nil && confirmation != nil
}

func (p *Provider) renderScriptedSignupError(w http.ResponseWriter, r *http.Request, record idpstore.InteractionRecord, interactionHandle, continuationHandle string, fields []idpworkflow.FieldDescriptor, actions []idpworkflow.ActionDescriptor, values map[idpworkflow.FieldID]string) {
	p.renderWorkflow(w, r, http.StatusBadRequest, workflowPage(p, record, interactionHandle, r.PostForm.Get(idpui.CSRFFieldName), continuationHandle, fields, actions, values, []idpui.WorkflowFieldError{{Field: idpworkflow.FieldEmail, Code: idpworkflow.ErrorRejected}}))
}

func (p *Provider) signupBindings(record idpstore.InteractionRecord, r *http.Request) idpcontinuation.Bindings {
	return idpcontinuation.Bindings{WorkflowID: idpsignup.WorkflowID, ClientID: record.ClientID, RedirectURI: record.RedirectURI, ClientGeneration: hex.EncodeToString(record.GenerationHash), ProgramFingerprint: p.scriptedSignup.Fingerprint(), RequestDigest: record.RequestDigest, BrowserBindingHash: p.browserBindingHash(r), SessionIDHash: p.browserSessionHash(r), BrowserContextHash: p.browserContextHash(r)}
}

func workflowPage(p *Provider, record idpstore.InteractionRecord, interactionHandle, csrfToken, continuationHandle string, fields []idpworkflow.FieldDescriptor, actions []idpworkflow.ActionDescriptor, values map[idpworkflow.FieldID]string, fieldErrors []idpui.WorkflowFieldError) idpui.WorkflowPage {
	page := idpui.WorkflowPage{DocumentTitle: "Create an account", Form: idpui.WorkflowForm{ActionURL: p.issuer.Endpoint("/authorize"), RedirectOrigin: interactionRedirectOrigin(record.RedirectURI), InteractionField: idpui.InteractionFieldName, Interaction: interactionHandle, ContinuationField: idpui.WorkflowContinuationFieldName, Continuation: continuationHandle, CSRFField: idpui.CSRFFieldName, CSRFToken: csrfToken, ActionField: idpui.ActionFieldName}, Errors: fieldErrors}
	for _, field := range fields {
		page.Fields = append(page.Fields, idpui.WorkflowField{Descriptor: field, Value: values[field.ID]})
	}
	for _, action := range actions {
		page.Actions = append(page.Actions, idpui.WorkflowAction{Descriptor: action})
	}
	return page
}

func workflowDescriptors(state idpcontinuation.PresentationState) ([]idpworkflow.FieldDescriptor, []idpworkflow.ActionDescriptor, error) {
	registry := idpworkflow.DefaultRegistry()
	fields := make([]idpworkflow.FieldDescriptor, 0, len(state.Fields))
	for _, raw := range state.Fields {
		field, ok := registry.Field(idpworkflow.FieldID(raw))
		if !ok {
			return nil, nil, errors.New("workflow continuation has unknown field")
		}
		fields = append(fields, field)
	}
	actions := make([]idpworkflow.ActionDescriptor, 0, len(state.AllowedActions))
	for _, raw := range state.AllowedActions {
		action, ok := registry.Action(idpworkflow.ActionID(raw))
		if !ok {
			return nil, nil, errors.New("workflow continuation has unknown action")
		}
		actions = append(actions, action)
	}
	return fields, actions, nil
}

func fieldIDs(fields []idpworkflow.FieldDescriptor) []string {
	ids := make([]string, 0, len(fields))
	for _, field := range fields {
		ids = append(ids, string(field.ID))
	}
	return ids
}
func actionIDs(actions []idpworkflow.ActionDescriptor) []string {
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		ids = append(ids, string(action.ID))
	}
	return ids
}
