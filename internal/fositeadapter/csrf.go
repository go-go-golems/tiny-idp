package fositeadapter

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

const csrfCookieName = "tinyidp_csrf"

func (p *Provider) issueCSRF(w http.ResponseWriter) (string, error) {
	nonce, err := randomB64(32)
	if err != nil {
		return "", err
	}
	token := nonce + "." + p.csrfMAC(nonce)
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: token, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: 600})
	return token, nil
}

func (p *Provider) validateCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := r.PostForm.Get("csrf_token")
	if formToken == "" || !hmac.Equal([]byte(cookie.Value), []byte(formToken)) {
		return false
	}
	parts := strings.Split(formToken, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	want := p.csrfMAC(parts[0])
	return hmac.Equal([]byte(want), []byte(parts[1]))
}

func (p *Provider) csrfMAC(nonce string) string {
	mac := hmac.New(sha256.New, p.csrfKey)
	_, _ = mac.Write([]byte("tinyidp-csrf:" + nonce))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (p *Provider) clearCSRF(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: "", Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: -1})
}

func (p *Provider) cookiePath() string {
	path := p.issuer.URL.EscapedPath()
	if path == "" {
		return "/"
	}
	return path
}
