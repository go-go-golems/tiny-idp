package embeddedidp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/fositeadapter"
	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

type Provider struct {
	handler                http.Handler
	adapter                *fositeadapter.Provider
	store                  idpstore.Store
	closed                 atomic.Bool
	mode                   idpstore.Mode
	audit                  idp.Sink
	limiter                idp.RateLimiter
	scriptedSignup         *idpsignup.Executor
	scriptedSignupManager  *idpsignup.GenerationManager
	tokenSecretReady       bool
	maintenanceConfig      MaintenanceConfig
	maintenanceRunMu       sync.Mutex
	maintenanceStatusMu    sync.Mutex
	maintenanceStatus      idp.MaintenanceStatus
	createdAt              time.Time
	lifecycleAuditFailures atomic.Uint64
	healthPath             string
	readyPath              string
}

// tinyidp:development-default -- production validation runs before the development fallback.
func New(ctx context.Context, opts Options) (*Provider, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if opts.Mode == "" {
		opts.Mode = idpstore.DevMode
	}
	if err := opts.Validate(ctx); err != nil {
		return nil, err
	}
	clients, err := opts.Store.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("list clients for maintenance policy: %w", err)
	}
	maintenance, err := normalizeMaintenance(opts.Maintenance, clients)
	if err != nil {
		return nil, err
	}
	if opts.Audit == nil {
		opts.Audit = idp.NoopSink{}
	}
	adapter, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: opts.Issuer, Store: opts.Store, SecretKey: opts.Token.SecretKey, Mode: opts.Mode, CookieSecure: opts.Cookie.Secure, CookieSameSite: opts.Cookie.SameSite, SessionCookieName: opts.Cookie.SessionName, CSRFCookieName: opts.Cookie.CSRFName, CookiePath: opts.Cookie.Path, AccountChooser: fositeadapter.AccountChooserConfig{Enabled: opts.AccountChooser.Enabled, ContextCookieName: opts.AccountChooser.ContextCookieName, ContextTTL: opts.AccountChooser.ContextTTL, MaxRememberedAccounts: opts.AccountChooser.MaxRememberedAccounts, RememberOnPasswordLogin: opts.AccountChooser.RememberOnPasswordLogin, DisplayLabel: opts.AccountChooser.DisplayLabel}, Registration: fositeadapter.RegistrationConfig{Enabled: opts.Registration.Enabled, Accounts: opts.Registration.Accounts}, ScriptedSignup: opts.ScriptedSignup.Executor, ScriptedSignupManager: opts.ScriptedSignup.GenerationManager, DurableInvitations: opts.ScriptedSignup.DurableInvitations, EmailChallenges: opts.ScriptedSignup.EmailChallenges, WorkflowContinuations: opts.ScriptedSignup.Continuations, Audit: opts.Audit, Consent: opts.Consent, Authorization: opts.Authorization, RateLimiter: opts.RateLimiter, ClientAddress: opts.ClientAddress, Authenticator: opts.Authenticator, PasswordPolicy: opts.PasswordPolicy, PasswordWork: opts.PasswordWork, InteractionRenderer: opts.UI.Renderer, WorkflowRenderer: opts.UI.WorkflowRenderer, DeviceVerificationRenderer: opts.UI.DeviceVerificationRenderer})
	if err != nil {
		return nil, err
	}
	issuerURL, err := url.Parse(opts.Issuer)
	if err != nil {
		return nil, err
	}
	prefix := issuerURL.EscapedPath()
	if prefix == "/" {
		prefix = ""
	}
	now := time.Now().UTC()
	return &Provider{handler: adapter.Handler(), adapter: adapter, store: opts.Store, mode: opts.Mode, audit: opts.Audit, limiter: opts.RateLimiter, scriptedSignup: opts.ScriptedSignup.Executor, scriptedSignupManager: opts.ScriptedSignup.GenerationManager, tokenSecretReady: len(opts.Token.SecretKey) >= 32, maintenanceConfig: maintenance, createdAt: now, healthPath: prefix + "/healthz", readyPath: prefix + "/readyz"}, nil
}

func (p *Provider) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == p.healthPath {
			writeReport(w, p.Liveness(r.Context()))
			return
		}
		if r.URL.Path == p.readyPath {
			writeReport(w, p.Readiness(r.Context()))
			return
		}
		if p.closed.Load() {
			http.Error(w, "provider closed", http.StatusServiceUnavailable)
			return
		}
		p.handler.ServeHTTP(w, r)
	})
}

