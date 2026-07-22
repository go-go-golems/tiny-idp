package fositeadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/assurance"
	"github.com/go-go-golems/tiny-idp/internal/fositeadapter"
	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/securitytrace"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestProviderOwnedRegistrationResumesPKCEAuthorization(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{
		ID:                "message-desk",
		Public:            true,
		RequirePKCE:       true,
		RedirectURIs:      []string{"http://localhost/callback"},
		AllowedScopes:     []string{"openid", "profile"},
		AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode},
	}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("registration-test-key", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	audit := idp.NewMemorySink()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:        "http://127.0.0.1:5556",
		Store:         store,
		SecretKey:     []byte("provider-registration-test-secret-key"),
		Audit:         audit,
		Authenticator: accounts,
		Consent:       fositeadapter.AlwaysSkipConsent{},
		Registration:  fositeadapter.RegistrationConfig{Enabled: true, Accounts: accounts},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("client_id", "message-desk")
	request.Set("scope", "openid profile")
	request.Set("tinyidp_signup", "1")
	response, err := client.Get(server.URL + "/authorize?" + request.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="`+idpui.PasswordConfirmationFieldName+`"`) {
		t.Fatalf("registration page status=%d body=%s", response.StatusCode, body)
	}
	form := parseInteractionInputs(string(body))
	form.Set(idpui.ActionFieldName, "submit")
	form.Set("email", "new-user@example.test")
	form.Set(idpui.DisplayNameFieldName, "New User")
	form.Set(idpui.PasswordFieldName, "correct horse battery staple 2026")
	form.Set(idpui.PasswordConfirmationFieldName, "correct horse battery staple 2026")
	crossSiteRequest, err := http.NewRequest(http.MethodPost, server.URL+"/authorize", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	crossSiteRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	crossSiteRequest.Header.Set("Origin", "https://attacker.example.test")
	crossSiteRequest.Header.Set("Sec-Fetch-Site", "cross-site")
	crossSiteResponse, err := client.Do(crossSiteRequest)
	if err != nil {
		t.Fatal(err)
	}
	crossSiteBody, err := io.ReadAll(crossSiteResponse.Body)
	crossSiteResponse.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if crossSiteResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site registration status=%d", crossSiteResponse.StatusCode)
	}
	if contentType := crossSiteResponse.Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("cross-site registration content type=%q", contentType)
	}
	if body := string(crossSiteBody); !strings.Contains(body, "Registration could not be completed") || !strings.Contains(body, "Restart registration from the application") || strings.Contains(body, "new-user@example.test") || strings.Contains(body, "correct horse battery staple") || strings.Contains(body, "<form") {
		t.Fatalf("unsafe or incomplete themed rejection body=%s", body)
	}
	if _, err := store.GetUserByLogin(ctx, "new-user@example.test"); err == nil {
		t.Fatal("cross-site registration created an account")
	}

	response = submitRegistration(t, client, server.URL, form)
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("registration status=%d body=%s", response.StatusCode, body)
	}
	location, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Query().Get("code") == "" || location.Query().Get("state") != request.Get("state") {
		t.Fatalf("registration did not resume authorization: %s", location)
	}
	user, err := store.GetUserByLogin(ctx, "new-user@example.test")
	if err != nil || user.Sub == "" || user.Name != "New User" {
		t.Fatalf("registered user=%#v err=%v", user, err)
	}

	secondRequest := authorizeForm("secondABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnop")
	secondRequest.Del("login")
	secondRequest.Set("client_id", "message-desk")
	secondRequest.Set("scope", "openid profile")
	secondRequest.Set("state", "second-registration-state")
	secondRequest.Set("tinyidp_signup", "1")
	secondResponse, err := client.Get(server.URL + "/authorize?" + secondRequest.Encode())
	if err != nil {
		t.Fatal(err)
	}
	secondBody, err := io.ReadAll(secondResponse.Body)
	secondResponse.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if secondResponse.StatusCode != http.StatusOK || !strings.Contains(string(secondBody), `name="`+idpui.PasswordConfirmationFieldName+`"`) {
		t.Fatalf("registration with active provider session status=%d body=%s", secondResponse.StatusCode, secondBody)
	}

	replay := submitRegistration(t, client, server.URL, form)
	defer replay.Body.Close()
	if replay.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(replay.Body)
		t.Fatalf("registration replay status=%d body=%s", replay.StatusCode, body)
	}
	for _, event := range audit.Events() {
		for key, value := range event.Fields {
			if strings.Contains(strings.ToLower(key+value), "password") {
				t.Fatalf("registration audit leaks password material: %#v", event)
			}
		}
	}
}

func TestScriptedSignupDoesNotRequireLegacyRegistrationOption(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	requireRegistrationClient(t, ctx, store)
	key, err := keys.GenerateRSA("scripted-signup-without-legacy-registration", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	audit := idp.NewMemorySink()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.DefaultSource, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(context.Background())
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:                "http://127.0.0.1:5556",
		Store:                 store,
		SecretKey:             []byte("scripted-signup-without-legacy-key"),
		Audit:                 audit,
		Authenticator:         accounts,
		Consent:               fositeadapter.AlwaysSkipConsent{},
		ScriptedSignupManager: manager,
		WorkflowContinuations: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := server.Client()
	client.Jar = jar
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("client_id", "message-desk")
	request.Set("scope", "openid profile")
	request.Set("tinyidp_signup", "1")
	response, err := client.Get(server.URL + "/authorize?" + request.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="`+idpui.WorkflowContinuationFieldName+`"`) {
		t.Fatalf("scripted signup status=%d body=%s", response.StatusCode, body)
	}
	firstForm := parseInteractionInputs(string(body))
	firstForm.Set(idpui.ActionFieldName, string(idpworkflow.ActionSubmit))
	firstForm.Set(idpui.DisplayNameFieldName, "First User")
	firstForm.Set(string(idpworkflow.FieldEmail), "first-user@example.test")
	firstForm.Set(idpui.PasswordFieldName, "correct horse battery staple 2026")
	firstForm.Set(idpui.PasswordConfirmationFieldName, "correct horse battery staple 2026")
	firstResponse := submitRegistration(t, client, server.URL, firstForm)
	firstResponse.Body.Close()
	if firstResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("first scripted signup status=%d", firstResponse.StatusCode)
	}

	request.Set("state", "remembered-browser-signup")
	response, err = client.Get(server.URL + "/authorize?" + request.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("second scripted signup page status=%d body=%s", response.StatusCode, body)
	}
	secondForm := parseInteractionInputs(string(body))
	secondForm.Set(idpui.ActionFieldName, string(idpworkflow.ActionSubmit))
	secondForm.Set(idpui.DisplayNameFieldName, "Second User")
	secondForm.Set(string(idpworkflow.FieldEmail), "second-user@example.test")
	secondForm.Set(idpui.PasswordFieldName, "correct horse battery staple 2026")
	secondForm.Set(idpui.PasswordConfirmationFieldName, "different horse battery staple 2026")
	secondResponse := submitRegistration(t, client, server.URL, secondForm)
	defer secondResponse.Body.Close()
	secondBody, err := io.ReadAll(secondResponse.Body)
	if err != nil {
		t.Fatal(err)
	}
	if secondResponse.StatusCode != http.StatusBadRequest || !strings.Contains(secondResponse.Header.Get("Content-Type"), "text/html") || !strings.Contains(string(secondBody), `name="`+idpui.WorkflowContinuationFieldName+`"`) {
		t.Fatalf("remembered-browser signup POST status=%d body=%s", secondResponse.StatusCode, secondBody)
	}
	for _, event := range audit.Events() {
		if event.Name == "workflow.signup.resume_rejected" {
			t.Fatalf("fresh signup continuation was rejected in remembered browser: %#v", event)
		}
	}
}

