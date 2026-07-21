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

	"github.com/go-go-golems/tiny-idp/internal/assurance"
	"github.com/go-go-golems/tiny-idp/internal/securitytrace"
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
	executor, err := p.activeSignupExecutor()
	if err != nil || p.workflowContinuations == nil {
		return errors.New("scripted signup is unavailable")
	}
	record, err := p.store.GetInteraction(r.Context(), idpstore.HashSecret(p.csrfKey, interactionHandle))
	if err != nil {
		return errors.Wrap(err, "load signup authorization interaction")
	}
	presentation, err := executor.Start(r.Context(), idpsignup.StartInput{
		ClientID:          client.ID,
		RedirectURI:       ar.GetRedirectURI().String(),
		RequestedScope:    strings.Join(ar.GetRequestedScopes(), " "),
		InteractionID:     hex.EncodeToString(record.IDHash),
		HasBrowserSession: len(record.SessionIDHash) != 0,
	})
	if err != nil {
		return err
	}
	workflow := executor.Program().Workflows[idpsignup.WorkflowID]
	publicValues, err := json.Marshal(presentation.Presentation.PublicValues)
	if err != nil {
		return errors.Wrap(err, "encode signup public values")
	}
	continuationHandle, _, err := p.workflowContinuations.Create(r.Context(), idpcontinuation.WorkflowContinuation{
		WorkflowID: idpsignup.WorkflowID, ResumeHandlerID: presentation.Presentation.ResumeHandler,
		ProgramFingerprint: executor.Fingerprint(), SchemaVersion: "v1", WorkflowVersion: workflow.Version,
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
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.ContinuationCreated, InteractionID: interactionTraceID(record), Transition: assurance.StepContinuationCreate, Outcome: assurance.TransitionApplied})
	p.renderWorkflow(w, r, http.StatusOK, workflowPage(p, record, interactionHandle, csrfToken, continuationHandle, presentation.Fields, presentation.Actions, presentation.Presentation.PublicValues, nil))
	return nil
}

