package fositeadapter_test

import (
	"context"
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
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
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
