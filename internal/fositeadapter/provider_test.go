package fositeadapter_test

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestUnsupportedRequestObjectRedirectsWithStableError(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "https://issuer.example.test", Store: st, SecretKey: []byte("request-object-secret-32-bytes!!")})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	claims := map[string]string{"response_type": "code", "client_id": "spa", "redirect_uri": "http://localhost/callback", "scope": "openid", "state": "state-request-object", "nonce": "nonce-request-object"}
	payload, _ := json.Marshal(claims)
	requestObject := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
	q := url.Values{"client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "response_type": {"code"}, "scope": {"openid"}, "request": {requestObject}}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status=%d, want redirect", resp.StatusCode)
	}
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if loc.Query().Get("error") != "request_not_supported" || loc.Query().Get("state") != "state-request-object" {
		t.Fatalf("unexpected redirect: %s", loc.String())
	}
}

func TestStrictAuthorizationCodeFlow(t *testing.T) {
	ctx := context.Background()
	secretKey := []byte("test-secret-key-32-bytes-minimum!!")
	st := memory.New()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}, AllowedAudiences: []string{"https://inbox.example.test/api"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	resourceSecret := "resource-secret"
	resourceHash, err := bcrypt.GenerateFromPassword([]byte(resourceSecret), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutClient(ctx, idpstore.Client{ID: "inbox-api", SecretHash: resourceHash, AllowedAudiences: []string{"https://inbox.example.test/api"}, CanIntrospect: true, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	otherHash, err := bcrypt.GenerateFromPassword([]byte("other-secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutClient(ctx, idpstore.Client{ID: "other-api", SecretHash: otherHash, AllowedAudiences: []string{"https://other.example.test/api"}, CanIntrospect: true, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Email: "alice@example.test", EmailVerified: true, Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}

	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey, CookieSameSite: http.SameSiteStrictMode})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	challenge := s256(verifier)
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid profile email offline_access"},
		"audience":              {"https://inbox.example.test/api"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"login":                 {"alice"},
	}
	csrfToken, csrfCookie := fetchCSRF(t, ts.URL, form)
	if csrfCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("CSRF SameSite = %v, want Strict", csrfCookie.SameSite)
	}
	form.Set("csrf_token", csrfToken)
	noRedirect := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	reqAuth, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	reqAuth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuth.AddCookie(csrfCookie)
	resp, err := noRedirect.Do(reqAuth)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("authorize status = %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	cb, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	code := cb.Query().Get("code")
	if code == "" || cb.Query().Get("state") != "state-1234567890" {
		t.Fatalf("bad callback location: %s", loc)
	}

	tokResp, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"spa"},
		"code":          {code},
		"redirect_uri":  {"http://localhost/callback"},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer tokResp.Body.Close()
	if tokResp.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d", tokResp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(tokResp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["id_token"] == "" || body["access_token"] == "" || body["refresh_token"] == "" {
		t.Fatalf("missing token fields: %#v", body)
	}
	verifyIDTokenAgainstJWKS(t, ts.URL, body["id_token"].(string), "http://127.0.0.1:5556", "spa", "nonce-1234567890")

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+body["access_token"].(string))
	uiResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer uiResp.Body.Close()
	if uiResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(uiResp.Body)
		t.Fatalf("userinfo status = %d body=%s", uiResp.StatusCode, b)
	}
	var claims map[string]any
	if err := json.NewDecoder(uiResp.Body).Decode(&claims); err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "user-alice" || claims["email"] != "alice@example.test" {
		t.Fatalf("bad userinfo: %#v", claims)
	}
	if uiResp.Header.Get("Cache-Control") != "no-store" || uiResp.Header.Get("Pragma") != "no-cache" {
		t.Fatalf("userinfo cache headers = %q, %q", uiResp.Header.Get("Cache-Control"), uiResp.Header.Get("Pragma"))
	}

	introspectionForm := url.Values{"token": {body["access_token"].(string)}, "token_type_hint": {"access_token"}}
	introspectionRequest, _ := http.NewRequest(http.MethodPost, ts.URL+"/introspect", strings.NewReader(introspectionForm.Encode()))
	introspectionRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	introspectionRequest.SetBasicAuth("inbox-api", resourceSecret)
	introspectionResponse, err := http.DefaultClient.Do(introspectionRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer introspectionResponse.Body.Close()
	if introspectionResponse.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(introspectionResponse.Body)
		t.Fatalf("introspection status=%d body=%s", introspectionResponse.StatusCode, b)
	}
	var introspection map[string]any
	if err := json.NewDecoder(introspectionResponse.Body).Decode(&introspection); err != nil {
		t.Fatal(err)
	}
	if introspection["active"] != true || introspection["sub"] != "user-alice" || introspection["iss"] != "http://127.0.0.1:5556" || introspection["client_id"] != "spa" || !claimHasAudience(introspection["aud"], "https://inbox.example.test/api") {
		t.Fatalf("unexpected introspection response: %#v", introspection)
	}

	wrongSecretRequest, _ := http.NewRequest(http.MethodPost, ts.URL+"/introspect", strings.NewReader(introspectionForm.Encode()))
	wrongSecretRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wrongSecretRequest.SetBasicAuth("inbox-api", "wrong-resource-secret")
	wrongSecretResponse, err := http.DefaultClient.Do(wrongSecretRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer wrongSecretResponse.Body.Close()
	if wrongSecretResponse.StatusCode != http.StatusUnauthorized || wrongSecretResponse.Header.Get("WWW-Authenticate") == "" {
		responseBody, _ := io.ReadAll(wrongSecretResponse.Body)
		t.Fatalf("wrong-secret introspection status=%d headers=%q body=%q", wrongSecretResponse.StatusCode, wrongSecretResponse.Header.Get("WWW-Authenticate"), responseBody)
	}

	wrongAudienceRequest, _ := http.NewRequest(http.MethodPost, ts.URL+"/introspect", strings.NewReader(introspectionForm.Encode()))
	wrongAudienceRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wrongAudienceRequest.SetBasicAuth("other-api", "other-secret")
	wrongAudienceResponse, err := http.DefaultClient.Do(wrongAudienceRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer wrongAudienceResponse.Body.Close()
	var wrongAudience map[string]any
	if err := json.NewDecoder(wrongAudienceResponse.Body).Decode(&wrongAudience); err != nil {
		t.Fatal(err)
	}
	if wrongAudienceResponse.StatusCode != http.StatusOK || wrongAudience["active"] != false || len(wrongAudience) != 1 {
		t.Fatalf("wrong-audience introspection = status=%d body=%#v", wrongAudienceResponse.StatusCode, wrongAudience)
	}

	duplicateAuthorization, _ := http.NewRequest(http.MethodPost, ts.URL+"/introspect", strings.NewReader(introspectionForm.Encode()))
	duplicateAuthorization.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	duplicateAuthorization.Header.Add("Authorization", "Basic aW5ib3gtYXBpOmlubmJveC1hcGktc2VjcmV0")
	duplicateAuthorization.Header.Add("Authorization", "Basic b3RoZXItYXBpOm90aGVyLXNlY3JldA==")
	duplicateAuthorizationResponse, err := http.DefaultClient.Do(duplicateAuthorization)
	if err != nil {
		t.Fatal(err)
	}
	defer duplicateAuthorizationResponse.Body.Close()
	if duplicateAuthorizationResponse.StatusCode != http.StatusUnauthorized || duplicateAuthorizationResponse.Header.Get("WWW-Authenticate") == "" {
		body, _ := io.ReadAll(duplicateAuthorizationResponse.Body)
		t.Fatalf("duplicate authorization status=%d headers=%q body=%q", duplicateAuthorizationResponse.StatusCode, duplicateAuthorizationResponse.Header.Get("WWW-Authenticate"), body)
	}
	postUserInfo, _ := http.NewRequest(http.MethodPost, ts.URL+"/userinfo", nil)
	postUserInfo.Header.Set("Authorization", "Bearer "+body["access_token"].(string))
	postUserInfoResponse, err := http.DefaultClient.Do(postUserInfo)
	if err != nil {
		t.Fatal(err)
	}
	_ = postUserInfoResponse.Body.Close()
	if postUserInfoResponse.StatusCode != http.StatusOK {
		t.Fatalf("POST userinfo status=%d, want 200", postUserInfoResponse.StatusCode)
	}

	for _, tc := range []struct {
		name   string
		method string
		target string
		body   string
		status int
		auth   bool
		dup    bool
	}{
		{name: "query bearer rejected", method: http.MethodGet, target: ts.URL + "/userinfo?access_token=" + url.QueryEscape(body["access_token"].(string)), status: http.StatusBadRequest},
		{name: "form bearer rejected", method: http.MethodPost, target: ts.URL + "/userinfo", body: "access_token=" + url.QueryEscape(body["access_token"].(string)), status: http.StatusBadRequest},
		{name: "mixed query and header rejected", method: http.MethodGet, target: ts.URL + "/userinfo?access_token=" + url.QueryEscape(body["access_token"].(string)), status: http.StatusBadRequest, auth: true},
		{name: "mixed form and header rejected", method: http.MethodPost, target: ts.URL + "/userinfo", body: "access_token=" + url.QueryEscape(body["access_token"].(string)), status: http.StatusBadRequest, auth: true},
		{name: "duplicate authorization headers rejected", method: http.MethodGet, target: ts.URL + "/userinfo", status: http.StatusBadRequest, dup: true},
		{name: "unsupported method", method: http.MethodPut, target: ts.URL + "/userinfo", status: http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if tc.auth {
				req.Header.Set("Authorization", "Bearer "+body["access_token"].(string))
			}
			if tc.dup {
				req.Header.Add("Authorization", "Bearer "+body["access_token"].(string))
				req.Header.Add("Authorization", "Bearer "+body["access_token"].(string))
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Fatalf("status=%d, want %d", resp.StatusCode, tc.status)
			}
			if tc.status == http.StatusUnauthorized && !strings.HasPrefix(resp.Header.Get("WWW-Authenticate"), "Bearer ") {
				t.Fatalf("missing bearer challenge: %q", resp.Header.Get("WWW-Authenticate"))
			}
			if resp.Header.Get("Cache-Control") != "no-store" {
				t.Fatalf("Cache-Control=%q, want no-store", resp.Header.Get("Cache-Control"))
			}
		})
	}

	refreshResp, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"spa"},
		"refresh_token": {body["refresh_token"].(string)},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer refreshResp.Body.Close()
	if refreshResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(refreshResp.Body)
		t.Fatalf("refresh status = %d body=%s", refreshResp.StatusCode, b)
	}
	var refreshed map[string]any
	if err := json.NewDecoder(refreshResp.Body).Decode(&refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed["access_token"] == "" || refreshed["refresh_token"] == "" {
		t.Fatalf("missing refreshed token fields: %#v", refreshed)
	}
	refreshedIntrospection, _ := http.NewRequest(http.MethodPost, ts.URL+"/introspect", strings.NewReader(url.Values{"token": {refreshed["access_token"].(string)}}.Encode()))
	refreshedIntrospection.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	refreshedIntrospection.SetBasicAuth("inbox-api", resourceSecret)
	refreshedIntrospectionResponse, err := http.DefaultClient.Do(refreshedIntrospection)
	if err != nil {
		t.Fatal(err)
	}
	defer refreshedIntrospectionResponse.Body.Close()
	var refreshedMetadata map[string]any
	if err := json.NewDecoder(refreshedIntrospectionResponse.Body).Decode(&refreshedMetadata); err != nil {
		t.Fatal(err)
	}
	if refreshedIntrospectionResponse.StatusCode != http.StatusOK || refreshedMetadata["active"] != true || !claimHasAudience(refreshedMetadata["aud"], "https://inbox.example.test/api") {
		t.Fatalf("refreshed introspection status=%d body=%#v", refreshedIntrospectionResponse.StatusCode, refreshedMetadata)
	}
}