func (p *Provider) resumeScriptedSignup(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, client idpstore.Client, record idpstore.InteractionRecord, interactionHandle, clientAddress string) {
	continuationHandle := r.PostForm.Get(idpui.WorkflowContinuationFieldName)
	continuation, err := p.workflowContinuations.Load(r.Context(), continuationHandle, p.signupLoadBindings(record, r))
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "workflow.signup.resume_rejected", ClientID: record.ClientID, Result: "rejected", Reason: "continuation_unavailable"})
		http.Error(w, "registration request was not accepted", http.StatusBadRequest)
		return
	}
	executor, err := p.signupExecutorFor(continuation.ProgramFingerprint)
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "workflow.signup.resume_rejected", ClientID: record.ClientID, Result: "rejected", Reason: "generation_unavailable"})
		http.Error(w, "registration request was not accepted", http.StatusBadRequest)
		return
	}
	continuationBindings := p.signupBindingsFor(record, r, continuation.ProgramFingerprint)
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
		if _, err := p.workflowContinuations.Consume(r.Context(), continuationHandle, continuation.Revision, continuationBindings, idpcontinuation.TerminalOutcome{Kind: idpcontinuation.TerminalDeny}); err != nil {
			http.Error(w, "registration request was not accepted", http.StatusBadRequest)
			return
		}
		if _, err := p.store.ConsumeInteraction(r.Context(), record.IDHash, p.now(), idpstore.InteractionOutcomeDenied); err != nil {
			http.Error(w, "authorization interaction already completed", http.StatusBadRequest)
			return
		}
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.ContinuationTerminal, InteractionID: interactionTraceID(record), Transition: assurance.StepContinuationConsume, Outcome: assurance.TransitionDenied})
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionTerminal, InteractionID: interactionTraceID(record), Transition: assurance.StepInteractionDeny, Outcome: assurance.TransitionDenied})
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrAccessDenied)
		return
	}
	if submission.Action == idpworkflow.ActionResend {
		challengeID, ok := pendingEmailChallengeReference(continuation)
		if !ok || p.emailChallenges == nil {
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, nil)
			return
		}
		if err := p.emailChallenges.Resend(r.Context(), idpemailchallenge.Reference{ID: challengeID, Version: idpemailchallenge.RecordVersionV1}, idpemailchallenge.BindingsFromContinuation(continuation)); err != nil {
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "workflow.signup.email_challenge_resend", ClientID: record.ClientID, Result: "rejected", Reason: emailChallengeFailureReason(err)})
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, nil)
			return
		}
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "workflow.signup.email_challenge_resend", ClientID: record.ClientID, Result: "accepted"})
		p.renderWorkflow(w, r, http.StatusOK, workflowPage(p, record, interactionHandle, r.PostForm.Get(idpui.CSRFFieldName), continuationHandle, fields, actions, nil, nil))
		return
	}
	input, err := executor.SubmissionInput(continuation.ResumeHandlerID, submission.PublicValues)
	if err == nil {
		input, err = mergeWorkflowCarry(continuation.Carry, input)
	}
	var evidence map[string]json.RawMessage
	verifiedEmail := ""
	verifiedReference := idpcontinuation.EvidenceReference{}
	if challengeID, ok := pendingEmailChallengeReference(continuation); ok {
		if p.emailChallenges == nil {
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
			return
		}
		verified, verifyErr := p.emailChallenges.Verify(r.Context(), idpemailchallenge.Reference{ID: challengeID, Version: idpemailchallenge.RecordVersionV1}, submission.PublicValues[idpworkflow.FieldEmailCode], idpemailchallenge.BindingsFromContinuation(continuation))
		if verifyErr != nil {
			p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.EvidenceVerified, InteractionID: interactionTraceID(record), Transition: assurance.StepEvidenceVerify, Outcome: assurance.TransitionRejected})
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
			return
		}
		evidence, err = idpemailchallenge.EvidenceProjection(verified)
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.EvidenceVerified, InteractionID: interactionTraceID(record), Transition: assurance.StepEvidenceVerify, Outcome: assurance.TransitionApplied})
		verifiedEmail = verified.Address
		verifiedReference = idpcontinuation.EvidenceReference{Kind: "verifiedEmail", ID: challengeID}
		input = continuation.Carry
	} else if challengeID, ok := verifiedEmailReference(continuation); ok {
		if p.emailChallenges == nil {
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
			return
		}
		verified, evidenceErr := p.emailChallenges.Evidence(r.Context(), idpemailchallenge.Reference{ID: challengeID, Version: idpemailchallenge.RecordVersionV1}, idpemailchallenge.BindingsFromContinuation(continuation))
		if evidenceErr != nil {
			p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.EvidenceVerified, InteractionID: interactionTraceID(record), Transition: assurance.StepEvidenceVerify, Outcome: assurance.TransitionRejected})
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
			return
		}
		evidence, err = idpemailchallenge.EvidenceProjection(verified)
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.EvidenceVerified, InteractionID: interactionTraceID(record), Transition: assurance.StepEvidenceVerify, Outcome: assurance.TransitionApplied})
		verifiedEmail = verified.Address
		verifiedReference = idpcontinuation.EvidenceReference{Kind: "verifiedEmail", ID: challengeID}
	}
	if err != nil || p.workflowContinuations.ValidateResumeInput(r.Context(), continuation, input) != nil {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.LambdaInvocationStarted, InteractionID: interactionTraceID(record), Transition: assurance.StepLambdaInvoke, Outcome: assurance.TransitionApplied})
	outcome, err := executor.InvokeSubmission(r.Context(), continuation.ResumeHandlerID, input, signupSubmissionSecrets(submission), evidence)
	if err != nil {
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.LambdaInvocationRejected, InteractionID: interactionTraceID(record), Transition: assurance.StepLambdaInvoke, Outcome: assurance.TransitionRejected})
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	lambdaOutcome, mapErr := assurance.LambdaOutcomeID(outcome.Kind)
	if mapErr != nil {
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.LambdaInvocationRejected, InteractionID: interactionTraceID(record), Transition: assurance.StepLambdaInvoke, Outcome: assurance.TransitionRejected})
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.LambdaInvocationCompleted, InteractionID: interactionTraceID(record), Transition: assurance.StepLambdaInvoke, Outcome: lambdaOutcome})
	if outcome.Kind == idpprogram.OutcomeChallenge {
		if err := p.beginEmailChallenge(w, r, executor, outcome, continuationHandle, continuation, record, interactionHandle); err != nil {
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "workflow.signup.email_challenge_send", ClientID: record.ClientID, Result: "rejected", Reason: emailChallengeFailureReason(err)})
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
			return
		}
		return
	}
	if outcome.Kind == idpprogram.OutcomePresent {
		var evidenceReferences []idpcontinuation.EvidenceReference
		if verifiedReference.ID != "" {
			evidenceReferences = []idpcontinuation.EvidenceReference{verifiedReference}
		}
		if err := p.advanceSignupPresentation(w, r, executor, outcome, continuationHandle, continuation, record, interactionHandle, evidenceReferences); err != nil {
			p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		}
		return
	}
	if outcome.Kind != idpprogram.OutcomeCommit {
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	registered, err := p.commitScriptedSignup(r.Context(), outcome, submission, continuation, continuationBindings, record, clientAddress, verifiedEmail)
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "account.self_registration", ClientID: record.ClientID, Result: "rejected", Reason: signupCommitFailureReason(err)})
		p.renderScriptedSignupError(w, r, record, interactionHandle, continuationHandle, fields, actions, submission.PublicValues)
		return
	}
	p.completeScriptedSignup(w, r, ar, client, record, registered)
}

