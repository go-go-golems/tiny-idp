// Command public-two-app-flow exercises both production relying parties
// through their public HTTPS origins. It creates one unique acceptance account
// through MessageDesk, uses that same account to log into goja-auth, verifies
// client-selected themes, and logs out of goja-auth. It deliberately never
// prints credentials, cookies, authorization codes, or CSRF tokens.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	idpIssuer     = "https://idp-message-desk.yolo.scapegoat.dev"
	messageOrigin = "https://message-desk.yolo.scapegoat.dev"
	gojaOrigin    = "https://goja-auth.yolo.scapegoat.dev"
)

type browser struct {
	client *http.Client
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "public two-app flow failed:", err)
		os.Exit(1)
	}
	fmt.Println("PASS public readiness for TinyIDP, MessageDesk, and goja-auth")
	fmt.Println("PASS MessageDesk signup used its client-selected same-origin theme")
	fmt.Println("PASS the same identity logged into goja-auth using its distinct theme")
	fmt.Println("PASS goja-auth session, protected /me, and CSRF logout contracts")
}

func run(ctx context.Context) error {
	loginSuffix, err := randomHex(8)
	if err != nil {
		return err
	}
	passwordSuffix, err := randomHex(16)
	if err != nil {
		return err
	}
	login := "tinyidp-acceptance+" + loginSuffix + "@example.test"
	password := "TinyIDP acceptance " + passwordSuffix + "!"

	messageBrowser, err := newBrowser()
	if err != nil {
		return err
	}
	gojaBrowser, err := newBrowser()
	if err != nil {
		return err
	}
	for _, endpoint := range []string{idpIssuer + "/readyz", messageOrigin + "/readyz", gojaOrigin + "/auth/readyz"} {
		if err := ready(ctx, messageBrowser, endpoint); err != nil {
			return err
		}
	}
	if err := signupThroughMessageDesk(ctx, messageBrowser, login, password); err != nil {
		return err
	}
	return loginAndLogoutThroughGoja(ctx, gojaBrowser, login, password)
}

func newBrowser() (*browser, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	return &browser{client: &http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}, nil
}

func signupThroughMessageDesk(ctx context.Context, b *browser, login, password string) error {
	response, err := b.get(ctx, messageOrigin+"/auth/register?return_to=/")
	if err != nil {
		return err
	}
	authorizeURL, err := redirect(response, messageOrigin)
	if err != nil {
		return fmt.Errorf("begin MessageDesk signup: %w", err)
	}
	if err := authorizeContract(authorizeURL, "tinyidp-message-app", messageOrigin+"/auth/callback", true); err != nil {
		return err
	}
	page, err := b.getPage(ctx, authorizeURL, http.StatusOK)
	if err != nil {
		return fmt.Errorf("load MessageDesk signup: %w", err)
	}
	if err := themeContract(ctx, b, page, "Message Desk", "/static/themes/message-desk.css", "--paper"); err != nil {
		return err
	}
	form := url.Values{
		"action":                {"submit"},
		"interaction":           {hidden(page, "interaction")},
		"workflow_continuation": {hidden(page, "workflow_continuation")},
		"csrf_token":            {hidden(page, "csrf_token")},
		"display_name":          {"TinyIDP Acceptance"},
		"email":                 {login},
		"password":              {password},
		"password_confirmation": {password},
	}
	if err := requireHidden(form, "interaction", "workflow_continuation", "csrf_token"); err != nil {
		return fmt.Errorf("MessageDesk signup form: %w", err)
	}
	response, err = b.postForm(ctx, idpIssuer+"/authorize", form)
	if err != nil {
		return err
	}
	callbackURL, err := approveConsentOrRedirect(ctx, b, response, "Message Desk", "/static/themes/message-desk.css")
	if err != nil {
		return fmt.Errorf("complete MessageDesk consent: %w", err)
	}
	if err := callbackContract(callbackURL, messageOrigin, "/auth/callback"); err != nil {
		return err
	}
	response, err = b.get(ctx, callbackURL)
	if err != nil {
		return err
	}
	if _, err := redirect(response, messageOrigin); err != nil {
		return fmt.Errorf("finish MessageDesk callback: %w", err)
	}
	var session struct {
		Authenticated bool   `json:"authenticated"`
		Subject       string `json:"subject"`
		CSRFToken     string `json:"csrfToken"`
	}
	if err := b.getJSON(ctx, messageOrigin+"/api/session", http.StatusOK, &session); err != nil {
		return err
	}
	if !session.Authenticated || session.Subject == "" || session.CSRFToken == "" {
		return fmt.Errorf("MessageDesk did not establish an authenticated application session")
	}
	return nil
}