func TestProductionProviderRejectsMissingSecretKey(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	if _, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "https://issuer.example.test", Store: st, Mode: idpstore.ProductionMode}); err == nil {
		t.Fatal("expected production provider to reject missing secret key")
	}
}

func TestStrictProviderHasNoDebugRoute(t *testing.T) {
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(context.Background(), key)
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/debug")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("/debug status = %d, want 404", resp.StatusCode)
	}
}

func fetchCSRF(t *testing.T, baseURL string, form url.Values) (string, *http.Cookie) {
	return fetchCSRFNamed(t, baseURL, form, "tinyidp_csrf")
}

func fetchCSRFNamed(t *testing.T, baseURL string, form url.Values, cookieName string) (string, *http.Cookie) {
	t.Helper()
	q := cloneValues(form)
	q.Del("login")
	resp, err := http.Get(baseURL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	re := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	m := re.FindStringSubmatch(string(body))
	if len(m) != 2 {
		t.Fatalf("csrf token not found in %s", body)
	}
	interactionRE := regexp.MustCompile(`name="interaction" value="([^"]+)"`)
	interaction := interactionRE.FindStringSubmatch(string(body))
	if len(interaction) != 2 {
		t.Fatalf("interaction handle not found in %s", body)
	}
	form.Set("interaction", interaction[1])
	form.Set("action", "approve")
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			return m[1], c
		}
	}
	t.Fatalf("csrf cookie %q not found", cookieName)
	return "", nil
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vv := range v {
		out[k] = append([]string(nil), vv...)
	}
	return out
}