func (p *Provider) advanceSignupPresentation(w http.ResponseWriter, r *http.Request, executor *idpsignup.Executor, outcome idpprogram.Outcome, handle string, current idpcontinuation.WorkflowContinuation, record idpstore.InteractionRecord, interactionHandle string, evidenceReferences []idpcontinuation.EvidenceReference) error {
	if outcome.Continuation == nil {
		return errors.New("signup presentation continuation is missing")
	}
	presentation, err := idpworkflow.DecodePresentation(outcome.Presentation)
	if err != nil {
		return err
	}
	validated, err := idpworkflow.ValidatePresentation(executor.Program(), idpsignup.WorkflowID, current.ResumeHandlerID, presentation, idpworkflow.DefaultRegistry(), idpworkflow.DefaultMaximumContinuationTTL)
	if err != nil {
		return err
	}
	publicValues, err := json.Marshal(validated.Presentation.PublicValues)
	if err != nil {
		return err
	}
	next := idpcontinuation.WorkflowContinuation{ResumeHandlerID: validated.Presentation.ResumeHandler, InputSchema: validated.InputSchema, Carry: validated.Presentation.Carry, EvidenceReferences: evidenceReferences, Presentation: idpcontinuation.PresentationState{ID: "signup", Fields: fieldIDs(validated.Fields), AllowedActions: actionIDs(validated.Actions), PublicValues: publicValues}, ExpiresAt: p.now().Add(validated.Presentation.ExpiresIn)}
	nextHandle, _, err := p.workflowContinuations.Advance(r.Context(), handle, current.Revision, p.signupBindingsFor(record, r, current.ProgramFingerprint), next)
	if err != nil {
		return err
	}
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.ContinuationCreated, InteractionID: interactionTraceID(record), Transition: assurance.StepContinuationCreate, Outcome: assurance.TransitionApplied})
	p.renderWorkflow(w, r, http.StatusOK, workflowPage(p, record, interactionHandle, r.PostForm.Get(idpui.CSRFFieldName), nextHandle, validated.Fields, validated.Actions, validated.Presentation.PublicValues, nil))
	return nil
}

