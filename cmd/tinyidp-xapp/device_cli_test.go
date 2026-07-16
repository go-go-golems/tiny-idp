package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeviceLoginPollsSlowDownThenCachesOnlySuccess(t *testing.T) {
	var pollCount int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": server.URL, "device_authorization_endpoint": server.URL + "/device_authorization", "token_endpoint": server.URL + "/token"})
		case "/device_authorization":
			if err := r.ParseForm(); err != nil || r.Form.Get("audience") != "https://app.example.test/api" || r.Form.Get("scope") != "openid bbs.read" {
				t.Fatalf("start form=%v err=%v", r.Form, err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"device_code": "device-secret", "user_code": "ABCD-EFGH", "verification_uri": server.URL + "/device", "expires_in": 60, "interval": 1})
		case "/token":
			pollCount++
			switch pollCount {
			case 1:
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			case 2:
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "slow_down"})
			default:
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "opaque-token", "expires_in": 60})
			}
		}
	}))
	defer server.Close()
	oldWait := deviceLoginPollWait
	deviceLoginPollWait = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { deviceLoginPollWait = oldWait })
	token, expiry, err := deviceLogin(context.Background(), DeviceLoginSettings{Issuer: server.URL, ClientID: deviceClientID, Audience: "https://app.example.test/api", Scopes: "openid bbs.read"})
	if err != nil || token != "opaque-token" || expiry.Before(time.Now()) || pollCount != 3 {
		t.Fatalf("token=%q expiry=%s polls=%d err=%v", token, expiry, pollCount, err)
	}
}

func TestDeviceTokenCacheAndBBSRequest(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "token.json")
	if err := writeDeviceTokenCache(cache, deviceTokenCache{AccessToken: "opaque", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cache)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cache info=%v err=%v", info, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer opaque" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %q", r.Method, r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	if err := callBBSAPI(context.Background(), BBSSettings{APIBaseURL: server.URL, TokenCache: cache}, http.MethodPost, "/api/device/bbs/posts", map[string]string{"title": "t"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cache, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDeviceTokenCache(cache); err == nil {
		t.Fatal("loose cache mode accepted")
	}
}