func TestDurableInvitationSignupValidatesThenConsumesAtomically(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "goja-auth-host-demo", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	signingKey, err := keys.GenerateRSA("durable-invitation-signup", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, signingKey); err != nil {
		t.Fatal(err)
	}
	audit := idp.NewMemorySink()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.InviteRequiredSource, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(context.Background())
	invitations, err := idpinvite.NewDurableService(store, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := invitations.Issue(ctx, idpinvite.DurableIssue{Code: "valid-invite", ID: "invite-1", Audience: "goja-auth-host-demo", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: []byte("durable-invitation-signup-secret"), Audit: audit,
		Authenticator: accounts, Consent: fositeadapter.AlwaysSkipConsent{}, Registration: fositeadapter.RegistrationConfig{Enabled: true, Accounts: accounts},
		ScriptedSignupManager: manager, WorkflowContinuations: store, DurableInvitations: invitations,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	authorize := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	authorize.Del("login")
	authorize.Set("client_id", "goja-auth-host-demo")
	authorize.Set("scope", "openid profile")
	authorize.Set("tinyidp_signup", "1")
	response, err := client.Get(server.URL + "/authorize?" + authorize.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="invite_code"`) {
		t.Fatalf("invite signup page status=%d body=%s", response.StatusCode, body)
	}
	form := parseInteractionInputs(string(body))
	form.Set(idpui.ActionFieldName, "submit")
	form.Set("display_name", "Invited User")
	form.Set("email", "invited@example.test")
	form.Set("password", "correct horse battery staple 2026")
	form.Set("password_confirmation", "correct horse battery staple 2026")
	form.Set("invite_code", "invalid-invite")

	response = submitRegistration(t, client, server.URL, form)
	invalidBody, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(string(invalidBody), `name="invite_code"`) || !strings.Contains(string(invalidBody), `aria-invalid="true"`) {
		t.Fatalf("invalid invite response status=%d body=%s", response.StatusCode, invalidBody)
	}
	if _, err := store.GetUserByLogin(ctx, "invited@example.test"); err == nil {
		t.Fatal("invalid invitation created an account")
	}

	form.Set("invite_code", "valid-invite")
	response = submitRegistration(t, client, server.URL, form)
	response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("valid invite signup status=%d", response.StatusCode)
	}
	if _, err := store.GetUserByLogin(ctx, "invited@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := invitations.Inspect(ctx, "valid-invite", "goja-auth-host-demo", time.Now()); !errors.Is(err, idpstore.ErrAlreadyConsumed) {
		t.Fatalf("consumed invitation inspection error=%v", err)
	}
	consumedAudit := false
	for _, event := range audit.Events() {
		if event.Name == "signup_invitation.consumed" && event.Fields["invitation_id"] == "invite-1" {
			consumedAudit = true
		}
		encoded, _ := json.Marshal(event)
		if strings.Contains(string(encoded), "valid-invite") {
			t.Fatalf("audit leaked raw invitation: %s", encoded)
		}
	}
	if !consumedAudit {
		t.Fatal("missing signup_invitation.consumed audit event")
	}
}

type signupEmailCapture struct {
	requests []idpemailchallenge.MailRequest
	err      error
}

func (m *signupEmailCapture) SendEmailChallenge(_ context.Context, request idpemailchallenge.MailRequest) error {
	m.requests = append(m.requests, request)
	return m.err
}

func TestEmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	requireRegistrationClient(t, ctx, store)
	key, err := keys.GenerateRSA("email-verified-registration-test-key", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	audit := idp.NewMemorySink()
	securityEvents := &securitytrace.Recorder{}
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.EmailVerifiedSource, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(context.Background())
	mail := &signupEmailCapture{}
	emailChallenges, err := idpemailchallenge.NewService(idpemailchallenge.NewMemoryStore(), mail, []byte("email-verified-registration-test-key"))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:                "http://127.0.0.1:5556",
		Store:                 store,
		SecretKey:             []byte("provider-registration-test-secret-key"),
		Audit:                 audit,
		Authenticator:         accounts,
		Consent:               fositeadapter.AlwaysSkipConsent{},
		Registration:          fositeadapter.RegistrationConfig{Enabled: true, Accounts: accounts},
		ScriptedSignupManager: manager,
		WorkflowContinuations: store,
		EmailChallenges:       emailChallenges,
		SecurityEvents:        securityEvents,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("client_id", "message-desk")
	request.Set("scope", "openid profile")
	request.Set("tinyidp_signup", "1")
	response, err := client.Get(server.URL + "/authorize?" + request.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="email"`) || strings.Contains(string(body), `name="password"`) {
		t.Fatalf("initial email-verification page status=%d body=%s", response.StatusCode, body)
	}
	identityForm := parseInteractionInputs(string(body))
	identityForm.Set(idpui.ActionFieldName, "submit")
	identityForm.Set("email", "verified-user@example.test")
	identityForm.Set(idpui.DisplayNameFieldName, "Verified User")

	mail.err = errors.New("test delivery failure")
	response = submitRegistration(t, client, server.URL, identityForm)
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("mail failure signup status=%d", response.StatusCode)
	}
	if _, err := store.GetUserByLogin(ctx, "verified-user@example.test"); err == nil {
		t.Fatal("mailer failure created an account")
	}
	mail.err = nil
	response = submitRegistration(t, client, server.URL, identityForm)
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="email_code"`) || len(mail.requests) != 2 {
		t.Fatalf("email-code page status=%d mail=%d body=%s", response.StatusCode, len(mail.requests), body)
	}
	if _, err := manager.Activate(ctx, strings.Replace(idpsignup.EmailVerifiedSource, "Choose a password", "Choose your password", 1)); err != nil {
		t.Fatal(err)
	}
	resendForm := parseInteractionInputs(string(body))
	resendForm.Set(idpui.ActionFieldName, string(idpworkflow.ActionResend))
	resendForm.Set("email_code", "")
	response = submitRegistration(t, client, server.URL, resendForm)
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(mail.requests) != 3 || mail.requests[1].Code == mail.requests[2].Code {
		t.Fatalf("email resend status=%d mail=%d body=%s", response.StatusCode, len(mail.requests), body)
	}
	codeForm := parseInteractionInputs(string(body))
	codeForm.Set(idpui.ActionFieldName, "submit")
	codeForm.Set("email_code", mail.requests[2].Code)
	attacker := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	attackerResponse := submitRegistration(t, attacker, server.URL, codeForm)
	attackerResponse.Body.Close()
	if attackerResponse.StatusCode == http.StatusOK || attackerResponse.StatusCode == http.StatusSeeOther {
		t.Fatalf("different-browser code submission status=%d", attackerResponse.StatusCode)
	}
	wrongCodeForm := parseInteractionInputs(string(body))
	wrongCodeForm.Set(idpui.ActionFieldName, "submit")
	wrongCodeForm.Set("email_code", mail.requests[1].Code)
	response = submitRegistration(t, client, server.URL, wrongCodeForm)
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong-code status=%d body=%s", response.StatusCode, body)
	}
	codeForm = parseInteractionInputs(string(body))
	codeForm.Set(idpui.ActionFieldName, "submit")
	codeForm.Set("email_code", mail.requests[2].Code)

	response = submitRegistration(t, client, server.URL, codeForm)
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="`+idpui.PasswordFieldName+`"`) || !strings.Contains(string(body), `name="`+idpui.PasswordConfirmationFieldName+`"`) {
		t.Fatalf("password page status=%d body=%s", response.StatusCode, body)
	}
	replay := submitRegistration(t, client, server.URL, codeForm)
	replayBody, err := io.ReadAll(replay.Body)
	replay.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if replay.StatusCode != http.StatusBadRequest {
		t.Fatalf("verified-code replay status=%d", replay.StatusCode)
	}
	if contentType := replay.Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("verified-code replay content type=%q", contentType)
	}
	if body := string(replayBody); !strings.Contains(body, "Registration needs to be restarted") || !strings.Contains(body, "Return to the application and begin registration again.") || strings.Contains(body, "registration request was not accepted") {
		t.Fatalf("verified-code replay has unsafe terminal page: %s", body)
	}
	passwordForm := parseInteractionInputs(string(body))
	passwordForm.Set(idpui.ActionFieldName, "submit")
	passwordForm.Set(idpui.PasswordFieldName, "correct horse battery staple 2026")
	passwordForm.Set(idpui.PasswordConfirmationFieldName, "correct horse battery staple 2026")

	response = submitRegistration(t, client, server.URL, passwordForm)
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		body, _ = io.ReadAll(response.Body)
		t.Fatalf("verified signup status=%d body=%s", response.StatusCode, body)
	}
	user, err := store.GetUserByLogin(ctx, "verified-user@example.test")
	if err != nil || !user.EmailVerified || user.Name != "Verified User" {
		t.Fatalf("verified signup user=%#v err=%v", user, err)
	}
	assertScriptedSignupSecurityTrace(t, securityEvents.Events())
}

