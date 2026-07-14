package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSessionEndpointAndLogoutUseIndependentAppSession(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	csrf := bytes.Repeat([]byte{9}, sha256.Size)
	if err := store.createAppSession(ctx, "browser-token", appSession{Subject: "subject", DisplayName: "Alice", CSRFSecret: csrf, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	app := newMessageApp(store, nil, nil, false)
	app.now = func() time.Time { return now }
	request := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	request.AddCookie(&http.Cookie{Name: appCookieName, Value: "browser-token"})
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"authenticated":true`) || !strings.Contains(response.Body.String(), base64.RawURLEncoding.EncodeToString(csrf)) {
		t.Fatalf("session response = %d %s", response.Code, response.Body.String())
	}
	logout := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logout.AddCookie(&http.Cookie{Name: appCookieName, Value: "browser-token"})
	logout.Header.Set("X-CSRF-Token", base64.RawURLEncoding.EncodeToString(csrf))
	logoutResponse := httptest.NewRecorder()
	app.ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d: %s", logoutResponse.Code, logoutResponse.Body.String())
	}
	if _, err := store.getAppSession(ctx, "browser-token", now.Add(time.Second)); err == nil {
		t.Fatal("logout did not revoke application session")
	}
}

func TestLoginRejectsAmbiguousReturnTo(t *testing.T) {
	store, err := openAppStore(context.Background(), filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	app := newMessageApp(store, &oidcClient{config: oauth2.Config{Endpoint: oauth2.Endpoint{AuthURL: "http://issuer/authorize"}}, now: time.Now}, nil, false)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/login?return_to=//attacker.test", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("login status = %d", response.Code)
	}
}
