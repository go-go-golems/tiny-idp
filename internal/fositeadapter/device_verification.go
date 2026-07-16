package fositeadapter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/manuel/tinyidp/internal/securitytrace"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/idpui"
)

const deviceVerificationMaxBody = 16 << 10

// deviceVerification is the browser half of RFC 8628. The public user code
// only selects a short-lived, server-owned continuation. Approval and denial
// then require a fresh password authentication, an opaque continuation handle,
// and a CSRF token bound to this browser.
func (p *Provider) deviceVerification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	switch r.Method {
	case http.MethodGet:
		p.beginDeviceVerification(w, r)
	case http.MethodPost:
		p.completeDeviceVerification(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *Provider) beginDeviceVerification(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if duplicateFormValue(query, idpui.UserCodeFieldName) {
		p.renderDeviceVerification(w, r, http.StatusBadRequest, p.newDeviceCodeEntryPage(&idpui.PublicError{Code: idpui.ErrorInvalidUserCode, Field: idpui.FieldUserCode, Summary: "Enter a valid current code."}))
		return
	}
	rawCode := query.Get(idpui.UserCodeFieldName)
	if rawCode == "" {
		p.renderDeviceVerification(w, r, http.StatusOK, p.newDeviceCodeEntryPage(nil))
		return
	}
	address, err := p.clientAddress.ResolveClientAddress(r)
	if err != nil {
		http.Error(w, "resolve client address failed", http.StatusInternalServerError)
		return
	}
	if !p.rateLimiter.Allow(r.Context(), "device:verify:address:"+address) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "device.verification.rejected", Result: "rejected", Reason: "rate_limited"})
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	normalized := normalizeUserCode(rawCode)
	if normalized == "" {
		p.renderDeviceVerification(w, r, http.StatusBadRequest, p.newDeviceCodeEntryPage(&idpui.PublicError{Code: idpui.ErrorInvalidUserCode, Field: idpui.FieldUserCode, Summary: "Enter a valid current code."}))
		return
	}
	grant, err := p.store.GetDeviceGrantByUserCodeHash(r.Context(), userCodeHash(p.csrfKey, normalized))
	if err != nil || grant.Status != idpstore.DeviceGrantPending || !p.now().Before(grant.ExpiresAt) {
		// Do not expose whether an arbitrary public code was once valid, expired,
		// denied, or already approved.
		p.renderDeviceVerification(w, r, http.StatusBadRequest, p.newDeviceCodeEntryPage(&idpui.PublicError{Code: idpui.ErrorInvalidUserCode, Field: idpui.FieldUserCode, Summary: "Enter a valid current code."}))
		return
	}
	handle, csrfToken, record, err := p.createDeviceVerificationInteraction(w, r, grant)
	if err != nil {
		http.Error(w, "create device verification interaction failed", http.StatusInternalServerError)
		return
	}
	p.renderDeviceVerification(w, r, http.StatusOK, p.newDeviceConfirmationPage(record, handle, csrfToken, grant, "", nil))
}

