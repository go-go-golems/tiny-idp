package fositeadapter_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/fositeadapter"
	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
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
	crossSiteResponse.Body.Close()
	if crossSiteResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site registration status=%d", crossSiteResponse.StatusCode)
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
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	executor, err := idpsignup.New(ctx, idpsignup.EmailVerifiedSource, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close(context.Background())
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
		ScriptedSignup:        executor,
		WorkflowContinuations: store,
		EmailChallenges:       emailChallenges,
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
	replay.Body.Close()
	if replay.StatusCode != http.StatusBadRequest {
		t.Fatalf("verified-code replay status=%d", replay.StatusCode)
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