func assertScriptedSignupSecurityTrace(t *testing.T, events []securitytrace.Event) {
	t.Helper()
	monitor := securitytrace.NewMonitor()
	model, err := assurance.NewDeclaredLambdaModel([]assurance.OutcomeID{
		assurance.LambdaOutcomePresent,
		assurance.LambdaOutcomeChallenge,
		assurance.LambdaOutcomeCommit,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := map[securitytrace.Kind]assurance.StepID{
		securitytrace.LambdaInvocationStarted:   assurance.StepLambdaInvoke,
		securitytrace.LambdaInvocationCompleted: assurance.StepLambdaInvoke,
		securitytrace.ContinuationCreated:       assurance.StepContinuationCreate,
		securitytrace.ContinuationTerminal:      assurance.StepContinuationConsume,
		securitytrace.EvidenceVerified:          assurance.StepEvidenceVerify,
		securitytrace.EffectValidationCompleted: assurance.StepEffectValidate,
		securitytrace.NativeEffectCommitted:     assurance.StepSignupCommit,
	}
	seen := map[securitytrace.Kind]bool{}
	for _, event := range events {
		monitor.Observe(event)
		result, err := event.Result()
		if err != nil {
			t.Fatalf("invalid scripted-signup trace event=%#v err=%v", event, err)
		}
		if violations := model.Apply(assurance.TraceObservation{Step: result.Step, Kind: result.Observation, Outcome: result.Outcome}); len(violations) != 0 {
			t.Fatalf("scripted-signup lambda model violations=%v event=%#v", violations, event)
		}
		if step, ok := want[event.Kind]; ok {
			if event.Transition != step {
				t.Fatalf("trace kind %q transition=%q want %q", event.Kind, event.Transition, step)
			}
			seen[event.Kind] = true
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"verified-user", "correct horse", "email_code", "password", "token", "secret"} {
			if strings.Contains(string(encoded), forbidden) {
				t.Fatalf("scripted-signup trace leaked %q: %s", forbidden, encoded)
			}
		}
	}
	for kind := range want {
		if !seen[kind] {
			t.Fatalf("missing scripted-signup trace kind %q events=%#v", kind, events)
		}
	}
	if violations := monitor.Violations(); len(violations) != 0 {
		t.Fatalf("scripted-signup trace monitor violations=%v events=%#v", violations, events)
	}
}

func TestEmailVerifiedScriptedSignupSurvivesSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "idp.db")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(databasePath))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutClient(ctx, idpstore.Client{ID: "message-desk", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("email-verified-restart-test-key", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	mail := &signupEmailCapture{}
	const challengeKey = "email-verified-restart-challenge-key"
	provider, executor := newEmailVerifiedSignupProvider(t, ctx, store, mail, []byte(challengeKey))
	server := httptest.NewServer(provider.Handler())
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("client_id", "message-desk")
	request.Set("scope", "openid profile")
	request.Set("tinyidp_signup", "1")
	response, err := client.Get(server.URL + "/authorize?" + request.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	identityForm := parseInteractionInputs(string(body))
	identityForm.Set(idpui.ActionFieldName, "submit")
	identityForm.Set("email", "restart-user@example.test")
	identityForm.Set(idpui.DisplayNameFieldName, "Restart User")
	response = submitRegistration(t, client, server.URL, identityForm)
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(mail.requests) != 1 || !strings.Contains(string(body), `name="email_code"`) {
		t.Fatalf("pre-restart code page status=%d mail=%d body=%s", response.StatusCode, len(mail.requests), body)
	}
	codeForm := parseInteractionInputs(string(body))
	codeForm.Set(idpui.ActionFieldName, "submit")
	codeForm.Set("email_code", mail.requests[0].Code)
	firstURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	cookies := jar.Cookies(firstURL)
	server.Close()
	if err := executor.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = sqlitestore.Open(ctx, sqlitestore.DefaultConfig(databasePath))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider, executor = newEmailVerifiedSignupProvider(t, ctx, store, mail, []byte(challengeKey))
	defer executor.Close(context.Background())
	server = httptest.NewServer(provider.Handler())
	defer server.Close()
	secondURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(secondURL, cookies)

	response = submitRegistration(t, client, server.URL, codeForm)
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="`+idpui.PasswordFieldName+`"`) {
		t.Fatalf("post-restart password page status=%d body=%s", response.StatusCode, body)
	}
	passwordForm := parseInteractionInputs(string(body))
	passwordForm.Set(idpui.ActionFieldName, "submit")
	passwordForm.Set(idpui.PasswordFieldName, "correct horse battery staple 2026")
	passwordForm.Set(idpui.PasswordConfirmationFieldName, "correct horse battery staple 2026")
	response = submitRegistration(t, client, server.URL, passwordForm)
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		body, _ = io.ReadAll(response.Body)
		t.Fatalf("post-restart signup status=%d body=%s", response.StatusCode, body)
	}
	user, err := store.GetUserByLogin(ctx, "restart-user@example.test")
	if err != nil || !user.EmailVerified {
		t.Fatalf("restart signup user=%#v err=%v", user, err)
	}
}

func newEmailVerifiedSignupProvider(t *testing.T, ctx context.Context, store idpstore.Store, mail idpemailchallenge.Mailer, challengeKey []byte) (*fositeadapter.Provider, *idpsignup.Executor) {
	t.Helper()
	audit := idp.NewMemorySink()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	executor, err := idpsignup.New(ctx, idpsignup.EmailVerifiedSource, 1)
	if err != nil {
		t.Fatal(err)
	}
	emailStore, ok := store.(idpemailchallenge.Store)
	if !ok {
		executor.Close(context.Background())
		t.Fatal("test store does not implement email challenge storage")
	}
	emailChallenges, err := idpemailchallenge.NewService(emailStore, mail, challengeKey)
	if err != nil {
		executor.Close(context.Background())
		t.Fatal(err)
	}
	continuations, ok := store.(idpcontinuation.Store)
	if !ok {
		executor.Close(context.Background())
		t.Fatal("test store does not implement workflow continuation storage")
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: []byte("provider-registration-test-secret-key"), Audit: audit, Authenticator: accounts, Consent: fositeadapter.AlwaysSkipConsent{}, Registration: fositeadapter.RegistrationConfig{Enabled: true, Accounts: accounts}, ScriptedSignup: executor, WorkflowContinuations: continuations, EmailChallenges: emailChallenges})
	if err != nil {
		executor.Close(context.Background())
		t.Fatal(err)
	}
	return provider, executor
}

func requireRegistrationClient(t *testing.T, ctx context.Context, store *memory.Store) {
	t.Helper()
	if err := store.PutClient(ctx, idpstore.Client{
		ID:                "message-desk",
		Public:            true,
		RequirePKCE:       true,
		RedirectURIs:      []string{"http://localhost/callback"},
		AllowedScopes:     []string{"openid", "profile"},
		AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestProviderOwnedRegistrationRejectsMalformedIntent(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("tinyidp_signup", "true")
	response, err := fixture.client.Get(fixture.server.URL + "/authorize?" + request.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound && response.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("malformed intent status=%d body=%s", response.StatusCode, body)
	}
	location, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Query().Get("error") != "invalid_request" {
		t.Fatalf("malformed intent redirect=%s", location)
	}
}

func submitRegistration(t *testing.T, client *http.Client, baseURL string, form url.Values) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, baseURL+"/authorize", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", baseURL)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
