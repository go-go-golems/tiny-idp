package fositeadapter

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
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
	session := idpstore.Session{IDHash: hash, UserID: u.ID, AuthTime: authTime, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(p.sessionTTL)}
	contextHandle, remembered, err := p.persistBrowserSession(r, u, session, now)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: p.sessionCookieName, Value: handle, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: int(p.sessionTTL.Seconds())})
	if contextHandle != "" {
		http.SetCookie(w, &http.Cookie{Name: p.chooser.ContextCookieName, Value: contextHandle, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: int(p.chooser.ContextTTL.Seconds())})
	}
	if remembered {
		p.recordAudit(r.Context(), idp.Event{Time: now, Name: "browser_session.remembered", Subject: u.Sub, Result: "accepted"})
	}
	return nil
}

// persistBrowserSession creates the active session and, when explicitly
// enabled, atomically attaches it to a valid browser context. The returned raw
// context handle is emitted only when a new context was created; session and
// remembered-entry raw handles are never stored.
func (p *Provider) persistBrowserSession(r *http.Request, user idpstore.User, session idpstore.Session, now time.Time) (string, bool, error) {
	if !p.chooser.Enabled || !p.chooser.RememberOnPasswordLogin {
		return "", false, p.store.CreateSession(r.Context(), session)
	}
	label, err := p.chooser.labelFor(user)
	if err != nil {
		return "", false, err
	}
	candidateContextHash := p.browserContextHash(r)
	newContextHandle, err := randomB64(32)
	if err != nil {
		return "", false, err
	}
	newContextHash := idpstore.HashSecret(p.csrfKey, newContextHandle)
	entryHandle, err := randomB64(32)
	if err != nil {
		return "", false, err
	}
	entryHash := idpstore.HashSecret(p.csrfKey, entryHandle)
	createdContext := false
	remembered := false
	err = p.store.Update(r.Context(), func(tx idpstore.TxStore) error {
		contextHash := candidateContextHash
		if len(contextHash) != 0 {
			browserContext, getErr := tx.GetBrowserContext(r.Context(), contextHash)
			if getErr != nil && !errors.Is(getErr, idpstore.ErrNotFound) {
				return getErr
			}
			if getErr != nil || browserContext.RevokedAt != nil || !now.Before(browserContext.ExpiresAt) {
				contextHash = nil
			}
		}
		if len(contextHash) == 0 {
			contextHash = newContextHash
			if err := tx.CreateBrowserContext(r.Context(), idpstore.BrowserContext{IDHash: contextHash, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(p.chooser.ContextTTL)}); err != nil {
				return err
			}
			createdContext = true
		}
		if err := tx.CreateSession(r.Context(), session); err != nil {
			return err
		}
		entries, listErr := tx.ListRememberedBrowserSessions(r.Context(), contextHash, now)
		if listErr != nil {
			return listErr
		}
		// A successful password login refreshes this account's remembered
		// authentication rather than showing the same account more than once.
		otherAccounts := entries[:0]
		for _, entry := range entries {
			if entry.UserID != user.ID {
				otherAccounts = append(otherAccounts, entry)
				continue
			}
			if err := tx.RemoveRememberedBrowserSession(r.Context(), contextHash, entry.IDHash, now); err != nil {
				return err
			}
		}
		entries = otherAccounts
		if len(entries) >= p.chooser.MaxRememberedAccounts {
			// List is ordered newest first. Removing the final entry only removes
			// its context membership; the source IdP session remains independently
			// valid until normal expiry/revocation.
			if err := tx.RemoveRememberedBrowserSession(r.Context(), contextHash, entries[len(entries)-1].IDHash, now); err != nil {
				return err
			}
		}
		if err := tx.CreateRememberedBrowserSession(r.Context(), idpstore.RememberedBrowserSession{IDHash: entryHash, ContextIDHash: contextHash, SessionIDHash: session.IDHash, UserID: user.ID, DisplayLabel: label, CreatedAt: now, LastUsedAt: now}); err != nil {
			return err
		}
		remembered = true
		return nil
	})
	if err != nil {
		return "", false, err
	}
	if !createdContext {
		newContextHandle = ""
	}
	return newContextHandle, remembered, nil
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