func (p *Provider) completeDeviceVerification(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, deviceVerificationMaxBody)
	if err := r.ParseForm(); err != nil || duplicateFormValue(r.PostForm, idpui.InteractionFieldName) || duplicateFormValue(r.PostForm, idpui.CSRFFieldName) || duplicateFormValue(r.PostForm, idpui.ActionFieldName) || duplicateFormValue(r.PostForm, idpui.LoginFieldName) || duplicateFormValue(r.PostForm, idpui.PasswordFieldName) {
		http.Error(w, "invalid verification form", http.StatusBadRequest)
		return
	}
	address, err := p.clientAddress.ResolveClientAddress(r)
	if err != nil {
		http.Error(w, "resolve client address failed", http.StatusInternalServerError)
		return
	}
	if !p.rateLimiter.Allow(r.Context(), "device:decision:address:"+address) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "device.verification.rejected", Result: "rejected", Reason: "rate_limited"})
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	handle := r.PostForm.Get(idpui.InteractionFieldName)
	if !p.validateCSRF(r, handle) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "device.verification.rejected", Result: "rejected", Reason: "invalid_csrf"})
		http.Error(w, "invalid csrf token", http.StatusBadRequest)
		return
	}
	record, err := p.store.GetInteraction(r.Context(), deviceVerificationHandleHash(p.csrfKey, handle))
	if err != nil || record.ConsumedAt != nil || len(record.DeviceUserCodeHash) == 0 || !p.now().Before(record.ExpiresAt) || !equalBytes(record.BrowserBindingHash, p.browserBindingHash(r)) {
		http.Error(w, "device verification interaction is invalid or expired", http.StatusBadRequest)
		return
	}
	grant, err := p.store.GetDeviceGrantByUserCodeHash(r.Context(), record.DeviceUserCodeHash)
	if err != nil || grant.ClientID != record.ClientID || grant.Status != idpstore.DeviceGrantPending || !p.now().Before(grant.ExpiresAt) {
		http.Error(w, "device verification request is unavailable", http.StatusBadRequest)
		return
	}
	client, err := p.store.GetClient(r.Context(), record.ClientID)
	if err != nil || client.Disabled || !client.AllowsGrantType(idpstore.GrantDeviceCode) || !equalBytes(record.GenerationHash, clientGenerationHash(client)) {
		http.Error(w, "device verification client is unavailable", http.StatusBadRequest)
		return
	}
	action := idpui.Action(r.PostForm.Get(idpui.ActionFieldName))
	if action != idpui.ActionApprove && action != idpui.ActionDeny {
		http.Error(w, "invalid device verification decision", http.StatusBadRequest)
		return
	}
	login := strings.ToLower(strings.TrimSpace(r.PostForm.Get(idpui.LoginFieldName)))
	if login == "" {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "device.verification.rejected", ClientID: client.ID, Result: "rejected", Reason: "missing_login"})
		p.renderDeviceVerification(w, r, http.StatusBadRequest, p.newDeviceConfirmationPage(record, handle, r.PostForm.Get(idpui.CSRFFieldName), grant, "", &idpui.PublicError{Code: idpui.ErrorMissingLogin, Field: idpui.FieldCredentials, Summary: "Enter your username and password."}))
		return
	}
	if !p.allowLogin(r.Context(), client.ID, address, login) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "device.verification.rejected", ClientID: client.ID, Result: "rejected", Reason: "rate_limited"})
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	result, authErr := p.authenticator.AuthenticatePassword(r.Context(), login, r.PostForm.Get(idpui.PasswordFieldName), idp.LoginMetadata{RemoteAddr: address, UserAgent: r.UserAgent(), ClientID: client.ID})
	if authErr != nil {
		if errors.Is(authErr, idpaccounts.ErrAuthenticationUnavailable) || errors.Is(authErr, idpaccounts.ErrPasswordWorkRejected) {
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "device.verification.unavailable", ClientID: client.ID, Result: "rejected", Reason: "authentication_unavailable"})
			http.Error(w, "authentication temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "device.verification.rejected", ClientID: client.ID, Result: "rejected", Reason: idpaccounts.AuditReason(authErr)})
		p.renderDeviceVerification(w, r, http.StatusUnauthorized, p.newDeviceConfirmationPage(record, handle, r.PostForm.Get(idpui.CSRFFieldName), grant, login, &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: "Invalid login or password."}))
		return
	}
	user, err := p.store.GetUser(r.Context(), result.User.ID)
	if err != nil || user.Disabled {
		http.Error(w, "user is unavailable", http.StatusForbidden)
		return
	}
	now := p.now()
	outcome := idpstore.InteractionOutcomeApproved
	decision := idpstore.DeviceGrantApprove
	if action == idpui.ActionDeny {
		outcome = idpstore.InteractionOutcomeDenied
		decision = idpstore.DeviceGrantDeny
	}
	err = p.store.Update(r.Context(), func(tx idpstore.TxStore) error {
		current, err := tx.GetDeviceGrantByUserCodeHash(r.Context(), record.DeviceUserCodeHash)
		if err != nil {
			return err
		}
		if current.ClientID != record.ClientID || current.Status != idpstore.DeviceGrantPending || !now.Before(current.ExpiresAt) {
			return idpstore.ErrDeviceGrantNotPending
		}
		if _, err := tx.ConsumeInteraction(r.Context(), record.IDHash, now, outcome); err != nil {
			return err
		}
		request := idpstore.DeviceDecisionRequest{UserCodeHash: record.DeviceUserCodeHash, Decision: decision, Now: now}
		if decision == idpstore.DeviceGrantApprove {
			request.UserID = user.ID
			request.Subject = user.Sub
			request.AuthTime = now
			request.AuthenticationMethods = append([]string(nil), result.AMR...)
			request.ApprovedScopes = append([]string(nil), current.RequestedScopes...)
			request.ApprovedAudiences = append([]string(nil), current.RequestedAudiences...)
		}
		_, err = tx.DecideDeviceGrant(r.Context(), request)
		return err
	})
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: now, Name: "device.verification.rejected", ClientID: client.ID, Result: "rejected", Reason: "state_transition_failed"})
		http.Error(w, "device verification request is unavailable", http.StatusBadRequest)
		return
	}
	eventName := "device.verification.approved"
	notice := "The device request was approved. You may return to your device."
	if decision == idpstore.DeviceGrantDeny {
		eventName = "device.verification.denied"
		notice = "The device request was denied. You may return to your device."
	}
	p.recordAudit(r.Context(), idp.Event{Time: now, Name: eventName, ClientID: client.ID, Subject: user.Sub, Result: "accepted"})
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.AuthenticationSatisfied, InteractionID: interactionTraceID(record), ClientID: client.ID})
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionTerminal, InteractionID: interactionTraceID(record), ClientID: client.ID, Outcome: string(outcome)})
	p.renderDeviceVerification(w, r, http.StatusOK, idpui.DeviceVerificationPage{DocumentTitle: "Device verification complete", Form: idpui.DeviceVerificationForm{ActionURL: p.issuer.Endpoint("/device")}, Notice: &idpui.DeviceVerificationNotice{Summary: notice}})
}

