package fositeadapter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type failingSessionStore struct {
	idpstore.Store
	err error
}

var _ idpstore.Store = (*failingSessionStore)(nil)

func (s *failingSessionStore) GetSession(context.Context, []byte) (idpstore.Session, error) {
	return idpstore.Session{}, s.err
}

func TestParseMaxAgeAndBoundarySemantics(t *testing.T) {
	for _, raw := range []string{"-1", "+1", "invalid", "9223372036854775808"} {
		if _, _, err := parseMaxAge(raw); err == nil {
			t.Fatalf("parseMaxAge(%q) succeeded", raw)
		}
	}
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	if sessionSatisfiesMaxAge(now.Add(-time.Nanosecond), now, 0, true) {
		t.Fatal("max_age=0 accepted an earlier authentication")
	}
	if !sessionSatisfiesMaxAge(now.Add(-30*time.Second), now, 30, true) {
		t.Fatal("exact max_age boundary was rejected")
	}
	if sessionSatisfiesMaxAge(now.Add(-30*time.Second-time.Nanosecond), now, 30, true) {
		t.Fatal("authentication beyond max_age boundary was accepted")
	}
}

func FuzzParseMaxAgeAcceptsOnlyBoundedDecimal(f *testing.F) {
	for _, seed := range []string{"", "0", "1", "30", "-1", "+1", " 1", "1e3", "9223372036854775807", "9223372036854775808"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		value, present, err := parseMaxAge(raw)
		if raw == "" {
			if err != nil || present || value != 0 {
				t.Fatalf("empty max_age result=(%d,%v,%v)", value, present, err)
			}
			return
		}
		if err != nil {
			return
		}
		if !present || value < 0 {
			t.Fatalf("accepted max_age=%q result=(%d,%v)", raw, value, present)
		}
		for _, character := range raw {
			if character < '0' || character > '9' {
				t.Fatalf("accepted non-decimal max_age=%q", raw)
			}
		}
	})
}

func TestBrowserSessionStorageFailureDoesNotRenderLogin(t *testing.T) {
	ctx := context.Background()
	base := memory.New()
	if err := base.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-session-failure", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := base.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(ctx, Options{
		Issuer:    "http://127.0.0.1:5556",
		Store:     &failingSessionStore{Store: base, err: errors.New("injected session storage failure")},
		SecretKey: []byte("session-failure-secret-key-32-bytes"),
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid"},
		"state":                 {"session-storage-failure-state"},
		"nonce":                 {"session-storage-failure-nonce"},
		"code_challenge":        {strings.Repeat("a", 43)},
		"code_challenge_method": {"S256"},
	}
	request, _ := http.NewRequest(http.MethodGet, server.URL+"/authorize?"+query.Encode(), nil)
	request.AddCookie(&http.Cookie{Name: defaultSessionCookieName, Value: "existing-browser-session"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", response.StatusCode, body)
	}
	if strings.Contains(string(body), `name="password"`) {
		t.Fatalf("storage failure rendered credentials: %s", body)
	}
}

func TestSQLAuthorizeWritesRequireLifecycleTransaction(t *testing.T) {
	store := &sqlFositeStore{}
	err := store.authorizeExec(context.Background(), "authorize_code", "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "requires an active lifecycle transaction") {
		t.Fatalf("authorizeExec error=%v", err)
	}
}
