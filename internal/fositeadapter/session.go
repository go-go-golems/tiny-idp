package fositeadapter

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

const sessionCookieName = "tinyidp_session"

func (p *Provider) createBrowserSession(w http.ResponseWriter, r *http.Request, u idpstore.User, authTime time.Time) error {
	handle, err := randomB64(32)
	if err != nil {
		return err
	}
	hash := idpstore.HashSecret(p.csrfKey, handle)
	now := time.Now().UTC()
	if err := p.store.CreateSession(r.Context(), idpstore.Session{IDHash: hash, UserID: u.ID, AuthTime: authTime, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(p.sessionTTL)}); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: handle, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: int(p.sessionTTL.Seconds())})
	return nil
}

func (p *Provider) readBrowserSession(r *http.Request) (idpstore.User, idpstore.Session, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return idpstore.User{}, idpstore.Session{}, false
	}
	sess, err := p.store.GetSession(r.Context(), idpstore.HashSecret(p.csrfKey, c.Value))
	if err != nil || sess.RevokedAt != nil || time.Now().UTC().After(sess.ExpiresAt) {
		return idpstore.User{}, idpstore.Session{}, false
	}
	u, err := p.store.GetUser(r.Context(), sess.UserID)
	if err != nil || u.Disabled {
		return idpstore.User{}, idpstore.Session{}, false
	}
	return u, sess, true
}

func promptHas(prompt, want string) bool {
	for _, p := range strings.Fields(prompt) {
		if p == want {
			return true
		}
	}
	return false
}

func sessionSatisfiesMaxAge(authTime time.Time, maxAgeValue string) bool {
	if maxAgeValue == "" {
		return true
	}
	maxAge, err := strconv.ParseInt(maxAgeValue, 10, 64)
	if err != nil || maxAge < 0 {
		return true
	}
	return !authTime.Add(time.Duration(maxAge) * time.Second).Before(time.Now().UTC())
}
