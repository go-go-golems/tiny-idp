package fositeadapter

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

// endSession implements the current-browser portion of OIDC RP-Initiated
// Logout. The relying party supplies its client ID and, optionally, an exact
// registered post-logout redirect URI. The browser session named by the
// provider cookie is revoked server-side before either rendering success or
// redirecting.
func (p *Provider) endSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")

	clientID := r.URL.Query().Get("client_id")
	postLogoutRedirectURI := r.URL.Query().Get("post_logout_redirect_uri")
	state := r.URL.Query().Get("state")
	if postLogoutRedirectURI != "" {
		if !p.clientAllowsPostLogoutRedirect(r, clientID, postLogoutRedirectURI) {
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "logout.request.rejected", ClientID: clientID, Result: "rejected", Reason: "invalid_post_logout_redirect_uri"})
			http.Error(w, "invalid post_logout_redirect_uri", http.StatusBadRequest)
			return
		}
	}

	sessionHash := p.browserSessionHash(r)
	if len(sessionHash) != 0 {
		if err := p.store.RevokeSession(r.Context(), sessionHash, p.now()); err != nil && !errors.Is(err, idpstore.ErrNotFound) {
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "logout.request.rejected", ClientID: clientID, Result: "rejected", Reason: "session_store_unavailable"})
			http.Error(w, "revoke browser session", http.StatusInternalServerError)
			return
		}
	}
	p.clearBrowserCookies(w)
	p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "logout.success", ClientID: clientID, Result: "accepted"})

	if postLogoutRedirectURI != "" {
		target := postLogoutRedirectURI
		if state != "" {
			target = appendLogoutState(target, state)
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Signed out</title></head><body><h1>Signed out</h1><p>Your tiny-idp browser session has ended.</p></body></html>"))
}

func (p *Provider) clientAllowsPostLogoutRedirect(r *http.Request, clientID, redirectURI string) bool {
	if clientID == "" || redirectURI == "" {
		return false
	}
	client, err := p.store.GetClient(r.Context(), clientID)
	if err != nil || client.Disabled {
		return false
	}
	for _, allowed := range client.PostLogoutRedirectURIs {
		if redirectURI == allowed {
			return true
		}
	}
	return false
}

func (p *Provider) clearBrowserCookies(w http.ResponseWriter) {
	for _, name := range []string{p.sessionCookieName, p.csrfCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     p.cookiePath(),
			HttpOnly: true,
			Secure:   p.cookieSecure,
			SameSite: p.cookieSameSite,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}
}

func appendLogoutState(rawURL, state string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
