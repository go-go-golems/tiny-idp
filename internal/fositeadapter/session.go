package fositeadapter

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

const defaultSessionCookieName = "tinyidp_session"

type browserSessionState string

const (
	browserSessionAbsent       browserSessionState = "absent"
	browserSessionActive       browserSessionState = "active"
	browserSessionNotFound     browserSessionState = "not_found"
	browserSessionExpired      browserSessionState = "expired"
	browserSessionRevoked      browserSessionState = "revoked"
	browserSessionUserDisabled browserSessionState = "user_disabled"
	browserSessionUnavailable  browserSessionState = "unavailable"
)

func (p *Provider) createBrowserSession(w http.ResponseWriter, r *http.Request, u idpstore.User, authTime time.Time) error {
	handle, err := randomB64(32)
	if err != nil {
		return err
	}
	hash := idpstore.HashSecret(p.csrfKey, handle)
	now := p.now()
	if err := p.store.CreateSession(r.Context(), idpstore.Session{IDHash: hash, UserID: u.ID, AuthTime: authTime, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(p.sessionTTL)}); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: p.sessionCookieName, Value: handle, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: int(p.sessionTTL.Seconds())})
	return nil
}

func (p *Provider) readBrowserSession(r *http.Request) (idpstore.User, idpstore.Session, browserSessionState, error) {
	c, err := r.Cookie(p.sessionCookieName)
	if err != nil || c.Value == "" {
		return idpstore.User{}, idpstore.Session{}, browserSessionAbsent, nil
	}
	sess, err := p.store.GetSession(r.Context(), idpstore.HashSecret(p.csrfKey, c.Value))
	if errors.Is(err, idpstore.ErrNotFound) {
		return idpstore.User{}, idpstore.Session{}, browserSessionNotFound, nil
	}
	if err != nil {
		return idpstore.User{}, idpstore.Session{}, browserSessionUnavailable, fmt.Errorf("read browser session: %w", err)
	}
	if sess.RevokedAt != nil {
		return idpstore.User{}, idpstore.Session{}, browserSessionRevoked, nil
	}
	if !p.now().Before(sess.ExpiresAt) {
		return idpstore.User{}, idpstore.Session{}, browserSessionExpired, nil
	}
	u, err := p.store.GetUser(r.Context(), sess.UserID)
	if errors.Is(err, idpstore.ErrNotFound) || u.Disabled {
		return idpstore.User{}, idpstore.Session{}, browserSessionUserDisabled, nil
	}
	if err != nil {
		return idpstore.User{}, idpstore.Session{}, browserSessionUnavailable, fmt.Errorf("read browser session user: %w", err)
	}
	return u, sess, browserSessionActive, nil
}

func promptHas(prompt, want string) bool {
	for _, p := range strings.Fields(prompt) {
		if p == want {
			return true
		}
	}
	return false
}

func parseMaxAge(maxAgeValue string) (int64, bool, error) {
	if maxAgeValue == "" {
		return 0, false, nil
	}
	for _, character := range maxAgeValue {
		if character < '0' || character > '9' {
			return 0, true, fmt.Errorf("invalid max_age")
		}
	}
	maxAge, err := strconv.ParseInt(maxAgeValue, 10, 64)
	if err != nil || maxAge < 0 {
		return 0, true, fmt.Errorf("invalid max_age")
	}
	return maxAge, true, nil
}

func sessionSatisfiesMaxAge(authTime, now time.Time, maxAge int64, present bool) bool {
	if !present {
		return true
	}
	if !authTime.Before(now) {
		return true
	}
	elapsed := now.Sub(authTime)
	elapsedSeconds := int64(elapsed / time.Second)
	if elapsedSeconds < maxAge {
		return true
	}
	if elapsedSeconds > maxAge {
		return false
	}
	return elapsed%time.Second == 0
}
