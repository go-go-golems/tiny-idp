package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/pkg/idpaccounts"
)

func TestDeviceAPITwoUsersScopesAndMalformedCredentials(t *testing.T) {
	server := httptest.NewUnstartedServer(nil)
	base := "http://" + server.Listener.Addr().String()
	app, err := NewDevelopmentApplication(context.Background(), DevelopmentApplicationConfig{PublicBaseURL: base, StateRoot: t.TempDir(), Login: "alice", Password: "correct horse battery staple", SecondLogin: "bob", SecondPassword: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	server.Config.Handler = app.Handler()
	server.Start()
	t.Cleanup(func() { server.Close(); _ = app.Close(context.Background()) })
	alice := issueDeviceAccessToken(t, server.Client(), server.URL, "alice", "correct horse battery staple", apiAudience(server.URL), "openid bbs.read bbs.post.create")
	postDeviceBBS(t, server.Client(), server.URL, alice, "Alice dispatch")
	bob := issueDeviceAccessToken(t, server.Client(), server.URL, "bob", "correct horse battery staple", apiAudience(server.URL), "openid bbs.read bbs.post.create")
	body := postDeviceBBS(t, server.Client(), server.URL, bob, "Bob dispatch")
	if !bytes.Contains(body, []byte(`"author":"dev-alice-subject"`)) || !bytes.Contains(body, []byte(`"author":"dev-bob-subject"`)) {
		t.Fatalf("two-user board=%s", body)
	}
	readOnly := issueDeviceAccessToken(t, server.Client(), server.URL, "bob", "correct horse battery staple", apiAudience(server.URL), "openid bbs.read")
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/device/bbs/posts", strings.NewReader(`{"title":"denied","body":"denied","category":"notes"}`))
	req.Header.Set("Authorization", "Bearer "+readOnly)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("read-only post status=%d", resp.StatusCode)
	}
	readRequest, _ := http.NewRequest(http.MethodGet, server.URL+"/api/device/bbs", nil)
	readRequest.Header.Set("Authorization", "Bearer "+bob)
	readResponse, err := server.Client().Do(readRequest)
	if err != nil {
		t.Fatal(err)
	}
	boardAfterDenied, _ := io.ReadAll(readResponse.Body)
	_ = readResponse.Body.Close()
	if readResponse.StatusCode != http.StatusOK || bytes.Contains(boardAfterDenied, []byte(`\"title\":\"denied\"`)) {
		t.Fatalf("denied request changed board: status=%d body=%s", readResponse.StatusCode, boardAfterDenied)
	}
	malformed, _ := http.NewRequest(http.MethodGet, server.URL+"/api/device/bbs", nil)
	malformed.Header.Add("Authorization", "Bearer one")
	malformed.Header.Add("Authorization", "Bearer two")
	resp, err = server.Client().Do(malformed)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ambiguous bearer status=%d", resp.StatusCode)
	}
	wrongAudience, err := server.Client().PostForm(server.URL+"/idp/device_authorization", url.Values{"client_id": {deviceClientID}, "scope": {"openid bbs.read"}, "audience": {"https://other.example.test/api"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = wrongAudience.Body.Close()
	if wrongAudience.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong audience status=%d", wrongAudience.StatusCode)
	}
	revoked := issueDeviceAccessToken(t, server.Client(), server.URL, "alice", "correct horse battery staple", apiAudience(server.URL), "openid bbs.read")
	accounts, err := idpaccounts.NewService(app.identityStore, idpaccounts.Options{Clock: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		t.Fatal(err)
	}
	if err := accounts.SetPassword(context.Background(), idpaccounts.SetPasswordRequest{Login: "alice", Password: []byte("new correct horse battery staple")}); err != nil {
		t.Fatal(err)
	}
	revokedRequest, _ := http.NewRequest(http.MethodGet, server.URL+"/api/device/bbs", nil)
	revokedRequest.Header.Set("Authorization", "Bearer "+revoked)
	revokedResponse, err := server.Client().Do(revokedRequest)
	if err != nil {
		t.Fatal(err)
	}
	_ = revokedResponse.Body.Close()
	if revokedResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("password-revoked token status=%d", revokedResponse.StatusCode)
	}
}

func TestInitializedTLSDeviceToBBSAPI(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := InitializeState(context.Background(), InitializeStateConfig{StateRoot: root, PublicBaseURL: "https://app.example.test", Login: "alice", Password: []byte("a unique production password phrase 2026")}); err != nil {
		t.Fatal(err)
	}
	app, err := NewInitializedApplication(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(app.Handler())
	t.Cleanup(func() { server.Close(); _ = app.Close(context.Background()) })
	token := issueDeviceAccessToken(t, server.Client(), server.URL, "alice", "a unique production password phrase 2026", "https://app.example.test/api", "openid bbs.read bbs.post.create")
	body := postDeviceBBS(t, server.Client(), server.URL, token, "TLS device dispatch")
	if !bytes.Contains(body, []byte(`"author":""`)) && !bytes.Contains(body, []byte(`"posts"`)) {
		t.Fatalf("TLS BBS response=%s", body)
	}
}

func issueDeviceAccessToken(t *testing.T, client *http.Client, baseURL, login, password, audience, scopes string) string {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	browser := &http.Client{Jar: jar, Transport: client.Transport}
	started, err := browser.PostForm(baseURL+"/idp/device_authorization", url.Values{"client_id": {deviceClientID}, "scope": {scopes}, "audience": {audience}})
	if err != nil {
		t.Fatal(err)
	}
	var grant struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
	}
	if err := json.NewDecoder(started.Body).Decode(&grant); err != nil {
		t.Fatal(err)
	}
	_ = started.Body.Close()
	if started.StatusCode != http.StatusOK {
		t.Fatalf("device start status=%d", started.StatusCode)
	}
	page, err := browser.Get(baseURL + "/idp/device?user_code=" + url.QueryEscape(grant.UserCode))
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(page.Body)
	_ = page.Body.Close()
	if page.StatusCode != http.StatusOK {
		t.Fatalf("device page status=%d body=%s", page.StatusCode, html)
	}
	form := hiddenFormValues(string(html))
	form.Set("login", login)
	form.Set("password", password)
	form.Set("action", "approve")
	approved, err := browser.PostForm(baseURL+"/idp/device", form)
	if err != nil {
		t.Fatal(err)
	}
	_ = approved.Body.Close()
	if approved.StatusCode != http.StatusOK {
		t.Fatalf("device approval status=%d", approved.StatusCode)
	}
	tokenResponse, err := browser.PostForm(baseURL+"/idp/token", url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "client_id": {deviceClientID}, "device_code": {grant.DeviceCode}})
	if err != nil {
		t.Fatal(err)
	}
	defer tokenResponse.Body.Close()
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResponse.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	if tokenResponse.StatusCode != http.StatusOK || token.AccessToken == "" {
		t.Fatalf("device token status=%d", tokenResponse.StatusCode)
	}
	return token.AccessToken
}

func postDeviceBBS(t *testing.T, client *http.Client, baseURL, token, title string) []byte {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/device/bbs/posts", strings.NewReader(`{"title":"`+title+`","body":"phase five","category":"notes"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("post status=%d body=%s", resp.StatusCode, body)
	}
	return body
}
