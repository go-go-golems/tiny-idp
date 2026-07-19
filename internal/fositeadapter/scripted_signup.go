package fositeadapter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
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
	record, err := p.store.GetInteraction(r.Context(), idpstore.HashSecret(p.csrfKey, interactionHandle))
	if err != nil {
		return errors.Wrap(err, "load signup authorization interaction")
	}
	presentation, err := p.scriptedSignup.Start(r.Context(), idpsignup.StartInput{
		ClientID:          client.ID,
		RedirectURI:       ar.GetRedirectURI().String(),
		RequestedScope:    strings.Join(ar.GetRequestedScopes(), " "),
		InteractionID:     hex.EncodeToString(record.IDHash),
		HasBrowserSession: len(record.SessionIDHash) != 0,
	})
	if err != nil {
		return err
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
	input, err := p.scriptedSignup.SubmissionInput(continuation.ResumeHandlerID, submission.PublicValues)
	if err != nil || p.workflowContinuations.ValidateResumeInput(r.Context(), continuation, input) != nil {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	outcome, err := p.scriptedSignup.InvokeSubmission(r.Context(), continuation.ResumeHandlerID, input, map[string]idpworkflow.SecretHandle{
		"password": submission.Secrets[idpworkflow.FieldPassword], "passwordConfirmation": submission.Secrets[idpworkflow.FieldPasswordConfirmation],
	}, nil)
	if err != nil {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	if outcome.Kind == idpprogram.OutcomeChallenge {
		if err := p.beginEmailChallenge(w, r, outcome, continuationHandle, continuation, record, interactionHandle); err != nil {
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
			return
		}
		return
	}
	if outcome.Kind != idpprogram.OutcomeCommit {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	registered, err := p.commitScriptedSignup(r.Context(), outcome, submission, continuation, p.signupBindings(record, r), record, clientAddress)
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "account.self_registration", ClientID: record.ClientID, Result: "rejected", Reason: signupCommitFailureReason(err)})
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	p.completeScriptedSignup(w, r, ar, client, record, registered)
}

func (p *Provider) beginEmailChallenge(w http.ResponseWriter, r *http.Request, outcome idpprogram.Outcome, handle string, current idpcontinuation.WorkflowContinuation, record idpstore.InteractionRecord, interactionHandle string) error {
	if p.emailChallenges == nil || outcome.Continuation == nil {
		return errors.New("email challenge is unavailable")
	}
	var request struct {
		Kind, Email, Template           string
		MaximumAttempts, MaximumResends int
	}
	if err := json.Unmarshal(outcome.Challenge, &request); err != nil || request.Kind != "email_code" || request.Email == "" || request.MaximumAttempts <= 0 || request.MaximumResends <= 0 {
		return errors.New("email challenge request is invalid")
	}
	workflow := p.scriptedSignup.Program().Workflows[idpsignup.WorkflowID]
	currentHandler := workflow.Handlers[current.ResumeHandlerID]
	inputSchema := ""
	for _, edge := range currentHandler.ContinuationEdges {
		if edge.OutcomeKind == idpprogram.OutcomeChallenge && edge.HandlerID == outcome.Continuation.HandlerID {
			inputSchema = edge.InputSchema
			break
		}
	}
	if inputSchema == "" {
		return errors.New("email challenge edge is not declared")
	}
	registry := idpworkflow.DefaultRegistry()
	code, _ := registry.Field(idpworkflow.FieldEmailCode)
	submit, _ := registry.Action(idpworkflow.ActionSubmit)
	deny, _ := registry.Action(idpworkflow.ActionDeny)
	next := idpcontinuation.WorkflowContinuation{ResumeHandlerID: outcome.Continuation.HandlerID, InputSchema: inputSchema, Carry: outcome.Continuation.Carry, Presentation: idpcontinuation.PresentationState{ID: "email-code", Fields: []string{string(idpworkflow.FieldEmailCode)}, AllowedActions: []string{string(idpworkflow.ActionSubmit), string(idpworkflow.ActionDeny)}, PublicValues: json.RawMessage(`{}`)}, ExpiresAt: p.now().Add(time.Duration(outcome.Continuation.ExpiresIn) * time.Second)}
	nextHandle, stored, err := p.workflowContinuations.Advance(r.Context(), handle, current.Revision, p.signupBindings(record, r), next)
	if err != nil {
		return err
	}
	_, err = p.emailChallenges.CreateAndSend(r.Context(), idpemailchallenge.CreateRequest{ID: nextHandle, Email: request.Email, Template: request.Template, Bindings: idpemailchallenge.BindingsFromContinuation(stored), ExpiresAt: stored.ExpiresAt, MaximumAttempts: uint32(request.MaximumAttempts), MaximumResends: uint32(request.MaximumResends)})
	if err != nil {
		return err
	}
	p.renderWorkflow(w, r, http.StatusOK, workflowPage(p, record, interactionHandle, r.PostForm.Get(idpui.CSRFFieldName), nextHandle, []idpworkflow.FieldDescriptor{code}, []idpworkflow.ActionDescriptor{submit, deny}, map[idpworkflow.FieldID]string{}, nil))
	return nil
}

func signupCommitFailureReason(err error) string {
	switch {
	case errors.Is(err, idpstore.ErrDuplicate):
		return "duplicate_login"
	case errors.Is(err, idp.ErrPasswordRejected):
		return "password_rejected"
	case errors.Is(err, idpcontinuation.ErrConflict), errors.Is(err, idpcontinuation.ErrAlreadyTerminal), errors.Is(err, idpstore.ErrAlreadyConsumed):
		return "state_conflict"
	case errors.Is(err, idpcontinuation.ErrExpired), errors.Is(err, idpcontinuation.ErrRevoked):
		return "continuation_rejected"
	default:
		return "registration_rejected"
	}
}

func (p *Provider) completeScriptedSignup(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, client idpstore.Client, record idpstore.InteractionRecord, registered signupCommitResult) {
	http.SetCookie(w, &http.Cookie{Name: p.sessionCookieName, Value: registered.SessionHandle, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: int(p.sessionTTL.Seconds())})
	p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "account.self_registration", ClientID: record.ClientID, Subject: registered.User.Sub, Result: "accepted"})
	requireConsent, err := p.consent.RequireConsent(r.Context(), registered.User, client, []string(ar.GetRequestedScopes()))
	if err != nil {
		http.Error(w, "consent policy failed", http.StatusInternalServerError)
		return
	}
	if requireConsent {
		consentHandle, consentCSRF, err := p.createInteractionForSession(w, r, ar, idpstore.InteractionRequireConsent, registered.SessionHash)
		if err != nil {
			http.Error(w, "create consent interaction failed", http.StatusInternalServerError)
			return
		}
		page := p.newInteractionPage(consentHandle, consentCSRF, idpstore.InteractionRequireConsent, nil, true, client.ID, []string(ar.GetRequestedScopes()), "", nil)
		p.renderInteraction(w, r, http.StatusOK, page)
		return
	}
	p.finishAuthorize(w, r, ar, registered.User, p.now(), false, nil)
}

