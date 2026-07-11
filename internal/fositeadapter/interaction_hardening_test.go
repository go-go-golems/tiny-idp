package fositeadapter_test

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/securitytrace"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type interactionFixture struct {
	t      *testing.T
	server *httptest.Server
	client *http.Client
	store  *memory.Store
}

func newInteractionFixture(t *testing.T, consentFactory func(*memory.Store) idp.ConsentPolicy) *interactionFixture {
	return newInteractionFixtureConfigured(t, consentFactory, nil, nil)
}

func newInteractionFixtureWithClock(t *testing.T, consentFactory func(*memory.Store) idp.ConsentPolicy, clock func() time.Time) *interactionFixture {
	return newInteractionFixtureConfigured(t, consentFactory, clock, nil)
}

func newInteractionFixtureConfigured(t *testing.T, consentFactory func(*memory.Store) idp.ConsentPolicy, clock func() time.Time, securityEvents securitytrace.Sink) *interactionFixture {
	t.Helper()
	ctx := context.Background()
	st := memory.New()
	if err := st.PutClient(ctx, idpstore.Client{
		ID:            "spa",
		Public:        true,
		RequirePKCE:   true,
		RedirectURIs:  []string{"http://localhost/callback"},
		AllowedScopes: []string{"openid", "email"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Email: "alice@example.test"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-interaction", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	consent := idp.ConsentPolicy(fositeadapter.AlwaysSkipConsent{})
	if consentFactory != nil {
		consent = consentFactory(st)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:         "http://127.0.0.1:5556",
		Store:          st,
		SecretKey:      []byte("interaction-hardening-secret-key-32"),
		Consent:        consent,
		Clock:          clock,
		SecurityEvents: securityEvents,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	t.Cleanup(server.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &interactionFixture{
		t:      t,
		server: server,
		client: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		store:  st,
	}
}

func TestAuthorizationSecurityTraceSatisfiesOfflineMonitor(t *testing.T) {
	recorder := &securitytrace.Recorder{}
	fixture := newInteractionFixtureConfigured(t, nil, nil, recorder)
	fixture.login()
	monitor := securitytrace.NewMonitor()
	for _, event := range recorder.Events() {
		monitor.Observe(event)
	}
	if violations := monitor.Violations(); len(violations) != 0 {
		t.Fatalf("security trace violations=%v events=%#v", violations, recorder.Events())
	}
}

func (f *interactionFixture) begin(values url.Values) (url.Values, string, int) {
	f.t.Helper()
	req, err := http.NewRequest(http.MethodGet, f.server.URL+"/authorize?"+values.Encode(), nil)
	if err != nil {
		f.t.Fatal(err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		f.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		f.t.Fatal(err)
	}
	return parseInteractionInputs(string(body)), string(body), resp.StatusCode
}

func (f *interactionFixture) submit(values url.Values) *http.Response {
	f.t.Helper()
	req, err := http.NewRequest(http.MethodPost, f.server.URL+"/authorize", strings.NewReader(values.Encode()))
	if err != nil {
		f.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := f.client.Do(req)
	if err != nil {
		f.t.Fatal(err)
	}
	return resp
}

func (f *interactionFixture) login() {
	f.t.Helper()
	values := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	values.Del("login")
	form, _, status := f.begin(values)
	if status != http.StatusOK {
		f.t.Fatalf("begin login status=%d", status)
	}
	form.Set("login", "alice")
	form.Set("consent_approved", "true")
	form.Set("action", "approve")
	resp := f.submit(form)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		f.t.Fatalf("login status=%d body=%s", resp.StatusCode, body)
	}
}

var interactionInputPattern = regexp.MustCompile(`<input type="hidden"[^>]+name="([^"]+)"[^>]*value="([^"]*)"[^>]*>`)

func parseInteractionInputs(body string) url.Values {
	values := url.Values{}
	for _, match := range interactionInputPattern.FindAllStringSubmatch(body, -1) {
		values.Add(html.UnescapeString(match[1]), html.UnescapeString(match[2]))
	}
	return values
}

func assertNoAuthorizationCode(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return
	}
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if code := location.Query().Get("code"); code != "" {
		t.Fatalf("authorization code issued unexpectedly: %s", location.String())
	}
}

func TestForcedPromptLoginCannotReuseExistingSession(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	fixture.login()

	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("state", "forced-login-state")
	request.Set("prompt", "login")
	form, body, status := fixture.begin(request)
	if status != http.StatusOK || !strings.Contains(body, `name="password"`) {
		t.Fatalf("forced-login interaction status=%d body=%s", status, body)
	}

	resp := fixture.submit(form)
	defer resp.Body.Close()
	assertNoAuthorizationCode(t, resp)
}

func TestExpiredMaxAgeCannotReuseExistingSession(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	fixture.login()

	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("state", "max-age-zero-state")
	request.Set("max_age", "0")
	form, body, status := fixture.begin(request)
	if status != http.StatusOK || !strings.Contains(body, `name="password"`) {
		t.Fatalf("max_age interaction status=%d body=%s", status, body)
	}

	resp := fixture.submit(form)
	defer resp.Body.Close()
	assertNoAuthorizationCode(t, resp)
}

func TestMalformedMaxAgeNeverRendersCredentialForm(t *testing.T) {
	for _, value := range []string{"not-a-number", "-1", "9223372036854775808"} {
		t.Run(value, func(t *testing.T) {
			fixture := newInteractionFixture(t, nil)
			request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			request.Del("login")
			request.Set("max_age", value)
			_, body, status := fixture.begin(request)
			if status == http.StatusOK && strings.Contains(body, `name="password"`) {
				t.Fatalf("malformed max_age=%q rendered credentials: %s", value, body)
			}
		})
	}
}

func TestConsentDenialUsesOAuthAccessDenied(t *testing.T) {
	fixture := newInteractionFixture(t, func(st *memory.Store) idp.ConsentPolicy {
		return fositeadapter.NewStoredConsent(st, 0)
	})
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("scope", "openid email")
	request.Set("state", "consent-denied-state")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin consent status=%d", status)
	}
	form.Set("login", "alice")
	form.Del("consent_approved")
	form.Set("action", "deny")

	resp := fixture.submit(form)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("consent denial status=%d body=%s", resp.StatusCode, body)
	}
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if got := location.Query().Get("error"); got != "access_denied" {
		t.Fatalf("consent denial error=%q location=%s", got, location.String())
	}
}

func TestAuthorizationStateCannotBeMutatedOnResume(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("state", "original-state")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin status=%d", status)
	}
	form.Set("state", "attacker-state")
	form.Add("state", "second-attacker-state")
	form.Set("login", "alice")

	resp := fixture.submit(form)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("submit status=%d body=%s", resp.StatusCode, body)
	}
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if got := location.Query().Get("state"); got != "original-state" {
		t.Fatalf("returned state=%q, want server-owned original-state", got)
	}
}

func TestConcurrentTabsKeepIndependentInteractions(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	firstRequest := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	firstRequest.Del("login")
	firstRequest.Set("state", "tab-one-state")
	first, _, firstStatus := fixture.begin(firstRequest)
	secondRequest := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	secondRequest.Del("login")
	secondRequest.Set("state", "tab-two-state")
	second, _, secondStatus := fixture.begin(secondRequest)
	if firstStatus != http.StatusOK || secondStatus != http.StatusOK {
		t.Fatalf("begin statuses=(%d,%d)", firstStatus, secondStatus)
	}
	for _, form := range []url.Values{first, second} {
		form.Set("login", "alice")
		resp := fixture.submit(form)
		location, _ := url.Parse(resp.Header.Get("Location"))
		_ = resp.Body.Close()
		if location.Query().Get("code") == "" {
			t.Fatalf("tab interaction failed: status=%d location=%s", resp.StatusCode, location)
		}
	}
}

func TestAuthorizationInteractionIsOneTimeUnderConcurrency(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin status=%d", status)
	}
	form.Set("login", "alice")
	baseURL, _ := url.Parse(fixture.server.URL)
	cookies := fixture.client.Jar.Cookies(baseURL)

	const workers = 2
	results := make(chan bool, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPost, fixture.server.URL+"/authorize", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			for _, cookie := range cookies {
				req.AddCookie(cookie)
			}
			client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			resp, err := client.Do(req)
			if err != nil {
				results <- false
				return
			}
			defer resp.Body.Close()
			location, _ := url.Parse(resp.Header.Get("Location"))
			results <- location.Query().Get("code") != ""
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for success := range results {
		if success {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful terminal authorizations=%d, want exactly 1", successes)
	}
}

func TestInteractionFormContainsNoProtocolContinuation(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, body, status := fixture.begin(request)
	if status != http.StatusOK || form.Get("interaction") == "" || form.Get("csrf_token") == "" {
		t.Fatalf("interaction form status=%d values=%v body=%s", status, form, body)
	}
	if !strings.Contains(body, "Client: <strong>spa</strong>") || !strings.Contains(body, "<code>openid</code>") {
		t.Fatalf("interaction form does not disclose bound client and scopes: %s", body)
	}
	for _, forbidden := range []string{`name="client_id"`, `name="redirect_uri"`, `name="state"`, `name="scope"`, `name="code_challenge"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("browser form contains protocol continuation %s: %s", forbidden, body)
		}
	}
}

func TestAuthorizationInteractionRejectsSequentialReplay(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin status=%d", status)
	}
	form.Set("login", "alice")
	first := fixture.submit(form)
	firstLocation, _ := url.Parse(first.Header.Get("Location"))
	_ = first.Body.Close()
	if firstLocation.Query().Get("code") == "" {
		t.Fatalf("first submission did not issue code: status=%d location=%s", first.StatusCode, firstLocation)
	}
	second := fixture.submit(form)
	defer second.Body.Close()
	assertNoAuthorizationCode(t, second)
}

func TestAuthorizationInteractionRevalidatesClientMutation(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin status=%d", status)
	}
	if err := fixture.store.PutClient(context.Background(), idpstore.Client{
		ID:            "spa",
		Public:        true,
		RequirePKCE:   true,
		RedirectURIs:  []string{"http://localhost/callback"},
		AllowedScopes: []string{"openid", "email"},
		Disabled:      true,
	}); err != nil {
		t.Fatal(err)
	}
	form.Set("login", "alice")
	resp := fixture.submit(form)
	defer resp.Body.Close()
	assertNoAuthorizationCode(t, resp)
}

func TestAuthorizationInteractionExpiresWithInjectedClock(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	fixture := newInteractionFixtureWithClock(t, nil, func() time.Time { return now })
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin status=%d", status)
	}
	now = now.Add(11 * time.Minute)
	form.Set("login", "alice")
	resp := fixture.submit(form)
	defer resp.Body.Close()
	assertNoAuthorizationCode(t, resp)
}

func TestAuthorizationInteractionRevalidatesDisabledSessionUser(t *testing.T) {
	fixture := newInteractionFixture(t, func(st *memory.Store) idp.ConsentPolicy {
		return fositeadapter.NewStoredConsent(st, 0)
	})
	fixture.login()
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("scope", "openid email")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin consent status=%d", status)
	}
	if err := fixture.store.PutUser(context.Background(), "alice", idpstore.User{ID: "u1", Sub: "user-alice", Email: "alice@example.test", Disabled: true}); err != nil {
		t.Fatal(err)
	}
	form.Set("action", "approve")
	resp := fixture.submit(form)
	defer resp.Body.Close()
	assertNoAuthorizationCode(t, resp)
}
