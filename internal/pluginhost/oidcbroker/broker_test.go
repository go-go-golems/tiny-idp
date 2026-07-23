package oidcbroker

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestBrokerStartAndCompleteUsesPKCENonceAndUserInfo(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	transactions, err := NewTransactionManager(store.SQLDB(), []byte("0123456789abcdef0123456789abcdef"), rand.Reader, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, &jose.SignerOptions{ExtraHeaders: map[jose.HeaderKey]any{"kid": "test-key"}})
	if err != nil {
		t.Fatal(err)
	}
	const issuer = "https://idp.example.test"
	var expectedNonce, expectedChallenge string
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(writer, map[string]any{
				"issuer": issuer, "authorization_endpoint": issuer + "/authorize",
				"token_endpoint": issuer + "/token", "userinfo_endpoint": issuer + "/userinfo",
				"jwks_uri": issuer + "/jwks", "response_types_supported": []string{"code"},
				"subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/jwks":
			writeJSON(writer, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test-key", Algorithm: string(jose.RS256), Use: "sig"}}})
		case "/token":
			if err := request.ParseForm(); err != nil {
				t.Error(err)
			}
			sum := sha256.Sum256([]byte(request.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(sum[:]) != expectedChallenge ||
				request.Form.Get("code") != "authorization-code" ||
				request.Form.Get("client_id") != "jitsi-client" {
				t.Errorf("token form = %v", request.Form)
			}
			raw, signErr := josejwt.Signed(signer).Claims(josejwt.Claims{
				Issuer: issuer, Subject: "user-123", Audience: josejwt.Audience{"jitsi-client"},
				IssuedAt: josejwt.NewNumericDate(now), Expiry: josejwt.NewNumericDate(now.Add(5 * time.Minute)),
			}).Claims(map[string]any{"nonce": expectedNonce}).Serialize()
			if signErr != nil {
				t.Error(signErr)
			}
			writeJSON(writer, map[string]any{"access_token": "access-token", "id_token": raw, "token_type": "Bearer"})
		case "/userinfo":
			if request.Header.Get("Authorization") != "Bearer access-token" {
				t.Errorf("authorization = %q", request.Header.Get("Authorization"))
			}
			writeJSON(writer, map[string]any{
				"sub": "user-123", "email": "user@example.test", "email_verified": true,
				"name": "Test User", "preferred_username": "test", "groups": []string{"staff"}, "roles": []string{"moderator"},
			})
		default:
			http.NotFound(writer, request)
		}
	})
	broker, err := New(ctx, issuer, handler, transactions)
	if err != nil {
		t.Fatal(err)
	}
	started, err := broker.Start(ctx, pluginapi.StartRequest{
		PluginID: "jitsi", ClientID: "jitsi-client", CallbackPath: "/integrations/jitsi/callback",
		Scopes: []string{"openid", "profile", "email"}, PluginState: []byte(`{"room":"engineering"}`),
		BrowserBinding: "browser-one", Registration: true, SelectAccount: true, TTL: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	authorizationURL, err := url.Parse(started.AuthorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	expectedNonce = authorizationURL.Query().Get("nonce")
	expectedChallenge = authorizationURL.Query().Get("code_challenge")
	if authorizationURL.Path != "/authorize" || authorizationURL.Query().Get("state") != started.State ||
		authorizationURL.Query().Get("code_challenge_method") != "S256" ||
		authorizationURL.Query().Get("tinyidp_signup") != "1" ||
		authorizationURL.Query().Get("prompt") != "select_account" {
		t.Fatalf("authorization URL = %s", authorizationURL)
	}
	completion, err := broker.Complete(ctx, pluginapi.CompleteRequest{
		PluginID: "jitsi", BrowserBinding: "browser-one", State: started.State, Code: "authorization-code",
	})
	if err != nil {
		t.Fatalf("%v: %v", err, errors.Unwrap(err))
	}
	if completion.Identity.Subject != "user-123" || completion.Identity.Email != "user@example.test" ||
		!completion.Identity.EmailVerified || strings.Join(completion.Identity.Roles, ",") != "moderator" ||
		string(completion.PluginState) != `{"room":"engineering"}` {
		t.Fatalf("completion = %#v", completion)
	}
	if _, err := broker.Complete(ctx, pluginapi.CompleteRequest{
		PluginID: "jitsi", BrowserBinding: "browser-one", State: started.State, Code: "authorization-code",
	}); err == nil {
		t.Fatal("broker replay was accepted")
	}
}

func writeJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(value)
}