func loginAndLogoutThroughGoja(ctx context.Context, b *browser, login, password string) error {
	response, err := b.get(ctx, gojaOrigin+"/auth/login")
	if err != nil {
		return err
	}
	authorizeURL, err := redirect(response, gojaOrigin)
	if err != nil {
		return fmt.Errorf("begin goja-auth login: %w", err)
	}
	if err := authorizeContract(authorizeURL, "goja-auth-host-demo", gojaOrigin+"/auth/callback", false); err != nil {
		return err
	}
	page, err := b.getPage(ctx, authorizeURL, http.StatusOK)
	if err != nil {
		return fmt.Errorf("load goja-auth login: %w", err)
	}
	if err := themeContract(ctx, b, page, "go-go-goja Auth Lab", "/static/themes/goja-auth-lab.css", "color-scheme: dark"); err != nil {
		return err
	}
	form := url.Values{
		"action":      {"approve"},
		"interaction": {hidden(page, "interaction")},
		"csrf_token":  {hidden(page, "csrf_token")},
		"login":       {login},
		"password":    {password},
	}
	if err := requireHidden(form, "interaction", "csrf_token"); err != nil {
		return fmt.Errorf("goja-auth login form: %w", err)
	}
	response, err = b.postForm(ctx, idpIssuer+"/authorize", form)
	if err != nil {
		return err
	}
	callbackURL, err := approveConsentOrRedirect(ctx, b, response, "go-go-goja Auth Lab", "/static/themes/goja-auth-lab.css")
	if err != nil {
		return fmt.Errorf("complete goja-auth consent: %w", err)
	}
	if err := callbackContract(callbackURL, gojaOrigin, "/auth/callback"); err != nil {
		return err
	}
	response, err = b.get(ctx, callbackURL)
	if err != nil {
		return err
	}
	if _, err := redirect(response, gojaOrigin); err != nil {
		return fmt.Errorf("finish goja-auth callback: %w", err)
	}
	var session struct {
		UserID    string `json:"userId"`
		Email     string `json:"email"`
		CSRFToken string `json:"csrfToken"`
	}
	if err := b.getJSON(ctx, gojaOrigin+"/auth/session", http.StatusOK, &session); err != nil {
		return err
	}
	if session.UserID == "" || !strings.EqualFold(session.Email, login) || session.CSRFToken == "" {
		return fmt.Errorf("goja-auth session does not represent the shared TinyIDP identity")
	}
	protected, err := b.get(ctx, gojaOrigin+"/me")
	if err != nil {
		return err
	}
	if _, err := bodyWithStatus(protected, http.StatusOK); err != nil {
		return fmt.Errorf("goja-auth protected route: %w", err)
	}
	logout, err := b.do(ctx, http.MethodPost, gojaOrigin+"/auth/logout", nil, "", http.Header{"X-CSRF-Token": []string{session.CSRFToken}})
	if err != nil {
		return err
	}
	if _, err := bodyWithStatus(logout, http.StatusNoContent); err != nil {
		return fmt.Errorf("goja-auth logout: %w", err)
	}
	after, err := b.get(ctx, gojaOrigin+"/auth/session")
	if err != nil {
		return err
	}
	if _, err := bodyWithStatus(after, http.StatusUnauthorized); err != nil {
		return fmt.Errorf("goja-auth session after logout: %w", err)
	}
	return nil
}

func approveConsentOrRedirect(ctx context.Context, b *browser, response *http.Response, productName, stylesheet string) (string, error) {
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return redirect(response, idpIssuer)
	}
	page, err := bodyWithStatus(response, http.StatusOK)
	if err != nil {
		return "", err
	}
	if !bytes.Contains(page, []byte(productName)) || !bytes.Contains(page, []byte(`href="`+stylesheet+`"`)) {
		return "", fmt.Errorf("consent page lost the selected %s theme", productName)
	}
	form := url.Values{
		"action":      {"approve"},
		"interaction": {hidden(page, "interaction")},
		"csrf_token":  {hidden(page, "csrf_token")},
	}
	if err := requireHidden(form, "interaction", "csrf_token"); err != nil {
		return "", err
	}
	response, err = b.postForm(ctx, idpIssuer+"/authorize", form)
	if err != nil {
		return "", err
	}
	return redirect(response, idpIssuer)
}