func (p *Provider) createDeviceVerificationInteraction(w http.ResponseWriter, r *http.Request, grant idpstore.DeviceGrant) (string, string, idpstore.InteractionRecord, error) {
	client, err := p.store.GetClient(r.Context(), grant.ClientID)
	if err != nil {
		return "", "", idpstore.InteractionRecord{}, fmt.Errorf("load device client: %w", err)
	}
	if client.Disabled || !client.AllowsGrantType(idpstore.GrantDeviceCode) {
		return "", "", idpstore.InteractionRecord{}, fmt.Errorf("device client is disabled or no longer device-capable")
	}
	handle, err := randomB64(32)
	if err != nil {
		return "", "", idpstore.InteractionRecord{}, fmt.Errorf("generate device verification handle: %w", err)
	}
	csrfToken, bindingHash, err := p.issueCSRF(w, r, handle)
	if err != nil {
		return "", "", idpstore.InteractionRecord{}, fmt.Errorf("issue device verification csrf: %w", err)
	}
	now := p.now()
	expiresAt := now.Add(p.interactionTTL)
	if grant.ExpiresAt.Before(expiresAt) {
		expiresAt = grant.ExpiresAt
	}
	record := idpstore.InteractionRecord{IDHash: deviceVerificationHandleHash(p.csrfKey, handle), ClientID: client.ID, GenerationHash: clientGenerationHash(client), DeviceUserCodeHash: append([]byte(nil), grant.UserCodeHash...), BrowserBindingHash: bindingHash, CreatedAt: now, ExpiresAt: expiresAt}
	if err := p.store.CreateInteraction(r.Context(), record); err != nil {
		return "", "", idpstore.InteractionRecord{}, fmt.Errorf("persist device verification interaction: %w", err)
	}
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionCreated, InteractionID: interactionTraceID(record), ClientID: client.ID})
	return handle, csrfToken, record, nil
}

func deviceVerificationHandleHash(key []byte, handle string) []byte {
	return idpstore.HashSecret(key, "tinyidp/device-verification/v1\x00"+handle)
}

func (p *Provider) newDeviceCodeEntryPage(publicError *idpui.PublicError) idpui.DeviceVerificationPage {
	return idpui.DeviceVerificationPage{DocumentTitle: "Verify your device", Form: idpui.DeviceVerificationForm{ActionURL: p.issuer.Endpoint("/device")}, Entry: &idpui.DeviceCodeEntryPrompt{UserCodeField: idpui.UserCodeFieldName}, Error: publicError}
}

func (p *Provider) newDeviceConfirmationPage(record idpstore.InteractionRecord, handle, csrfToken string, grant idpstore.DeviceGrant, login string, publicError *idpui.PublicError) idpui.DeviceVerificationPage {
	scopes := make([]idpui.Scope, 0, len(grant.RequestedScopes))
	for _, scope := range grant.RequestedScopes {
		scopes = append(scopes, idpui.Scope{Name: scope})
	}
	return idpui.DeviceVerificationPage{DocumentTitle: "Approve device access", Form: idpui.DeviceVerificationForm{ActionURL: p.issuer.Endpoint("/device"), InteractionField: idpui.InteractionFieldName, Interaction: handle, CSRFField: idpui.CSRFFieldName, CSRFToken: csrfToken, ActionField: idpui.ActionFieldName, Actions: []idpui.Action{idpui.ActionApprove, idpui.ActionDeny}}, Confirmation: &idpui.DeviceConfirmationPrompt{ClientID: record.ClientID, Scopes: scopes, Login: idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName, LoginValue: login, Autofocus: true}}, Error: publicError}
}