func writeReport(w http.ResponseWriter, report idp.ReadinessReport) {
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	if !report.Ready {
		status = http.StatusServiceUnavailable
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(report)
}

// PasswordWorkStats reports non-secret Argon2 capacity and saturation metrics
// when the configured authenticator exposes them.
func (p *Provider) PasswordWorkStats() (idp.PasswordWorkStats, bool) {
	if p == nil || p.adapter == nil {
		return idp.PasswordWorkStats{}, false
	}
	return p.adapter.PasswordWorkStats()
}

func (p *Provider) InteractionRenderStats() idpui.RenderStats {
	if p == nil || p.adapter == nil {
		return idpui.RenderStats{}
	}
	return p.adapter.InteractionRenderStats()
}

func (p *Provider) Readiness(ctx context.Context) idp.ReadinessReport {
	now := time.Now().UTC()
	report := idp.ReadinessReport{Ready: true}
	add := func(name string, ready, degraded bool, reason string) {
		check := idp.ReadinessCheck{Name: name, Ready: ready, Degraded: degraded, Reason: reason, CheckedAt: now}
		if !ready {
			report.Ready = false
		}
		report.Checks = append(report.Checks, check)
	}
	if ctx == nil {
		add("context", false, false, "context_required")
		return report
	}
	if p.closed.Load() {
		add("lifecycle", false, false, "provider_closed")
		return report
	}
	add("lifecycle", true, false, "")
	_, err := p.store.ListClients(ctx)
	add("store", err == nil, false, reasonIf(err, "store_unavailable"))
	if schema, ok := p.store.(idpstore.SchemaReporter); ok {
		version, schemaErr := schema.SchemaVersion(ctx)
		ready := schemaErr == nil && version > 0 && version == schema.SupportedSchemaVersion()
		add("schema", ready || p.mode != idpstore.ProductionMode, !ready && p.mode != idpstore.ProductionMode, reasonIfCondition(ready, "schema_unsupported"))
	} else {
		add("schema", p.mode != idpstore.ProductionMode, p.mode != idpstore.ProductionMode, "schema_reporting_unavailable")
	}
	key, keyErr := p.store.ActiveSigningKey(ctx)
	keyReady := keyErr == nil && signingKeyReady(key, now)
	if keyReady {
		verification, verifyErr := p.store.VerificationKeys(ctx)
		keyReady = verifyErr == nil && verificationKeysReady(verification)
	}
	add("signing_key", keyReady, false, reasonIfCondition(keyReady, "signing_key_unavailable_or_invalid"))
	add("token_secret", p.tokenSecretReady || p.mode != idpstore.ProductionMode, !p.tokenSecretReady && p.mode != idpstore.ProductionMode, reasonIfCondition(p.tokenSecretReady, "ephemeral_or_short_token_secret"))
	if audit, ok := p.audit.(idp.AuditReporter); ok {
		health := audit.AuditHealth(ctx)
		ready := health.Ready && audit.ProductionReady() && p.adapter.AuditDeliveryFailures() == 0 && p.lifecycleAuditFailures.Load() == 0
		add("audit", ready || p.mode != idpstore.ProductionMode, !ready && p.mode != idpstore.ProductionMode, reasonIfCondition(ready, firstReason(health.Reason, "audit_delivery_failure")))
	} else {
		add("audit", p.mode != idpstore.ProductionMode, p.mode != idpstore.ProductionMode, "durable_audit_unavailable")
	}
	limiterReady := false
	if limiter, ok := p.limiter.(idp.ProductionReadyReporter); ok {
		limiterReady = limiter.ProductionReady()
	}
	add("rate_limiter", limiterReady || p.mode != idpstore.ProductionMode, !limiterReady && p.mode != idpstore.ProductionMode, reasonIfCondition(limiterReady, "production_limiter_unavailable"))
	if p.scriptedSignupManager != nil {
		ready := p.scriptedSignupManager.Ready() == nil
		add("scripted_signup", ready, false, reasonIfCondition(ready, "active_generation_unavailable"))
	} else if p.scriptedSignup != nil {
		ready := p.scriptedSignup.Ready()
		add("scripted_signup", ready, false, reasonIfCondition(ready, "executor_unavailable"))
	}
	p.maintenanceStatusMu.Lock()
	status := p.maintenanceStatus
	p.maintenanceStatusMu.Unlock()
	_, maintenanceSupported := p.store.(idpstore.MaintenanceStore)
	maintenanceReady := maintenanceSupported && status.LastError == ""
	maintenanceReason := ""
	maintenanceDegraded := false
	last := status.LastSuccessAt
	if last.IsZero() {
		if now.Sub(p.createdAt) > 2*p.maintenanceConfig.Interval {
			maintenanceReady = false
			maintenanceReason = "maintenance_never_run"
		} else {
			maintenanceDegraded = true
			maintenanceReason = "maintenance_not_yet_run"
		}
	} else if now.Sub(last) > 2*p.maintenanceConfig.Interval {
		maintenanceReady = false
		maintenanceReason = "maintenance_overdue"
	}
	if status.LastError != "" {
		maintenanceReason = "maintenance_failed"
	}
	add("maintenance", maintenanceReady, maintenanceDegraded && maintenanceReady, maintenanceReason)
	return report
}

// Liveness reports only whether the in-process provider can serve. Dependency
// failures belong to Readiness so orchestrators do not restart healthy code for
// a transient database or audit outage.
func (p *Provider) Liveness(ctx context.Context) idp.ReadinessReport {
	now := time.Now().UTC()
	if ctx == nil {
		return idp.ReadinessReport{Checks: []idp.ReadinessCheck{{Name: "context", Ready: false, Reason: "context_required", CheckedAt: now}}}
	}
	ready := !p.closed.Load()
	reason := ""
	if !ready {
		reason = "provider_closed"
	}
	return idp.ReadinessReport{Ready: ready, Checks: []idp.ReadinessCheck{{Name: "lifecycle", Ready: ready, Reason: reason, CheckedAt: now}}}
}

// RunMaintenance executes one synchronous retention pass. Hosts should call it
// immediately after startup and then at MaintenanceConfig.Interval.
func (p *Provider) RunMaintenance(ctx context.Context) (idpstore.MaintenanceReport, error) {
	maintainer, ok := p.store.(idpstore.MaintenanceStore)
	if !ok {
		return idpstore.MaintenanceReport{}, fmt.Errorf("store does not support maintenance")
	}
	p.maintenanceRunMu.Lock()
	defer p.maintenanceRunMu.Unlock()
	started := time.Now().UTC()
	p.maintenanceStatusMu.Lock()
	p.maintenanceStatus.LastStartedAt = started
	p.maintenanceStatusMu.Unlock()
	policy := idpstore.MaintenancePolicy{RetainExpiredFor: p.maintenanceConfig.RetainExpiredFor, ProtocolStateRetention: p.maintenanceConfig.ProtocolStateRetention, SigningKeyRetention: p.maintenanceConfig.SigningKeyRetention}
	report, err := maintainer.Maintain(ctx, started, policy)
	finished := time.Now().UTC()
	p.maintenanceStatusMu.Lock()
	p.maintenanceStatus.LastFinishedAt = finished
	p.maintenanceStatus.LastReport = report
	if err != nil {
		p.maintenanceStatus.LastError = err.Error()
		p.maintenanceStatusMu.Unlock()
		return report, err
	}
	p.maintenanceStatus.LastSuccessAt = finished
	p.maintenanceStatus.LastError = ""
	p.maintenanceStatusMu.Unlock()
	event := idp.Event{Time: finished, Name: "maintenance.completed", Result: "accepted", Fields: map[string]string{"domain_records": fmt.Sprint(report.DomainRecords), "protocol_records": fmt.Sprint(report.ProtocolRecords), "retired_signing_keys": fmt.Sprint(report.RetiredSigningKeys)}}
	if err := p.audit.Emit(ctx, event); err != nil {
		p.lifecycleAuditFailures.Add(1)
		p.maintenanceStatusMu.Lock()
		p.maintenanceStatus.LastError = "audit_delivery_failed"
		p.maintenanceStatusMu.Unlock()
		return report, fmt.Errorf("%w: %v", idp.ErrAuditDelivery, err)
	}
	return report, nil
}

func (p *Provider) MaintenanceStatus() idp.MaintenanceStatus {
	p.maintenanceStatusMu.Lock()
	defer p.maintenanceStatusMu.Unlock()
	return p.maintenanceStatus
}

func signingKeyReady(key idpstore.SigningKey, now time.Time) bool {
	if key.ID == "" || key.Algorithm != "RS256" || !key.Active || now.Before(key.NotBefore) || (!key.NotAfter.IsZero() && !now.Before(key.NotAfter)) {
		return false
	}
	privateKey, err := keys.ParseRSAPrivateKey(key)
	return err == nil && privateKey.N.BitLen() >= 2048
}

func verificationKeysReady(values []idpstore.SigningKey) bool {
	active := 0
	for _, value := range values {
		if value.Active {
			active++
		} else if value.NotAfter.IsZero() {
			return false
		}
		if value.ID == "" || value.Algorithm != "RS256" {
			return false
		}
		privateKey, err := keys.ParseRSAPrivateKey(value)
		if err != nil || privateKey.N.BitLen() < 2048 {
			return false
		}
	}
	return active == 1
}

func reasonIf(err error, reason string) string {
	if err != nil {
		return reason
	}
	return ""
}
func reasonIfCondition(ok bool, reason string) string {
	if ok {
		return ""
	}
	return reason
}
func firstReason(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (p *Provider) Close(_ context.Context) error {
	p.closed.Store(true)
	return nil
}
