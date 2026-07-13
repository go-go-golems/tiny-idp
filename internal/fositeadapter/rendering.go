package fositeadapter

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/idpui"
)

const maxInteractionDocumentBytes = 256 << 10

var errInteractionDocumentTooLarge = errors.New("interaction document exceeds size limit")

func (p *Provider) newInteractionPage(
	interactionHandle string,
	csrfToken string,
	actions idpstore.InteractionRequiredAction,
	request url.Values,
	includeConsent bool,
	clientID string,
	scopes []string,
	loginValue string,
	publicError *idpui.PublicError,
) idpui.InteractionPage {
	needLogin := actions.Has(idpstore.InteractionRequireLogin) || actions.Has(idpstore.InteractionRequireFreshLogin) || actions.Has(idpstore.InteractionRequireStepUp)
	formActions := []idpui.Action{idpui.ActionContinue}
	if includeConsent {
		formActions = []idpui.Action{idpui.ActionApprove, idpui.ActionDeny}
	}
	title := "Continue authorization"
	if needLogin && includeConsent {
		title = "Sign in and approve access"
	} else if needLogin {
		title = "Sign in"
	} else if includeConsent {
		title = "Approve access"
	}
	page := idpui.InteractionPage{
		DocumentTitle: title,
		Form: idpui.InteractionForm{
			ActionURL:        p.issuer.Endpoint("/authorize"),
			InteractionField: idpui.InteractionFieldName,
			Interaction:      interactionHandle,
			CSRFField:        idpui.CSRFFieldName,
			CSRFToken:        csrfToken,
			ActionField:      idpui.ActionFieldName,
			Actions:          formActions,
		},
		Error: publicError,
	}
	if needLogin {
		page.Login = &idpui.LoginPrompt{
			Reason:        interactionLoginReason(actions, request),
			LoginField:    idpui.LoginFieldName,
			PasswordField: idpui.PasswordFieldName,
			LoginValue:    loginValue,
			Autofocus:     true,
		}
	}
	if includeConsent {
		prompt := &idpui.ConsentPrompt{ClientID: clientID, Scopes: make([]idpui.Scope, 0, len(scopes))}
		for _, scope := range scopes {
			prompt.Scopes = append(prompt.Scopes, idpui.Scope{Name: scope})
		}
		page.Consent = prompt
	}
	return page
}

func interactionLoginReason(actions idpstore.InteractionRequiredAction, request url.Values) idpui.LoginReason {
	if actions.Has(idpstore.InteractionRequireStepUp) {
		return idpui.LoginReasonStepUp
	}
	if actions.Has(idpstore.InteractionRequireFreshLogin) {
		if promptHas(request.Get("prompt"), "login") {
			return idpui.LoginReasonPromptLogin
		}
		return idpui.LoginReasonMaxAge
	}
	return idpui.LoginReasonSessionMissing
}

func (p *Provider) renderInteraction(w http.ResponseWriter, r *http.Request, status int, page idpui.InteractionPage) {
	started := time.Now()
	p.renderMetrics.attempts.Add(1)
	succeeded := false
	defer func() {
		p.renderMetrics.observe(time.Since(started), succeeded)
	}()
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if err := page.Validate(); err != nil {
		p.recordRenderFailure(r, page, "invalid_page")
		http.Error(w, "authentication page unavailable", http.StatusInternalServerError)
		return
	}
	buffer := &boundedInteractionBuffer{limit: maxInteractionDocumentBytes}
	if err := p.interactionUI.RenderInteraction(r.Context(), buffer, page.Clone()); err != nil {
		reason := "renderer_failed"
		if errors.Is(err, errInteractionDocumentTooLarge) {
			reason = "document_too_large"
			p.renderMetrics.oversizedDocuments.Add(1)
		}
		p.recordRenderFailure(r, page, reason)
		http.Error(w, "authentication page unavailable", http.StatusInternalServerError)
		return
	}
	if buffer.overflowed {
		p.renderMetrics.oversizedDocuments.Add(1)
		p.recordRenderFailure(r, page, "document_too_large")
		http.Error(w, "authentication page unavailable", http.StatusInternalServerError)
		return
	}
	if buffer.Len() == 0 {
		p.renderMetrics.emptyDocuments.Add(1)
		p.recordRenderFailure(r, page, "empty_document")
		http.Error(w, "authentication page unavailable", http.StatusInternalServerError)
		return
	}
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if written, err := w.Write(buffer.Bytes()); err != nil || written != buffer.Len() {
		p.renderMetrics.responseWriteFailures.Add(1)
		p.recordRenderFailure(r, page, "response_write_failed")
		return
	}
	succeeded = true
}

func (p *Provider) recordRenderFailure(r *http.Request, page idpui.InteractionPage, reason string) {
	clientID := ""
	if page.Consent != nil {
		clientID = page.Consent.ClientID
	}
	p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "interaction.render_failed", ClientID: clientID, Result: "rejected", Reason: reason})
}

type boundedInteractionBuffer struct {
	bytes.Buffer
	limit      int
	overflowed bool
}

func (b *boundedInteractionBuffer) Write(contents []byte) (int, error) {
	if len(contents) == 0 {
		return 0, nil
	}
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.overflowed = true
		return 0, errInteractionDocumentTooLarge
	}
	if len(contents) <= remaining {
		return b.Buffer.Write(contents)
	}
	written, err := b.Buffer.Write(contents[:remaining])
	if err != nil {
		return written, err
	}
	b.overflowed = true
	return written, errInteractionDocumentTooLarge
}

type seeOtherRedirectWriter struct {
	http.ResponseWriter
}

type interactionRenderMetrics struct {
	attempts              atomic.Uint64
	successes             atomic.Uint64
	failures              atomic.Uint64
	oversizedDocuments    atomic.Uint64
	emptyDocuments        atomic.Uint64
	responseWriteFailures atomic.Uint64
	totalLatencyNanos     atomic.Uint64
	maxLatencyNanos       atomic.Uint64
}

func (m *interactionRenderMetrics) observe(elapsed time.Duration, succeeded bool) {
	nanos := uint64(max(elapsed.Nanoseconds(), 0))
	m.totalLatencyNanos.Add(nanos)
	for current := m.maxLatencyNanos.Load(); nanos > current && !m.maxLatencyNanos.CompareAndSwap(current, nanos); current = m.maxLatencyNanos.Load() {
	}
	if succeeded {
		m.successes.Add(1)
	} else {
		m.failures.Add(1)
	}
}

func (m *interactionRenderMetrics) snapshot() idpui.RenderStats {
	return idpui.RenderStats{
		Attempts:              m.attempts.Load(),
		Successes:             m.successes.Load(),
		Failures:              m.failures.Load(),
		OversizedDocuments:    m.oversizedDocuments.Load(),
		EmptyDocuments:        m.emptyDocuments.Load(),
		ResponseWriteFailures: m.responseWriteFailures.Load(),
		TotalLatency:          time.Duration(m.totalLatencyNanos.Load()),
		MaxLatency:            time.Duration(m.maxLatencyNanos.Load()),
	}
}

func (w seeOtherRedirectWriter) WriteHeader(status int) {
	if status == http.StatusFound {
		status = http.StatusSeeOther
	}
	w.ResponseWriter.WriteHeader(status)
}
