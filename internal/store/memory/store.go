package memory

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/storage"
)

// Store is a concurrency-safe in-memory implementation of storage.Store. It is
// intended for tests, examples, and dev-mode strict engine runs.
type Store struct {
	mu sync.Mutex

	clients            map[string]domain.Client
	usersByID          map[string]domain.User
	usersByLogin       map[string]string
	credentialsByUser  map[string]domain.PasswordCredential
	credentialsByLogin map[string]string
	accountSecurity    map[string]domain.AccountSecurityState
	grants             map[string]domain.Grant
	codes              map[string]domain.AuthorizationCode
	access             map[string]domain.AccessToken
	refresh            map[string]domain.RefreshToken
	consents           map[string]domain.Consent
	sessions           map[string]domain.Session
	keys               map[string]domain.SigningKey
}

func New() *Store {
	return &Store{
		clients:            map[string]domain.Client{},
		usersByID:          map[string]domain.User{},
		usersByLogin:       map[string]string{},
		credentialsByUser:  map[string]domain.PasswordCredential{},
		credentialsByLogin: map[string]string{},
		accountSecurity:    map[string]domain.AccountSecurityState{},
		grants:             map[string]domain.Grant{},
		codes:              map[string]domain.AuthorizationCode{},
		access:             map[string]domain.AccessToken{},
		refresh:            map[string]domain.RefreshToken{},
		consents:           map[string]domain.Consent{},
		sessions:           map[string]domain.Session{},
		keys:               map[string]domain.SigningKey{},
	}
}

func hashKey(b []byte) string { return hex.EncodeToString(b) }

func (s *Store) PutClient(_ context.Context, c domain.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
	return nil
}