func verifyIDTokenAgainstJWKS(t *testing.T, baseURL, token, issuer, audience, nonce string) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("id_token has %d parts", len(parts))
	}
	var header map[string]any
	if b, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(b, &header); err != nil {
		t.Fatal(err)
	}
	kid, _ := header["kid"].(string)
	if header["alg"] != "RS256" || kid == "" {
		t.Fatalf("bad token header: %#v", header)
	}
	resp, err := http.Get(baseURL + "/jwks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		t.Fatal(err)
	}
	var pub *rsa.PublicKey
	for _, k := range jwks.Keys {
		if k.Kid != kid {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			t.Fatal(err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			t.Fatal(err)
		}
		pub = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(new(big.Int).SetBytes(eBytes).Int64())}
	}
	if pub == nil {
		t.Fatalf("jwks missing kid %q", kid)
	}
	input := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(input))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		t.Fatalf("id_token signature verification failed: %v", err)
	}
	var claims map[string]any
	if b, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(b, &claims); err != nil {
		t.Fatal(err)
	}
	if claims["iss"] != issuer || claims["nonce"] != nonce || !claimHasAudience(claims["aud"], audience) {
		t.Fatalf("bad id_token claims: %#v", claims)
	}
	if _, ok := claims["exp"].(float64); !ok {
		t.Fatalf("missing numeric exp: %#v", claims)
	}
}

