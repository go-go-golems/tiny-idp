package jitsi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

var roomPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

const bindingCookieName = "tinyidp_jitsi_binding"

type Runtime struct {
	descriptor pluginapi.Descriptor
	settings   Settings
	services   pluginapi.RuntimeServices
	signer     *Signer
	policy     *PolicyExecutor
	handler    http.Handler
	closed     atomic.Bool
	tracer     trace.Tracer
	requests   metric.Int64Counter
	duration   metric.Float64Histogram
	tokens     metric.Int64Counter
	oidc       metric.Int64Counter
	policies   metric.Int64Counter
}

var _ pluginapi.Runtime = (*Runtime)(nil)

func newRuntime(settings Settings, services pluginapi.RuntimeServices, signer *Signer, policy *PolicyExecutor) (*Runtime, error) {
	meter := services.Meter
	if meter == nil {
		meter = metricnoop.NewMeterProvider().Meter("tinyidp/plugins/jitsi")
	}
	tracer := services.Tracer
	if tracer == nil {
		tracer = tracenoop.NewTracerProvider().Tracer("tinyidp/plugins/jitsi")
	}
	requests, err := meter.Int64Counter("tinyidp.plugin.requests", metric.WithDescription("Plugin HTTP requests"))
	if err != nil {
		return nil, fmt.Errorf("create Jitsi request counter: %w", err)
	}
	duration, err := meter.Float64Histogram("tinyidp.plugin.request.duration", metric.WithUnit("s"), metric.WithDescription("Plugin HTTP request duration"))
	if err != nil {
		return nil, fmt.Errorf("create Jitsi request duration: %w", err)
	}
	tokens, err := meter.Int64Counter("tinyidp.jitsi.tokens.issued", metric.WithDescription("Jitsi tokens issued"))
	if err != nil {
		return nil, fmt.Errorf("create Jitsi token counter: %w", err)
	}
	oidc, err := meter.Int64Counter("tinyidp.jitsi.oidc.transactions", metric.WithDescription("Jitsi OIDC transaction outcomes"))
	if err != nil {
		return nil, fmt.Errorf("create Jitsi OIDC counter: %w", err)
	}
	policies, err := meter.Int64Counter("tinyidp.jitsi.policy.invocations", metric.WithDescription("Jitsi policy invocation outcomes"))
	if err != nil {
		return nil, fmt.Errorf("create Jitsi policy counter: %w", err)
	}
	runtime := &Runtime{
		descriptor: (Definition{}).Descriptor(), settings: settings, services: services,
		signer: signer, policy: policy, tracer: tracer, requests: requests,
		duration: duration, tokens: tokens, oidc: oidc, policies: policies,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/start", runtime.start)
	mux.HandleFunc("/callback", runtime.callback)
	runtime.handler = mux
	return runtime, nil
}

func (r *Runtime) Descriptor() pluginapi.Descriptor { return r.descriptor }
func (r *Runtime) Handler() http.Handler            { return r.handler }
func (r *Runtime) Readiness(_ context.Context) idp.ReadinessCheck {
	ready := !r.closed.Load() && r.signer != nil && (r.policy == nil || r.policy.Ready())
	reason := ""
	if !ready {
		reason = "jitsi_runtime_unavailable"
	}
	return idp.ReadinessCheck{Name: "plugin.jitsi", Ready: ready, Reason: reason, CheckedAt: r.services.Clock.Now().UTC()}
}

func (r *Runtime) Close(ctx context.Context) error {
	if !r.closed.CompareAndSwap(false, true) {
		return nil
	}
	var policyErr error
	if r.policy != nil {
		policyErr = r.policy.Close(ctx)
	}
	return errors.Join(policyErr, r.signer.Close())
}

func (r *Runtime) start(writer http.ResponseWriter, request *http.Request) {
	started := time.Now()
	ctx, span := r.tracer.Start(request.Context(), "tinyidp.jitsi.start",
		trace.WithAttributes(attribute.String("plugin", "jitsi"), attribute.String("operation", "start")))
	request = request.WithContext(ctx)
	outcome, reasonClass := "failed", "validation"
	defer func() {
		r.observe(request.Context(), "start", outcome, reasonClass, time.Since(started))
		span.SetAttributes(attribute.String("outcome", outcome), attribute.String("reason_class", reasonClass))
		if outcome == "failed" {
			span.SetStatus(codes.Error, reasonClass)
		}
		span.End()
	}()
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		r.renderError(writer, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	room, ok := singleQuery(request.URL.Query(), "room")
	if !ok || !roomPattern.MatchString(room) {
		r.reject(request.Context(), writer, "invalid_room", nil)
		return
	}
	registration, selectAccount, ok := startIntent(request.URL.Query())
	if !ok {
		r.reject(request.Context(), writer, "invalid_start_intent", nil)
		return
	}
	binding, err := r.browserBinding(writer, request)
	if err != nil {
		r.reject(request.Context(), writer, "authentication_start_failed", err)
		return
	}
	state, _ := json.Marshal(map[string]string{"room": room, "tenant": r.settings.XMPPDomain})
	result, err := r.services.OIDC.Start(request.Context(), pluginapi.StartRequest{
		PluginID: r.descriptor.ID, ClientID: r.settings.OIDCClientID,
		CallbackPath: r.descriptor.RoutePrefix() + "callback",
		Scopes:       []string{"openid", "profile", "email"}, PluginState: state,
		BrowserBinding: binding, Registration: registration, SelectAccount: selectAccount,
		TTL: 10 * time.Minute,
	})
	if err != nil {
		reasonClass = "oauth"
		r.reject(request.Context(), writer, "authentication_start_failed", err)
		return
	}
	outcome, reasonClass = "accepted", "none"
	r.oidc.Add(request.Context(), 1, metric.WithAttributes(
		attribute.String("plugin", "jitsi"), attribute.String("operation", "start"), attribute.String("outcome", "accepted"),
	))
	http.Redirect(writer, request, result.AuthorizationURL, http.StatusSeeOther)
}

func startIntent(query url.Values) (bool, bool, bool) {
	intent := query["intent"]
	if len(intent) > 1 || (len(intent) == 1 && intent[0] != "" && intent[0] != "signup") {
		return false, false, false
	}
	prompt := query["prompt"]
	if len(prompt) > 1 || (len(prompt) == 1 && prompt[0] != "" && prompt[0] != "select_account") {
		return false, false, false
	}
	registration := len(intent) == 1 && intent[0] == "signup"
	selectAccount := len(prompt) == 1 && prompt[0] == "select_account"
	if registration && selectAccount {
		return false, false, false
	}
	return registration, selectAccount, true
}

func (r *Runtime) callback(writer http.ResponseWriter, request *http.Request) {
	started := time.Now()
	ctx, span := r.tracer.Start(request.Context(), "tinyidp.jitsi.callback",
		trace.WithAttributes(attribute.String("plugin", "jitsi"), attribute.String("operation", "callback")))
	request = request.WithContext(ctx)
	outcome, reasonClass := "failed", "validation"
	defer func() {
		r.observe(request.Context(), "callback", outcome, reasonClass, time.Since(started))
		span.SetAttributes(attribute.String("outcome", outcome), attribute.String("reason_class", reasonClass))
		if outcome != "accepted" {
			span.SetStatus(codes.Error, reasonClass)
		}
		span.End()
	}()
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		r.renderError(writer, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if oauthError := request.URL.Query().Get("error"); oauthError != "" {
		outcome, reasonClass = "denied", "oauth"
		r.reject(request.Context(), writer, "authentication_canceled", errors.New("OIDC authorization returned an error"))
		return
	}
	state, stateOK := singleQuery(request.URL.Query(), "state")
	code, codeOK := singleQuery(request.URL.Query(), "code")
	cookie, err := request.Cookie(bindingCookieName)
	if !stateOK || !codeOK || err != nil || cookie.Value == "" {
		r.reject(request.Context(), writer, "authentication_callback_rejected", err)
		return
	}
	completion, err := r.services.OIDC.Complete(request.Context(), pluginapi.CompleteRequest{
		PluginID: r.descriptor.ID, BrowserBinding: cookie.Value, State: state, Code: code,
	})
	if err != nil {
		reasonClass = "oauth"
		r.oidc.Add(request.Context(), 1, metric.WithAttributes(
			attribute.String("plugin", "jitsi"), attribute.String("operation", "complete"), attribute.String("outcome", "failed"),
		))
		r.reject(request.Context(), writer, "authentication_callback_rejected", err)
		return
	}
	r.oidc.Add(request.Context(), 1, metric.WithAttributes(
		attribute.String("plugin", "jitsi"), attribute.String("operation", "complete"), attribute.String("outcome", "accepted"),
	))
	var pluginState struct {
		Room   string `json:"room"`
		Tenant string `json:"tenant"`
	}
	if err := json.Unmarshal(completion.PluginState, &pluginState); err != nil ||
		!roomPattern.MatchString(pluginState.Room) || pluginState.Tenant != r.settings.XMPPDomain {
		r.reject(request.Context(), writer, "authentication_callback_rejected", err)
		return
	}
	decision := Decision{
		Allowed: true, DisplayName: completion.Identity.Name,
		IncludeEmail: completion.Identity.EmailVerified, Moderator: false,
	}
	if strings.TrimSpace(decision.DisplayName) == "" {
		decision.DisplayName = completion.Identity.PreferredUsername
	}
	if r.policy != nil {
		decision, err = r.policy.Authorize(request.Context(), PolicyInputFromIdentity(
			completion.Identity, r.descriptor.ID, pluginState.Room, pluginState.Tenant,
		))
		if err != nil {
			reasonClass = "policy"
			r.policies.Add(request.Context(), 1, metric.WithAttributes(
				attribute.String("plugin", "jitsi"), attribute.String("outcome", "failed"),
			))
			r.reject(request.Context(), writer, "policy_unavailable", err)
			return
		}
		policyOutcome := "accepted"
		if !decision.Allowed {
			policyOutcome = "denied"
		}
		r.policies.Add(request.Context(), 1, metric.WithAttributes(
			attribute.String("plugin", "jitsi"), attribute.String("outcome", policyOutcome),
		))
	}
	if !decision.Allowed {
		outcome, reasonClass = "denied", "policy"
		r.reject(request.Context(), writer, decision.DiagnosticID, nil)
		return
	}
	token, err := r.signer.Issue(IssueRequest{Identity: completion.Identity, Room: pluginState.Room, Decision: decision})
	if err != nil {
		reasonClass = "signing"
		r.reject(request.Context(), writer, "token_issue_failed", err)
		return
	}
	if err := r.services.Audit.Emit(request.Context(), idp.Event{
		Time: r.services.Clock.Now().UTC(), Name: "integration.jitsi.token_issued",
		ClientID: r.settings.OIDCClientID, Subject: completion.Identity.Subject, Result: "accepted",
		Fields: map[string]string{"room": pluginState.Room, "moderator": boolString(decision.Moderator)},
	}); err != nil {
		reasonClass = "audit"
		r.renderError(writer, http.StatusServiceUnavailable, "audit_delivery_failed")
		return
	}
	outcome, reasonClass = "accepted", "none"
	r.tokens.Add(request.Context(), 1, metric.WithAttributes(
		attribute.String("plugin", "jitsi"), attribute.String("outcome", "accepted"),
	))
	target := r.settings.PublicOrigin + "/" + url.PathEscape(pluginState.Room) + "?jwt=" + url.QueryEscape(token)
	r.renderMeetingTransition(writer, target)
}

func (r *Runtime) observe(ctx context.Context, operation, outcome, reasonClass string, elapsed time.Duration) {
	attributes := metric.WithAttributes(
		attribute.String("plugin", "jitsi"),
		attribute.String("operation", operation),
		attribute.String("outcome", outcome),
		attribute.String("reason_class", reasonClass),
	)
	r.requests.Add(ctx, 1, attributes)
	r.duration.Record(ctx, elapsed.Seconds(), attributes)
}

func (r *Runtime) browserBinding(writer http.ResponseWriter, request *http.Request) (string, error) {
	if cookie, err := request.Cookie(bindingCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	randomSource := r.services.Random
	if randomSource == nil {
		randomSource = rand.Reader
	}
	value := make([]byte, 32)
	if _, err := io.ReadFull(randomSource, value); err != nil {
		return "", err
	}
	binding := base64.RawURLEncoding.EncodeToString(value)
	http.SetCookie(writer, &http.Cookie{
		Name: bindingCookieName, Value: binding, Path: r.descriptor.RoutePrefix(),
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 1800,
	})
	return binding, nil
}

func (r *Runtime) reject(ctx context.Context, writer http.ResponseWriter, reason string, err error) {
	event := r.services.Logger.Error()
	if err != nil {
		event = event.Err(err)
	}
	event.Str("plugin", "jitsi").Str("reason", reason).Msg("Jitsi integration request rejected")
	if auditErr := r.services.Audit.Emit(ctx, idp.Event{
		Time: r.services.Clock.Now().UTC(), Name: "integration.jitsi.rejected",
		ClientID: r.settings.OIDCClientID, Result: "rejected", Reason: reason,
	}); auditErr != nil {
		r.services.Logger.Error().
			Err(auditErr).
			Str("plugin", "jitsi").
			Str("reason", reason).
			Msg("Jitsi rejection audit delivery failed")
		r.renderError(writer, http.StatusServiceUnavailable, "audit_delivery_failed")
		return
	}
	r.renderError(writer, http.StatusBadRequest, reason)
}

var errorPage = template.Must(template.New("jitsi-error").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Meeting access was not completed</title><link rel="stylesheet" href="/static/themes/jitsi.css"></head>
<body><main class="identity workflow"><header><p class="kicker">Jitsi / identity</p>
<h1>Meeting access was not completed</h1></header><p role="alert">{{.}}</p>
<p><a href="/">Return to the meeting site</a></p></main></body></html>`))

var meetingTransitionPage = template.Must(template.New("jitsi-transition").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="refresh" content="0; url={{.}}">
<title>Continue to the meeting</title><link rel="stylesheet" href="/static/themes/jitsi.css"></head>
<body><main class="identity workflow"><header><p class="kicker">Jitsi / identity</p>
<h1>Continue to the meeting</h1></header>
<p>Your identity was accepted and the meeting token was issued.</p>
<p><a href="{{.}}">Enter the meeting</a></p></main></body></html>`))

func (r *Runtime) renderMeetingTransition(writer http.ResponseWriter, target string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusOK)
	_ = meetingTransitionPage.Execute(writer, target)
}

func (r *Runtime) renderError(writer http.ResponseWriter, status int, reason string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_ = errorPage.Execute(writer, publicErrorMessage(reason))
}

func publicErrorMessage(reason string) string {
	switch reason {
	case "invalid_room":
		return "Choose a valid meeting room and try again."
	case "verified_email_required":
		return "A verified email address is required for this meeting."
	case "authentication_canceled":
		return "Authentication was canceled. Start again when you are ready."
	case "meeting_access_denied":
		return "Your account is not permitted to enter this meeting."
	case "method_not_allowed":
		return "This meeting action used an unsupported request method."
	default:
		return "Start the meeting sign-in flow again. No meeting token was issued."
	}
}

func singleQuery(values url.Values, name string) (string, bool) {
	items, ok := values[name]
	return firstNonEmpty(items), ok && len(items) == 1 && len(items[0]) <= 4096
}

func firstNonEmpty(values []string) string {
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return ""
	}
	return values[0]
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
