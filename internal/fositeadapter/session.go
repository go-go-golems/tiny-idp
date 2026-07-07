package fositeadapter

import (
	"net/http"
	"strings"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
)

const sessionCookieName = "tinyidp_session"

func (p *Provider) createBrowserSession(w http.ResponseWriter, r *http.Request, u domain.User, authTime time.Time) error {
	handle := randomB64(32)
	hash := domain.HashSecret(p.csrfKey, handle)
	now := time.Now().UTC()
	if err := p.store.CreateSession(r.Context(), domain.Session{IDHash: hash, UserID: u.ID, AuthTime: authTime, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(p.sessionTTL)}); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: handle, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(p.sessionTTL.Seconds())})
	return nil
}

func (p *Provider) readBrowserSession(r *http.Request) (domain.User, domain.Session, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return domain.User{}, domain.Session{}, false
	}
	sess, err := p.store.GetSession(r.Context(), domain.HashSecret(p.csrfKey, c.Value))
	if err != nil || sess.RevokedAt != nil || time.Now().UTC().After(sess.ExpiresAt) {
		return domain.User{}, domain.Session{}, false
	}
	u, err := p.store.GetUser(r.Context(), sess.UserID)
	if err != nil || u.Disabled {
		return domain.User{}, domain.Session{}, false
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