func themeContract(ctx context.Context, b *browser, page []byte, productName, stylesheet, cssMarker string) error {
	if !bytes.Contains(page, []byte(productName)) {
		return fmt.Errorf("identity page does not contain product name %q", productName)
	}
	if !bytes.Contains(page, []byte(`href="`+stylesheet+`"`)) {
		return fmt.Errorf("identity page does not select stylesheet %q", stylesheet)
	}
	response, err := b.get(ctx, idpIssuer+stylesheet)
	if err != nil {
		return err
	}
	css, err := bodyWithStatus(response, http.StatusOK)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/css") || response.Header.Get("X-Content-Type-Options") != "nosniff" || !bytes.Contains(css, []byte(cssMarker)) {
		return fmt.Errorf("stylesheet %q failed its content/header contract", stylesheet)
	}
	return nil
}

func authorizeContract(rawURL, clientID, callback string, signup bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "idp-message-desk.yolo.scapegoat.dev" || parsed.Path != "/authorize" {
		return fmt.Errorf("unexpected authorization URL origin or path")
	}
	query := parsed.Query()
	if query.Get("client_id") != clientID || query.Get("redirect_uri") != callback || query.Get("response_type") != "code" || query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") == "" || query.Get("state") == "" || query.Get("nonce") == "" {
		return fmt.Errorf("authorization request for %s violates the public PKCE contract", clientID)
	}
	if signup != (query.Get("tinyidp_signup") == "1") {
		return fmt.Errorf("authorization request for %s has the wrong signup intent", clientID)
	}
	return nil
}

func callbackContract(rawURL, origin, path string) error {
	parsed, err := url.Parse(rawURL)
	want, parseErr := url.Parse(origin)
	if err != nil || parseErr != nil || parsed.Scheme != want.Scheme || parsed.Host != want.Host || parsed.Path != path || parsed.Query().Get("code") == "" || parsed.Query().Get("state") == "" {
		return fmt.Errorf("authorization callback violates the registered redirect contract")
	}
	return nil
}

func ready(ctx context.Context, b *browser, endpoint string) error {
	var last error
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		response, err := b.get(ctx, endpoint)
		if err == nil {
			_, last = bodyWithStatus(response, http.StatusOK)
			if last == nil {
				return nil
			}
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("readiness endpoint did not become healthy: %w", last)
}

func (b *browser) get(ctx context.Context, rawURL string) (*http.Response, error) {
	return b.do(ctx, http.MethodGet, rawURL, nil, "", nil)
}

func (b *browser) getPage(ctx context.Context, rawURL string, status int) ([]byte, error) {
	response, err := b.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return bodyWithStatus(response, status)
}

func (b *browser) getJSON(ctx context.Context, rawURL string, status int, destination any) error {
	response, err := b.get(ctx, rawURL)
	if err != nil {
		return err
	}
	body, err := bodyWithStatus(response, status)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}
	return nil
}

func (b *browser) postForm(ctx context.Context, rawURL string, form url.Values) (*http.Response, error) {
	return b.do(ctx, http.MethodPost, rawURL, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil)
}

func (b *browser) do(ctx context.Context, method, rawURL string, body io.Reader, contentType string, headers http.Header) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if method != http.MethodGet && method != http.MethodHead {
		parsed, parseErr := url.Parse(rawURL)
		if parseErr != nil {
			return nil, parseErr
		}
		request.Header.Set("Origin", parsed.Scheme+"://"+parsed.Host)
	}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	return b.client.Do(request)
}

func bodyWithStatus(response *http.Response, want int) ([]byte, error) {
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != want {
		return nil, fmt.Errorf("HTTP status %d, want %d", response.StatusCode, want)
	}
	return body, nil
}

func redirect(response *http.Response, base string) (string, error) {
	defer response.Body.Close()
	if response.StatusCode < 300 || response.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP status %d is not a redirect", response.StatusCode)
	}
	location := response.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("redirect has no Location header")
	}
	parsed, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() {
		return parsed.String(), nil
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(parsed).String(), nil
}

func hidden(page []byte, name string) string {
	match := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]+)"`).FindSubmatch(page)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}

func requireHidden(values url.Values, names ...string) error {
	for _, name := range names {
		if values.Get(name) == "" {
			return fmt.Errorf("missing hidden field %q", name)
		}
	}
	return nil
}

func randomHex(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate acceptance identity: %w", err)
	}
	return hex.EncodeToString(value), nil
}
