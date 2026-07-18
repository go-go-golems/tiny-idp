package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunDiscoversStartsPollsAndRedactsBearerCredential(t *testing.T) {
	var server *httptest.Server
	polls := 0
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_, _ = io.WriteString(w, `{"issuer":"`+server.URL+`","device_authorization_endpoint":"`+server.URL+`/device_authorization","token_endpoint":"`+server.URL+`/token","grant_types_supported":["urn:ietf:params:oauth:grant-type:device_code"]}`)
		case "/device_authorization":
			if err := r.ParseForm(); err != nil || r.Form.Get("client_id") != "smoke-client" {
				t.Fatalf("unexpected start form: %#v %v", r.Form, err)
			}
			_, _ = io.WriteString(w, `{"device_code":"secret-device-code","user_code":"ABCD-EFGH","verification_uri":"https://verify.example.test/device","expires_in":60,"interval":0}`)
		case "/token":
			if err := r.ParseForm(); err != nil || r.Form.Get("device_code") != "secret-device-code" {
				t.Fatalf("unexpected token form: %#v %v", r.Form, err)
			}
			polls++
			if polls == 1 {
				_, _ = io.WriteString(w, `{"error":"authorization_pending"}`)
				return
			}
			_, _ = io.WriteString(w, `{"access_token":"secret-access-token","token_type":"bearer"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	var output strings.Builder
	if err := run(ctx, server.URL, "smoke-client", "openid", "", &output); err != nil {
		t.Fatal(err)
	}
	if polls != 2 {
		t.Fatalf("polls=%d, want 2", polls)
	}
	text := output.String()
	if !strings.Contains(text, "ABCD-EFGH") || !strings.Contains(text, "redacted") || strings.Contains(text, "secret-device-code") || strings.Contains(text, "secret-access-token") {
		t.Fatalf("unsafe or incomplete CLI output: %q", text)
	}
}

func TestDecodeJSONRejectsOversizedAndTrailingData(t *testing.T) {
	if _, err := decodeJSON[map[string]any](strings.NewReader(`{} {}`)); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	if _, err := decodeJSON[map[string]any](strings.NewReader(`{"x":"` + strings.Repeat("a", maxResponseBytes) + `"}`)); err == nil {
		t.Fatal("oversized JSON accepted")
	}
}

func TestSameOriginRejectsCredentialedOrCrossOriginEndpoints(t *testing.T) {
	for endpoint, want := range map[string]bool{
		"https://idp.example.test/token":      true,
		"https://other.example.test/token":    false,
		"https://user@idp.example.test/token": false,
	} {
		if got := sameOrigin("https://idp.example.test", endpoint); got != want {
			t.Fatalf("sameOrigin(%q)=%v, want %v", endpoint, got, want)
		}
	}
}