type signupCommitResult struct {
	User          idpstore.User
	SessionHandle string
	SessionHash   []byte
}

// commitScriptedSignup is the sole native commit boundary for an approved
// signup effect plan. It commits the account, credential, browser session,
// workflow consumption, and authorization interaction together. JavaScript
// cannot call this operation directly; it can only return a declared effect
// plan that this method revalidates.
func (p *Provider) commitScriptedSignup(ctx context.Context, outcome idpprogram.Outcome, submission idpworkflow.Submission, continuation idpcontinuation.WorkflowContinuation, bindings idpcontinuation.Bindings, record idpstore.InteractionRecord, clientAddress string) (signupCommitResult, error) {
	if len(outcome.Effects) != 2 && len(outcome.Effects) != 3 || outcome.Effects[0].Kind != idpprogram.EffectCreateLocalIdentity || outcome.Effects[1].Kind != idpprogram.EffectAttachPasswordCredential || len(outcome.Effects) == 3 && outcome.Effects[2].Kind != idpprogram.EffectConsumeInvitation {
		return signupCommitResult{}, errors.New("signup script emitted an invalid effect sequence")
	}
	var identity struct {
		Login       string `json:"login"`
		DisplayName string `json:"displayName"`
	}
	var credential struct {
		PasswordHandle             string `json:"passwordHandle"`
		PasswordConfirmationHandle string `json:"passwordConfirmationHandle"`
	}
	var invitation struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(outcome.Effects[0].Payload, &identity); err != nil {
		return signupCommitResult{}, errors.Wrap(err, "decode signup identity effect")
	}
	if err := json.Unmarshal(outcome.Effects[1].Payload, &credential); err != nil {
		return signupCommitResult{}, errors.Wrap(err, "decode signup credential effect")
	}
	if len(outcome.Effects) == 3 {
		if p.durableInvitations == nil || json.Unmarshal(outcome.Effects[2].Payload, &invitation) != nil || strings.TrimSpace(invitation.Code) == "" {
			return signupCommitResult{}, errors.New("signup invitation effect is not acceptable")
		}
	}
	password, confirmation, ok := submissionSecrets(submission, credential.PasswordHandle, credential.PasswordConfirmationHandle)
	if !ok || len(password) == 0 || !equalBytes(password, confirmation) || !p.allowRegistration(ctx, record.ClientID, clientAddress, identity.Login) {
		return signupCommitResult{}, errors.New("signup effects are not acceptable")
	}
	defer clearBytes(password)
	defer clearBytes(confirmation)
	prepared, err := p.registration.PrepareCreate(ctx, idpaccounts.CreateRequest{Login: identity.Login, Name: identity.DisplayName, Password: password, Email: identity.Login})
	if err != nil {
		return signupCommitResult{}, err
	}
	sessionHandle, err := randomB64(32)
	if err != nil {
		return signupCommitResult{}, errors.Wrap(err, "generate signup session handle")
	}
	now := p.now()
	sessionHash := idpstore.HashSecret(p.csrfKey, sessionHandle)
	session := idpstore.Session{IDHash: sessionHash, UserID: prepared.User.ID, AuthTime: now, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(p.sessionTTL)}
	err = p.store.Update(ctx, func(tx idpstore.TxStore) error {
		continuationStore, ok := tx.(idpcontinuation.Store)
		if !ok {
			return errors.New("signup transaction store does not own workflow continuations")
		}
		if _, err := p.workflowContinuations.ConsumeLoaded(ctx, continuation, bindings, idpcontinuation.TerminalOutcome{Kind: idpcontinuation.TerminalComplete}, continuationStore); err != nil {
			return err
		}
		if err := p.registration.CommitPrepared(ctx, tx, prepared); err != nil {
			return err
		}
		if len(outcome.Effects) == 3 {
			if _, err := p.durableInvitations.RedeemInTransaction(ctx, tx, invitation.Code, record.ClientID, now); err != nil {
				return err
			}
		}
		if err := tx.CreateSession(ctx, session); err != nil {
			return err
		}
		_, err := tx.ConsumeInteraction(ctx, record.IDHash, now, idpstore.InteractionOutcomeApproved)
		return err
	})
	if err != nil {
		return signupCommitResult{}, err
	}
	return signupCommitResult{User: prepared.User, SessionHandle: sessionHandle, SessionHash: sessionHash}, nil
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
