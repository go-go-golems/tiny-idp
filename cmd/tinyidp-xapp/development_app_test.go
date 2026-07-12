package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestDevelopmentApplicationLoadsIdentityAndTrustedRoutes(t *testing.T) {
	ctx := context.Background()
	app, err := NewDevelopmentApplication(ctx, DevelopmentApplicationConfig{
		PublicBaseURL: "http://127.0.0.1:8787",
		StateRoot:     t.TempDir(),
		Login:         "alice",
		Password:      "correct horse battery staple",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.Close(context.Background()); err != nil {
			t.Errorf("close application: %v", err)
		}
	})

	tests := []struct {
		path string
		want int
	}{
		{path: "/", want: http.StatusOK},
		{path: "/static/app.js", want: http.StatusOK},
		{path: "/api/me", want: http.StatusUnauthorized},
		{path: "/api/object", want: http.StatusUnauthorized},
		{path: "/rpc/USER_STATE/injected", want: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			app.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8787"+tt.path, nil))
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.want, recorder.Body.String())
			}
		})
	}

	recorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8787/idp/.well-known/openid-configuration", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("discovery status = %d; body=%s", recorder.Code, recorder.Body.String())
	}
	var metadata struct {
		Issuer string `json:"issuer"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Issuer != "http://127.0.0.1:8787/idp" {
		t.Fatalf("issuer = %q", metadata.Issuer)
	}
}

func TestLoadOrCreateKeyIsStableAndOwnerOnly(t *testing.T) {
	file := filepath.Join(t.TempDir(), "secrets", "binding.key")
	first, err := loadOrCreateKey(file)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadOrCreateKey(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || len(first) != 32 {
		t.Fatal("binding key did not remain stable")
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("binding key permissions = %#o, want 0600", info.Mode().Perm())
	}
}

func TestLoadOrCreateKeyRejectsLooseExistingPermissions(t *testing.T) {
	file := filepath.Join(t.TempDir(), "binding.key")
	if err := os.WriteFile(file, make([]byte, 32), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateKey(file); err == nil {
		t.Fatal("expected loose binding key permissions to be rejected")
	}
}

func TestDevelopmentApplicationLoginToPrivateObjectVerticalSlice(t *testing.T) {
	server := httptest.NewUnstartedServer(nil)
	publicBaseURL := "http://" + server.Listener.Addr().String()
	app, err := NewDevelopmentApplication(context.Background(), DevelopmentApplicationConfig{
		PublicBaseURL: publicBaseURL,
		StateRoot:     t.TempDir(),
		Login:         "alice",
		Password:      "correct horse battery staple",
	})
	if err != nil {
		t.Fatal(err)
	}
	server.Config.Handler = app.Handler()
	server.Start()
	t.Cleanup(func() {
		server.Close()
		if err := app.Close(context.Background()); err != nil {
			t.Errorf("close application: %v", err)
		}
	})
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}

	loginPage, err := client.Get(server.URL + "/auth/login")
	if err != nil {
		t.Fatal(err)
	}
	loginHTML, _ := io.ReadAll(loginPage.Body)
	_ = loginPage.Body.Close()
	if loginPage.StatusCode != http.StatusOK {
		t.Fatalf("login page status = %d; body=%s", loginPage.StatusCode, loginHTML)
	}
	form := hiddenFormValues(string(loginHTML))
	form.Set("login", "alice")
	form.Set("password", "correct horse battery staple")
	form.Set("action", "approve")
	loginResponse, err := client.PostForm(server.URL+"/idp/authorize", form)
	if err != nil {
		t.Fatal(err)
	}
	loginBody, _ := io.ReadAll(loginResponse.Body)
	_ = loginResponse.Body.Close()
	if loginResponse.StatusCode != http.StatusOK || loginResponse.Request.URL.Path != "/" {
		t.Fatalf("login completion status=%d url=%s body=%s oidc_failure=%#v", loginResponse.StatusCode, loginResponse.Request.URL, loginBody, app.oidc.LastFailure())
	}

	sessionResponse, err := client.Get(server.URL + "/auth/session")
	if err != nil {
		t.Fatal(err)
	}
	var session struct {
		UserID    string `json:"userId"`
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.NewDecoder(sessionResponse.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	_ = sessionResponse.Body.Close()
	if sessionResponse.StatusCode != http.StatusOK || session.UserID == "" || session.CSRFToken == "" {
		t.Fatalf("session = %#v status=%d", session, sessionResponse.StatusCode)
	}

	objectResponse, err := client.Get(server.URL + "/api/object")
	if err != nil {
		t.Fatal(err)
	}
	objectBody, _ := io.ReadAll(objectResponse.Body)
	_ = objectResponse.Body.Close()
	if objectResponse.StatusCode != http.StatusOK || string(objectBody) != "{}\n" {
		t.Fatalf("initial object status=%d body=%s", objectResponse.StatusCode, objectBody)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/object", bytes.NewBufferString(`{"message":"private"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", session.CSRFToken)
	writeResponse, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	written, _ := io.ReadAll(writeResponse.Body)
	_ = writeResponse.Body.Close()
	if writeResponse.StatusCode != http.StatusOK {
		t.Fatalf("object write status=%d body=%s", writeResponse.StatusCode, written)
	}

	objectResponse, err = client.Get(server.URL + "/api/object")
	if err != nil {
		t.Fatal(err)
	}
	defer objectResponse.Body.Close()
	var document map[string]string
	if err := json.NewDecoder(objectResponse.Body).Decode(&document); err != nil {
		t.Fatal(err)
	}
	if document["message"] != "private" {
		t.Fatalf("stored document = %#v", document)
	}
}

func hiddenFormValues(document string) url.Values {
	values := url.Values{}
	pattern := regexp.MustCompile(`name="([^"]+)" value="([^"]*)"`)
	for _, match := range pattern.FindAllStringSubmatch(document, -1) {
		values.Set(html.UnescapeString(match[1]), html.UnescapeString(match[2]))
	}
	return values
}
