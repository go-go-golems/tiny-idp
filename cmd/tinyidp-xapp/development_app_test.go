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
	"strings"
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
		{path: "/api/me", want: http.StatusUnauthorized},
		{path: "/api/object", want: http.StatusUnauthorized},
		{path: "/api/bbs", want: http.StatusUnauthorized},
		{path: "/rpc/USER_STATE/injected", want: http.StatusNotFound},
		{path: "/fetch/BBS/community", want: http.StatusNotFound},
	}

	indexRecorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(indexRecorder, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8787/", nil))
	assetPattern := regexp.MustCompile(`/static/assets/[^"']+\.(?:js|css)`)
	assets := assetPattern.FindAllString(indexRecorder.Body.String(), -1)
	if len(assets) < 2 {
		t.Fatalf("generated index does not reference JS and CSS assets: %s", indexRecorder.Body.String())
	}
	for _, asset := range assets {
		assetRecorder := httptest.NewRecorder()
		app.Handler().ServeHTTP(assetRecorder, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8787"+asset, nil))
		if assetRecorder.Code != http.StatusOK {
			t.Fatalf("asset %s status=%d body=%s", asset, assetRecorder.Code, assetRecorder.Body.String())
		}
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
		Issuer             string `json:"issuer"`
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Issuer != "http://127.0.0.1:8787/idp" {
		t.Fatalf("issuer = %q", metadata.Issuer)
	}
	if metadata.EndSessionEndpoint != "http://127.0.0.1:8787/idp/end-session" {
		t.Fatalf("end-session endpoint = %q", metadata.EndSessionEndpoint)
	}
}

func TestDevelopmentApplicationInteractionDoctor(t *testing.T) {
	app, err := NewDevelopmentApplication(context.Background(), DevelopmentApplicationConfig{
		PublicBaseURL: "http://127.0.0.1:8787",
		StateRoot:     t.TempDir(),
		Login:         "alice",
		Password:      "correct horse battery staple",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close(context.Background()) })
	health, err := app.CheckInteractionUI(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if health.HTMLBytes == 0 || health.CSSBytes == 0 {
		t.Fatalf("interaction health=%#v", health)
	}
}

func TestDevelopmentApplicationReconcilesPersistentIdentityState(t *testing.T) {
	ctx := context.Background()
	config := DevelopmentApplicationConfig{
		PublicBaseURL: "http://127.0.0.1:8787",
		StateRoot:     t.TempDir(),
		Login:         "alice",
		Password:      "correct horse battery staple",
	}
	first, err := NewDevelopmentApplication(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(ctx); err != nil {
		t.Fatal(err)
	}
	second, err := NewDevelopmentApplication(ctx, config)
	if err != nil {
		t.Fatalf("restart with equivalent identity state: %v", err)
	}
	if err := second.Close(ctx); err != nil {
		t.Fatal(err)
	}
	config.Password = "different persisted password phrase"
	if _, err := NewDevelopmentApplication(ctx, config); err == nil || !strings.Contains(err.Error(), "password conflicts with persisted identity state") {
		t.Fatalf("conflicting restart error = %v", err)
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

func TestDevelopmentApplicationLoginToApplicationVerticalSlice(t *testing.T) {
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
	if !bytes.Contains(loginHTML, []byte(`href="/static/tinyidp/login.css"`)) || !bytes.Contains(loginHTML, []byte(`Tiny BBS identity service`)) {
		t.Fatalf("login page did not use the xapp renderer: %s", loginHTML)
	}
	wantCSP := "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'"
	if got := loginPage.Header.Get("Content-Security-Policy"); got != wantCSP {
		t.Fatalf("login CSP=%q want=%q", got, wantCSP)
	}
	stylesheetResponse, err := client.Get(server.URL + "/static/tinyidp/login.css")
	if err != nil {
		t.Fatal(err)
	}
	stylesheet, _ := io.ReadAll(stylesheetResponse.Body)
	_ = stylesheetResponse.Body.Close()
	if stylesheetResponse.StatusCode != http.StatusOK || !strings.HasPrefix(stylesheetResponse.Header.Get("Content-Type"), "text/css") || !bytes.Contains(stylesheet, []byte("--mint")) {
		t.Fatalf("login stylesheet status=%d content-type=%q body=%s", stylesheetResponse.StatusCode, stylesheetResponse.Header.Get("Content-Type"), stylesheet)
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

	missingCSRF, err := http.NewRequest(http.MethodPost, server.URL+"/api/bbs/posts", bytes.NewBufferString(`{"title":"Denied","body":"No token","category":"general"}`))
	if err != nil {
		t.Fatal(err)
	}
	missingCSRF.Header.Set("Content-Type", "application/json")
	missingCSRFResponse, err := client.Do(missingCSRF)
	if err != nil {
		t.Fatal(err)
	}
	missingCSRFBody, _ := io.ReadAll(missingCSRFResponse.Body)
	_ = missingCSRFResponse.Body.Close()
	if missingCSRFResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("missing CSRF status=%d body=%s", missingCSRFResponse.StatusCode, missingCSRFBody)
	}

	invalidPost, err := http.NewRequest(http.MethodPost, server.URL+"/api/bbs/posts", bytes.NewBufferString(`{"title":"Invalid","body":"Rejected","category":"private"}`))
	if err != nil {
		t.Fatal(err)
	}
	invalidPost.Header.Set("Content-Type", "application/json")
	invalidPost.Header.Set("X-CSRF-Token", session.CSRFToken)
	invalidPostResponse, err := client.Do(invalidPost)
	if err != nil {
		t.Fatal(err)
	}
	invalidPostBody, _ := io.ReadAll(invalidPostResponse.Body)
	_ = invalidPostResponse.Body.Close()
	if invalidPostResponse.StatusCode != http.StatusBadRequest || !bytes.Contains(invalidPostBody, []byte(`"invalid_category"`)) {
		t.Fatalf("invalid post status=%d body=%s", invalidPostResponse.StatusCode, invalidPostBody)
	}

	createPost, err := http.NewRequest(http.MethodPost, server.URL+"/api/bbs/posts", bytes.NewBufferString(`{
		"title":"Shared post",
		"body":"Created through the trusted route",
		"category":"projects",
		"actorId":"attacker-selected",
		"actorName":"Mallory",
		"namespace":"ADMIN",
		"objectName":"other"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	createPost.Header.Set("Content-Type", "application/json")
	createPost.Header.Set("X-CSRF-Token", session.CSRFToken)
	createPostResponse, err := client.Do(createPost)
	if err != nil {
		t.Fatal(err)
	}
	createPostBody, _ := io.ReadAll(createPostResponse.Body)
	_ = createPostResponse.Body.Close()
	if createPostResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create post status=%d body=%s", createPostResponse.StatusCode, createPostBody)
	}
	var board testBBSBoard
	if err := json.Unmarshal(createPostBody, &board); err != nil {
		t.Fatal(err)
	}
	if len(board.Posts) != 1 || board.Posts[0].ID != "post_000000000001" || board.Posts[0].Author != "Alice" || !board.Posts[0].CanDelete {
		t.Fatalf("created board = %#v", board)
	}
	if bytes.Contains(createPostBody, []byte("attacker-selected")) || bytes.Contains(createPostBody, []byte("Mallory")) {
		t.Fatalf("route accepted spoofed actor data: %s", createPostBody)
	}

	replyRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/bbs/posts/post_000000000001/replies", bytes.NewBufferString(`{"body":"A reply"}`))
	if err != nil {
		t.Fatal(err)
	}
	replyRequest.Header.Set("Content-Type", "application/json")
	replyRequest.Header.Set("X-CSRF-Token", session.CSRFToken)
	replyResponse, err := client.Do(replyRequest)
	if err != nil {
		t.Fatal(err)
	}
	replyBody, _ := io.ReadAll(replyResponse.Body)
	_ = replyResponse.Body.Close()
	if replyResponse.StatusCode != http.StatusCreated {
		t.Fatalf("reply status=%d body=%s", replyResponse.StatusCode, replyBody)
	}

	deleteRequest, err := http.NewRequest(http.MethodDelete, server.URL+"/api/bbs/posts/post_000000000001", nil)
	if err != nil {
		t.Fatal(err)
	}
	deleteRequest.Header.Set("X-CSRF-Token", session.CSRFToken)
	deleteResponse, err := client.Do(deleteRequest)
	if err != nil {
		t.Fatal(err)
	}
	deleteBody, _ := io.ReadAll(deleteResponse.Body)
	_ = deleteResponse.Body.Close()
	if deleteResponse.StatusCode != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteResponse.StatusCode, deleteBody)
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
