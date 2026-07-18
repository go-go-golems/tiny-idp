package fositeadapter

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

const defaultCSRFCookieName = "tinyidp_csrf"

func (p *Provider) issueCSRF(w http.ResponseWriter, r *http.Request, interactionHandle string) (string, []byte, error) {
	nonce := ""
	if cookie, err := r.Cookie(p.csrfCookieName); err == nil {
		nonce = cookie.Value
	}
	if nonce == "" {
		var err error
		nonce, err = randomB64(32)
		if err != nil {
			return "", nil, err
		}
	}
	http.SetCookie(w, &http.Cookie{Name: p.csrfCookieName, Value: nonce, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: int(p.interactionTTL.Seconds())})
	return p.csrfMAC(nonce, interactionHandle), idpstore.HashSecret(p.csrfKey, nonce), nil
}

func (p *Provider) validateCSRF(r *http.Request, interactionHandle string) bool {
	cookie, err := r.Cookie(p.csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := r.PostForm.Get("csrf_token")
	if formToken == "" || interactionHandle == "" {
		return false
	}
	want := p.csrfMAC(cookie.Value, interactionHandle)
	return hmac.Equal([]byte(want), []byte(formToken))
}

func (p *Provider) csrfMAC(nonce, interactionHandle string) string {
	mac := hmac.New(sha256.New, p.csrfKey)
	_, _ = mac.Write([]byte("tinyidp-csrf:" + nonce + ":" + interactionHandle))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (p *Provider) cookiePath() string {
	if p.cookiePathValue != "" {
		return p.cookiePathValue
	}
	path := p.issuer.URL.EscapedPath()
	if path == "" {
		return "/"
	}
	return path
}
