package consumer

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

type fixedAuthenticator struct{ user idpstore.User }

func (a fixedAuthenticator) ProductionReady() bool { return true }
func (a fixedAuthenticator) PasswordWorkStats() idp.PasswordWorkStats {
	return idp.PasswordWorkStats{Capacity: 1}
}

func (a fixedAuthenticator) AuthenticatePassword(_ context.Context, login, password string, _ idp.LoginMetadata) (idp.AuthResult, error) {
	if login != "alice" || password != "correct horse battery staple" {
		return idp.AuthResult{}, context.Canceled
	}
	return idp.AuthResult{User: a.user, AMR: []string{"pwd"}}, nil
}

func TestExternalProductionAuthorizationCodePKCE(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(filepath.Join(dir, "external.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	user := idpstore.User{ID: "user-1", Sub: "subject-1", Email: "alice@example.test", EmailVerified: true, Name: "Alice"}
	if err := store.PutUser(ctx, "alice", user); err != nil {
		t.Fatal(err)
	}
	client := idpstore.Client{ID: "external-spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://client.example.test/callback"}, AllowedScopes: []string{"openid", "profile", "email"}}
	if err := store.PutClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	key := generateSigningKey(t)
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	audit, err := idp.NewFileAuditSink(filepath.Join(dir, "audit", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = audit.Close() }()

	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer:        "https://issuer.example.test",
		Mode:          embeddedidp.ProductionMode,
		Store:         store,
		Cookie:        embeddedidp.CookieConfig{Secure: true},
		Token:         embeddedidp.TokenConfig{SecretKey: []byte("external-flow-secret-key-32-bytes-minimum")},
		Audit:         audit,
		RateLimiter:   idp.NewFixedWindowRateLimiter(10_000, time.Minute),
		ClientAddress: idp.DirectClientAddressResolver{},
		Authenticator: fixedAuthenticator{user: user},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = provider.Close(ctx) }()
	if report := provider.Readiness(ctx); !report.Ready {
		t.Fatalf("provider not ready: %#v", report)
	}

	server := httptest.NewTLSServer(provider.Handler())
	defer server.Close()
	httpClient := server.Client()
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digest := sha256.Sum256([]byte(verifier))
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {client.ID},
		"redirect_uri":          {client.RedirectURIs[0]},
		"scope":                 {"openid profile email"},
		"state":                 {"external-state-123456"},
		"nonce":                 {"external-nonce-123456"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(digest[:])},
		"code_challenge_method": {"S256"},
	}
	interaction, err := httpClient.Get(server.URL + "/authorize?" + form.Encode())
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, interaction)
	csrf := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindSubmatch(body)
	if interaction.StatusCode != http.StatusOK || len(csrf) != 2 {
		t.Fatalf("interaction status=%d body=%s", interaction.StatusCode, body)
	}
	var csrfCookie *http.Cookie
	for _, cookie := range interaction.Cookies() {
		if cookie.Name == "tinyidp_csrf" {
			csrfCookie = cookie
		}
	}
	if csrfCookie == nil {
		t.Fatal("csrf cookie missing")
	}

	form.Set("csrf_token", string(csrf[1]))
	form.Set("login", "alice")
	form.Set("password", "correct horse battery staple")
	form.Set("consent_approved", "true")
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	authorized, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = readBody(t, authorized)
	location, err := url.Parse(authorized.Header.Get("Location"))
	if err != nil || location.Query().Get("code") == "" {
		t.Fatalf("authorization response status=%d location=%q", authorized.StatusCode, authorized.Header.Get("Location"))
	}

	tokenForm := url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ID}, "code": {location.Query().Get("code")}, "redirect_uri": {client.RedirectURIs[0]}, "code_verifier": {verifier}}
	tokenReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/token", strings.NewReader(tokenForm.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResponse, err := httpClient.Do(tokenReq)
	if err != nil {
		t.Fatal(err)
	}
	tokenBody := readBody(t, tokenResponse)
	var tokens map[string]any
	if err := json.Unmarshal(tokenBody, &tokens); err != nil {
		t.Fatal(err)
	}
	if tokenResponse.StatusCode != http.StatusOK || tokens["access_token"] == "" || tokens["id_token"] == "" {
		t.Fatalf("token status=%d body=%s", tokenResponse.StatusCode, tokenBody)
	}
}

func generateSigningKey(t *testing.T) idpstore.SigningKey {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	return idpstore.SigningKey{ID: "external-key-1", Algorithm: "RS256", PrivateKeyPEM: pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}), CreatedAt: now, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(24 * time.Hour), Active: true}
}

func readBody(t *testing.T, response *http.Response) []byte {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