func (p *Provider) beginEmailChallenge(w http.ResponseWriter, r *http.Request, executor *idpsignup.Executor, outcome idpprogram.Outcome, handle string, current idpcontinuation.WorkflowContinuation, record idpstore.InteractionRecord, interactionHandle string) error {
	if p.emailChallenges == nil || outcome.Continuation == nil {
		return errors.New("email challenge is unavailable")
	}
	var request struct {
		Kind, Email, Template           string
		MaximumAttempts, MaximumResends int
	}
	if err := json.Unmarshal(outcome.Challenge, &request); err != nil || request.Kind != "email_code" || request.Email == "" || request.MaximumAttempts <= 0 || request.MaximumResends <= 0 || uint64(request.MaximumAttempts) > uint64(^uint32(0)) || uint64(request.MaximumResends) > uint64(^uint32(0)) {
		return errors.New("email challenge request is invalid")
	}
	maximumAttempts := uint32(request.MaximumAttempts) // #nosec G115 -- validated positive and no greater than MaxUint32 above.
	maximumResends := uint32(request.MaximumResends)   // #nosec G115 -- validated positive and no greater than MaxUint32 above.
	workflow := executor.Program().Workflows[idpsignup.WorkflowID]
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
	resend, _ := registry.Action(idpworkflow.ActionResend)
	challengeID, err := randomB64(24)
	if err != nil {
		return errors.Wrap(err, "generate email challenge id")
	}
	expiresAt := p.now().Add(time.Duration(outcome.Continuation.ExpiresIn) * time.Second)
	challengeBindings := idpemailchallenge.BindingsFromContinuation(current)
	challengeBindings.ResumeHandlerID = outcome.Continuation.HandlerID
	if _, err := p.emailChallenges.CreateAndSend(r.Context(), idpemailchallenge.CreateRequest{ID: challengeID, Email: request.Email, Template: request.Template, Bindings: challengeBindings, ExpiresAt: expiresAt, MaximumAttempts: maximumAttempts, MaximumResends: maximumResends}); err != nil {
		return err
	}
	next := idpcontinuation.WorkflowContinuation{ResumeHandlerID: outcome.Continuation.HandlerID, InputSchema: inputSchema, Carry: outcome.Continuation.Carry, EvidenceReferences: []idpcontinuation.EvidenceReference{{Kind: "pendingEmailChallenge", ID: challengeID}}, Presentation: idpcontinuation.PresentationState{ID: "email-code", Fields: []string{string(idpworkflow.FieldEmailCode)}, AllowedActions: []string{string(idpworkflow.ActionSubmit), string(idpworkflow.ActionResend), string(idpworkflow.ActionDeny)}, PublicValues: json.RawMessage(`{}`)}, ExpiresAt: expiresAt}
	nextHandle, _, err := p.workflowContinuations.Advance(r.Context(), handle, current.Revision, p.signupBindingsFor(record, r, current.ProgramFingerprint), next)
	if err != nil {
		return err
	}
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.ContinuationCreated, InteractionID: interactionTraceID(record), Transition: assurance.StepContinuationCreate, Outcome: assurance.TransitionApplied})
	p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "workflow.signup.email_challenge_send", ClientID: record.ClientID, Result: "accepted"})
	p.renderWorkflow(w, r, http.StatusOK, workflowPage(p, record, interactionHandle, r.PostForm.Get(idpui.CSRFFieldName), nextHandle, []idpworkflow.FieldDescriptor{code}, []idpworkflow.ActionDescriptor{submit, resend, deny}, map[idpworkflow.FieldID]string{}, nil))
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
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionTerminal, InteractionID: interactionTraceID(record), Transition: assurance.StepInteractionApprove, Outcome: assurance.TransitionApproved})
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
func (p *Provider) commitScriptedSignup(ctx context.Context, outcome idpprogram.Outcome, submission idpworkflow.Submission, continuation idpcontinuation.WorkflowContinuation, bindings idpcontinuation.Bindings, record idpstore.InteractionRecord, clientAddress, verifiedEmail string) (signupCommitResult, error) {
	if len(outcome.Effects) != 2 && len(outcome.Effects) != 3 || outcome.Effects[0].Kind != idpprogram.EffectCreateLocalIdentity || outcome.Effects[1].Kind != idpprogram.EffectAttachPasswordCredential || len(outcome.Effects) == 3 && outcome.Effects[2].Kind != idpprogram.EffectConsumeInvitation {
		p.recordSecurity(ctx, securitytrace.Event{Kind: securitytrace.EffectValidationCompleted, InteractionID: interactionTraceID(record), Transition: assurance.StepEffectValidate, Outcome: assurance.TransitionRejected})
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
	if !ok || len(password) == 0 || !equalBytes(password, confirmation) || verifiedEmail != "" && !strings.EqualFold(identity.Login, verifiedEmail) || !p.allowRegistration(ctx, record.ClientID, clientAddress, identity.Login) {
		p.recordSecurity(ctx, securitytrace.Event{Kind: securitytrace.EffectValidationCompleted, InteractionID: interactionTraceID(record), Transition: assurance.StepEffectValidate, Outcome: assurance.TransitionRejected})
		return signupCommitResult{}, errors.New("signup effects are not acceptable")
	}
	p.recordSecurity(ctx, securitytrace.Event{Kind: securitytrace.EffectValidationCompleted, InteractionID: interactionTraceID(record), Transition: assurance.StepEffectValidate, Outcome: assurance.TransitionApplied})
	defer clearBytes(password)
	defer clearBytes(confirmation)
	prepared, err := p.registration.PrepareCreate(ctx, idpaccounts.CreateRequest{Login: identity.Login, Name: identity.DisplayName, Password: password, Email: identity.Login, EmailVerified: verifiedEmail != ""})
	if err != nil {
		p.recordSecurity(ctx, securitytrace.Event{Kind: securitytrace.NativeEffectCommitted, InteractionID: interactionTraceID(record), Transition: assurance.StepSignupCommit, Outcome: assurance.TransitionRejected})
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
		p.recordSecurity(ctx, securitytrace.Event{Kind: securitytrace.NativeEffectCommitted, InteractionID: interactionTraceID(record), Transition: assurance.StepSignupCommit, Outcome: assurance.TransitionRejected})
		return signupCommitResult{}, err
	}
	p.recordSecurity(ctx, securitytrace.Event{Kind: securitytrace.NativeEffectCommitted, InteractionID: interactionTraceID(record), Transition: assurance.StepSignupCommit, Outcome: assurance.TransitionApplied})
	p.recordSecurity(ctx, securitytrace.Event{Kind: securitytrace.ContinuationTerminal, InteractionID: interactionTraceID(record), Transition: assurance.StepContinuationConsume, Outcome: assurance.TransitionApplied})
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

// signupSubmissionSecrets projects only descriptors submitted on the active
// page. A workflow that collects identity data before a password must not hand
// zero-value secret handles to the Goja binding, because those values are not
// valid native capabilities.
func signupSubmissionSecrets(submission idpworkflow.Submission) map[string]idpworkflow.SecretHandle {
	secrets := map[string]idpworkflow.SecretHandle{}
	if handle, ok := submission.Secrets[idpworkflow.FieldPassword]; ok && handle.Token() != "" {
		secrets["password"] = handle
	}
	if handle, ok := submission.Secrets[idpworkflow.FieldPasswordConfirmation]; ok && handle.Token() != "" {
		secrets["passwordConfirmation"] = handle
	}
	return secrets
}

func (p *Provider) renderScriptedSignupError(w http.ResponseWriter, r *http.Request, record idpstore.InteractionRecord, interactionHandle, continuationHandle string, fields []idpworkflow.FieldDescriptor, actions []idpworkflow.ActionDescriptor, values map[idpworkflow.FieldID]string) {
	errorField := idpworkflow.FieldEmail
	if len(fields) > 0 {
		errorField = fields[0].ID
	}
	p.renderWorkflow(w, r, http.StatusBadRequest, workflowPage(p, record, interactionHandle, r.PostForm.Get(idpui.CSRFFieldName), continuationHandle, fields, actions, values, []idpui.WorkflowFieldError{{Field: errorField, Code: idpworkflow.ErrorRejected}}))
}

func (p *Provider) signupBindingsFor(record idpstore.InteractionRecord, r *http.Request, fingerprint string) idpcontinuation.Bindings {
	bindings := p.signupLoadBindings(record, r)
	bindings.ProgramFingerprint = fingerprint
	return bindings
}

func (p *Provider) signupLoadBindings(record idpstore.InteractionRecord, r *http.Request) idpcontinuation.Bindings {
	bindings := idpcontinuation.Bindings{WorkflowID: idpsignup.WorkflowID, ClientID: record.ClientID, RedirectURI: record.RedirectURI, ClientGeneration: hex.EncodeToString(record.GenerationHash), RequestDigest: record.RequestDigest, BrowserBindingHash: p.browserBindingHash(r), SessionIDHash: p.browserSessionHash(r), BrowserContextHash: p.browserContextHash(r)}
	if p.scriptedSignupManager == nil && p.scriptedSignup != nil {
		bindings.ProgramFingerprint = p.scriptedSignup.Fingerprint()
	}
	return bindings
}

func (p *Provider) activeSignupExecutor() (*idpsignup.Executor, error) {
	if p.scriptedSignupManager != nil {
		return p.scriptedSignupManager.Active()
	}
	if p.scriptedSignup == nil {
		return nil, errors.New("scripted signup is unavailable")
	}
	return p.scriptedSignup, nil
}

func (p *Provider) signupExecutorFor(fingerprint string) (*idpsignup.Executor, error) {
	if p.scriptedSignupManager != nil {
		return p.scriptedSignupManager.ExecutorFor(fingerprint)
	}
	if p.scriptedSignup == nil || p.scriptedSignup.Fingerprint() != fingerprint {
		return nil, errors.New("scripted signup generation is unavailable")
	}
	return p.scriptedSignup, nil
}

func workflowPage(p *Provider, record idpstore.InteractionRecord, interactionHandle, csrfToken, continuationHandle string, fields []idpworkflow.FieldDescriptor, actions []idpworkflow.ActionDescriptor, values map[idpworkflow.FieldID]string, fieldErrors []idpui.WorkflowFieldError) idpui.WorkflowPage {
	page := idpui.WorkflowPage{DocumentTitle: "Create an account", ClientID: record.ClientID, Form: idpui.WorkflowForm{ActionURL: p.issuer.Endpoint("/authorize"), RedirectOrigin: interactionRedirectOrigin(record.RedirectURI), InteractionField: idpui.InteractionFieldName, Interaction: interactionHandle, ContinuationField: idpui.WorkflowContinuationFieldName, Continuation: continuationHandle, CSRFField: idpui.CSRFFieldName, CSRFToken: csrfToken, ActionField: idpui.ActionFieldName}, Errors: fieldErrors}
	for _, field := range fields {
		page.Fields = append(page.Fields, idpui.WorkflowField{Descriptor: field, Value: values[field.ID]})
	}
	for _, action := range actions {
		page.Actions = append(page.Actions, idpui.WorkflowAction{Descriptor: action})
	}
	return page
}

func pendingEmailChallengeReference(continuation idpcontinuation.WorkflowContinuation) (string, bool) {
	for _, reference := range continuation.EvidenceReferences {
		if reference.Kind == "pendingEmailChallenge" && reference.ID != "" {
			return reference.ID, true
		}
	}
	return "", false
}

func verifiedEmailReference(continuation idpcontinuation.WorkflowContinuation) (string, bool) {
	for _, reference := range continuation.EvidenceReferences {
		if reference.Kind == "verifiedEmail" && reference.ID != "" {
			return reference.ID, true
		}
	}
	return "", false
}

func emailChallengeFailureReason(err error) string {
	switch {
	case errors.Is(err, idpemailchallenge.ErrExpired):
		return "expired"
	case errors.Is(err, idpemailchallenge.ErrAttemptsExceeded):
		return "attempts_exceeded"
	case errors.Is(err, idpemailchallenge.ErrResendLimited):
		return "resend_limited"
	case errors.Is(err, idpemailchallenge.ErrBinding):
		return "binding_rejected"
	case errors.Is(err, idpemailchallenge.ErrAlreadyTerminal), errors.Is(err, idpemailchallenge.ErrConflict):
		return "already_terminal"
	default:
		return "unavailable"
	}
}

func mergeWorkflowCarry(carry, input json.RawMessage) (json.RawMessage, error) {
	if len(carry) == 0 {
		return input, nil
	}
	values := map[string]any{}
	if err := json.Unmarshal(carry, &values); err != nil {
		return nil, err
	}
	var submitted map[string]any
	if err := json.Unmarshal(input, &submitted); err != nil {
		return nil, err
	}
	for key, value := range submitted {
		values[key] = value
	}
	return json.Marshal(values)
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