func (s *Store) GetClient(_ context.Context, id string) (domain.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return domain.Client{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *Store) ListClients(_ context.Context) ([]domain.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Client, 0, len(s.clients))
	for _, c := range s.clients {
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) PutUser(_ context.Context, login string, u domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usersByID[u.ID] = u
	if login != "" {
		s.usersByLogin[login] = u.ID
	}
	return nil
}

func (s *Store) GetUser(_ context.Context, id string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersByID[id]
	if !ok {
		return domain.User{}, storage.ErrNotFound
	}
	return u, nil
}

func (s *Store) GetUserByLogin(_ context.Context, login string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.usersByLogin[login]
	if !ok {
		return domain.User{}, storage.ErrNotFound
	}
	u, ok := s.usersByID[id]
	if !ok {
		return domain.User{}, storage.ErrNotFound
	}
	return u, nil
}

func (s *Store) PutPasswordCredential(_ context.Context, credential domain.PasswordCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingUserID, ok := s.credentialsByLogin[credential.Login]; ok && existingUserID != credential.UserID {
		return storage.ErrDuplicate
	}
	if old, ok := s.credentialsByUser[credential.UserID]; ok && old.Login != credential.Login {
		delete(s.credentialsByLogin, old.Login)
	}
	s.credentialsByUser[credential.UserID] = credential
	s.credentialsByLogin[credential.Login] = credential.UserID
	return nil
}

func (s *Store) GetPasswordCredentialByLogin(_ context.Context, login string) (domain.PasswordCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	userID, ok := s.credentialsByLogin[login]
	if !ok {
		return domain.PasswordCredential{}, storage.ErrNotFound
	}
	credential, ok := s.credentialsByUser[userID]
	if !ok {
		return domain.PasswordCredential{}, storage.ErrNotFound
	}
	return credential, nil
}

func (s *Store) GetPasswordCredentialByUserID(_ context.Context, userID string) (domain.PasswordCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential, ok := s.credentialsByUser[userID]
	if !ok {
		return domain.PasswordCredential{}, storage.ErrNotFound
	}
	return credential, nil
}

func (s *Store) DeletePasswordCredential(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential, ok := s.credentialsByUser[userID]
	if !ok {
		return storage.ErrNotFound
	}
	delete(s.credentialsByUser, userID)
	delete(s.credentialsByLogin, credential.Login)
	return nil
}

func (s *Store) GetAccountSecurityState(_ context.Context, userID string) (domain.AccountSecurityState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.accountSecurity[userID]
	if !ok {
		return domain.AccountSecurityState{}, storage.ErrNotFound
	}
	return state, nil
}

func (s *Store) PutAccountSecurityState(_ context.Context, state domain.AccountSecurityState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountSecurity[state.UserID] = state
	return nil
}

func (s *Store) ResetAccountSecurityState(_ context.Context, userID string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.accountSecurity[userID]
	state.UserID = userID
	state.FailedLoginCount = 0
	state.FirstFailedLoginAt = nil
	state.LastFailedLoginAt = nil
	state.LockedUntil = nil
	state.LastSuccessfulLoginAt = &now
	s.accountSecurity[userID] = state
	return nil
}

func (s *Store) CreateGrant(_ context.Context, grant domain.Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.grants[grant.ID]; ok {
		return storage.ErrDuplicate
	}
	s.grants[grant.ID] = grant
	return nil
}

func (s *Store) GetGrant(_ context.Context, id string) (domain.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[id]
	if !ok {
		return domain.Grant{}, storage.ErrNotFound
	}
	return g, nil
}

func (s *Store) RevokeGrant(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[id]
	if !ok {
		return storage.ErrNotFound
	}
	g.RevokedAt = &at
	s.grants[id] = g
	return nil
}

func (s *Store) CreateAuthorizationCode(_ context.Context, code domain.AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(code.CodeHash)
	if _, ok := s.codes[k]; ok {
		return storage.ErrDuplicate
	}
	s.codes[k] = code
	return nil
}

func (s *Store) ConsumeAuthorizationCode(_ context.Context, codeHash []byte, now time.Time) (domain.AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(codeHash)
	code, ok := s.codes[k]
	if !ok {
		return domain.AuthorizationCode{}, storage.ErrNotFound
	}
	if code.ConsumedAt != nil {
		return domain.AuthorizationCode{}, storage.ErrAlreadyConsumed
	}
	if !code.ExpiresAt.IsZero() && now.After(code.ExpiresAt) {
		return domain.AuthorizationCode{}, storage.ErrExpired
	}
	consumed := now
	code.ConsumedAt = &consumed
	s.codes[k] = code
	return code, nil
}

func (s *Store) CreateAccessToken(_ context.Context, token domain.AccessToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(token.TokenHash)
	if _, ok := s.access[k]; ok {
		return storage.ErrDuplicate
	}
	s.access[k] = token
	return nil
}

func (s *Store) GetAccessToken(_ context.Context, tokenHash []byte) (domain.AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.access[hashKey(tokenHash)]
	if !ok {
		return domain.AccessToken{}, storage.ErrNotFound
	}
	return t, nil
}

func (s *Store) RevokeAccessToken(_ context.Context, tokenHash []byte, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(tokenHash)
	t, ok := s.access[k]
	if !ok {
		return storage.ErrNotFound
	}
	t.RevokedAt = &at
	s.access[k] = t
	return nil
}

func (s *Store) CreateRefreshToken(_ context.Context, token domain.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(token.TokenHash)
	if _, ok := s.refresh[k]; ok {
		return storage.ErrDuplicate
	}
	s.refresh[k] = token
	return nil
}

func (s *Store) GetRefreshToken(_ context.Context, tokenHash []byte) (domain.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.refresh[hashKey(tokenHash)]
	if !ok {
		return domain.RefreshToken{}, storage.ErrNotFound
	}
	return t, nil
}

func (s *Store) RotateRefreshToken(_ context.Context, oldHash []byte, next domain.RefreshToken, now time.Time) (domain.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldKey := hashKey(oldHash)
	old, ok := s.refresh[oldKey]
	if !ok {
		return domain.RefreshToken{}, storage.ErrNotFound
	}
	if old.RevokedAt != nil || len(old.ReplacedByHash) > 0 || old.ReuseDetectedAt != nil {
		detected := now
		old.ReuseDetectedAt = &detected
		s.refresh[oldKey] = old
		s.revokeRefreshFamilyLocked(old.GrantID, now)
		return domain.RefreshToken{}, storage.ErrRefreshReuseDetected
	}
	if !old.ExpiresAt.IsZero() && now.After(old.ExpiresAt) {
		return domain.RefreshToken{}, storage.ErrExpired
	}
	next.ParentTokenHash = append([]byte(nil), oldHash...)
	old.ReplacedByHash = append([]byte(nil), next.TokenHash...)
	s.refresh[oldKey] = old
	s.refresh[hashKey(next.TokenHash)] = next
	return next, nil
}

func (s *Store) RevokeRefreshTokenFamily(_ context.Context, tokenHash []byte, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.refresh[hashKey(tokenHash)]
	if !ok {
		return storage.ErrNotFound
	}
	s.revokeRefreshFamilyLocked(t.GrantID, at)
	return nil
}

func (s *Store) revokeRefreshFamilyLocked(grantID string, at time.Time) {
	for k, t := range s.refresh {
		if t.GrantID == grantID && t.RevokedAt == nil {
			t.RevokedAt = &at
			s.refresh[k] = t
		}
	}
}

func consentKey(userID, clientID string, scopes []string) string {
	return userID + "\x00" + clientID + "\x00" + strings.Join(domain.NormalizeScopes(scopes), " ")
}

func (s *Store) PutConsent(_ context.Context, consent domain.Consent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	consent.Scope = domain.NormalizeScopes(consent.Scope)
	s.consents[consentKey(consent.UserID, consent.ClientID, consent.Scope)] = consent
	return nil
}

func (s *Store) GetConsent(_ context.Context, userID, clientID string, scopes []string) (domain.Consent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.consents[consentKey(userID, clientID, scopes)]
	if !ok {
		return domain.Consent{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *Store) RevokeConsent(_ context.Context, userID, clientID string, scopes []string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := consentKey(userID, clientID, scopes)
	c, ok := s.consents[k]
	if !ok {
		return storage.ErrNotFound
	}
	c.RevokedAt = &at
	s.consents[k] = c
	return nil
}

func (s *Store) CreateSession(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(session.IDHash)
	if _, ok := s.sessions[k]; ok {
		return storage.ErrDuplicate
	}
	s.sessions[k] = session
	return nil
}

func (s *Store) GetSession(_ context.Context, idHash []byte) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[hashKey(idHash)]
	if !ok {
		return domain.Session{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *Store) RevokeSession(_ context.Context, idHash []byte, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(idHash)
	sess, ok := s.sessions[k]
	if !ok {
		return storage.ErrNotFound
	}
	sess.RevokedAt = &at
	s.sessions[k] = sess
	return nil
}

func (s *Store) CreateSigningKey(_ context.Context, key domain.SigningKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[key.ID]; ok {
		return storage.ErrDuplicate
	}
	s.keys[key.ID] = key
	return nil
}

func (s *Store) ActiveSigningKey(_ context.Context) (domain.SigningKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.keys {
		if k.Active {
			return k, nil
		}
	}
	return domain.SigningKey{}, storage.ErrNotFound
}

func (s *Store) VerificationKeys(_ context.Context) ([]domain.SigningKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.SigningKey, 0, len(s.keys))
	for _, k := range s.keys {
		if k.Active || !k.NotAfter.IsZero() {
			out = append(out, k)
		}
	}
	return out, nil
}

func (s *Store) ActivateSigningKey(_ context.Context, kid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[kid]; !ok {
		return storage.ErrNotFound
	}
	for id, k := range s.keys {
		k.Active = id == kid
		s.keys[id] = k
	}
	return nil
}

func (s *Store) RetireSigningKey(_ context.Context, kid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.keys[kid]
	if !ok {
		return storage.ErrNotFound
	}
	k.Active = false
	if k.NotAfter.IsZero() {
		k.NotAfter = time.Now()
	}
	s.keys[kid] = k
	return nil
}

func EqualHash(a, b []byte) bool { return bytes.Equal(a, b) }

// Persistent reports whether the store survives process restarts. Memory never
// does, so production validation rejects it unless tests explicitly override.
func (s *Store) Persistent() bool { return false }