func claimHasAudience(v any, audience string) bool {
	switch x := v.(type) {
	case string:
		return x == audience
	case []any:
		for _, item := range x {
			if s, ok := item.(string); ok && s == audience {
				return true
			}
		}
	}
	return false
}

func s256(v string) string {
	sum := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestStrictLoginRequiresStoredPasswordWhenAuthenticatorConfigured(t *testing.T) {
	ctx := context.Background()
	secretKey := []byte("password-auth-secret-32-bytes!!!!")
	st := memory.New()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	svc, err := idpaccounts.NewService(st, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, idpaccounts.CreateRequest{ID: "u1", Subject: "user-alice", Login: "alice", Password: []byte("alice-password-long"), Email: "alice@example.test"}); err != nil {
		t.Fatal(err)
	}
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "https://issuer.example.test", Store: st, SecretKey: secretKey, Mode: idpstore.ProductionMode, Authenticator: svc, Consent: fositeadapter.AlwaysSkipConsent{}})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {s256(verifier)},
		"code_challenge_method": {"S256"},
		"login":                 {"alice"},
		"password":              {"wrong-password"},
	}
	csrfToken, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrfToken)
	noRedirect := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	reqAuth, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	reqAuth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuth.AddCookie(csrfCookie)
	resp, err := noRedirect.Do(reqAuth)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want 401", resp.StatusCode)
	}
	if !strings.Contains(string(body), `role="alert"`) || !strings.Contains(string(body), `value="alice"`) || regexp.MustCompile(`name="password"[^>]*value=`).Match(body) {
		t.Fatalf("wrong password did not render a safe retry form: %s", body)
	}

	form.Set("password", "alice-password-long")
	csrfToken, csrfCookie = fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrfToken)
	reqAuth, _ = http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	reqAuth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuth.AddCookie(csrfCookie)
	resp, err = noRedirect.Do(reqAuth)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("correct password status = %d", resp.StatusCode)
	}
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if loc.Query().Get("code") == "" {
		t.Fatalf("correct password did not issue code: %s", loc.String())
	}
}
