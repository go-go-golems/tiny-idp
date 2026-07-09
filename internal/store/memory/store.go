package memory

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

// Store is a concurrency-safe in-memory implementation of idpstore.Store. It is
// intended for tests, examples, and dev-mode strict engine runs.
type Store struct {
	mu            sync.Mutex
	inTransaction bool

	clients            map[string]idpstore.Client
	usersByID          map[string]idpstore.User
	usersByLogin       map[string]string
	credentialsByUser  map[string]idpstore.PasswordCredential
	credentialsByLogin map[string]string
	accountSecurity    map[string]idpstore.AccountSecurityState
	grants             map[string]idpstore.Grant
	codes              map[string]idpstore.AuthorizationCode
	access             map[string]idpstore.AccessToken
	refresh            map[string]idpstore.RefreshToken
	consents           map[string]idpstore.Consent
	sessions           map[string]idpstore.Session
	keys               map[string]idpstore.SigningKey
}

var _ idpstore.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		clients:            map[string]idpstore.Client{},
		usersByID:          map[string]idpstore.User{},
		usersByLogin:       map[string]string{},
		credentialsByUser:  map[string]idpstore.PasswordCredential{},
		credentialsByLogin: map[string]string{},
		accountSecurity:    map[string]idpstore.AccountSecurityState{},
		grants:             map[string]idpstore.Grant{},
		codes:              map[string]idpstore.AuthorizationCode{},
		access:             map[string]idpstore.AccessToken{},
		refresh:            map[string]idpstore.RefreshToken{},
		consents:           map[string]idpstore.Consent{},
		sessions:           map[string]idpstore.Session{},
		keys:               map[string]idpstore.SigningKey{},
	}
}

func hashKey(b []byte) string { return hex.EncodeToString(b) }

func (s *Store) PutClient(_ context.Context, c idpstore.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
	return nil
}

func (s *Store) GetClient(_ context.Context, id string) (idpstore.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return idpstore.Client{}, idpstore.ErrNotFound
	}
	return c, nil
}

func (s *Store) ListClients(_ context.Context) ([]idpstore.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]idpstore.Client, 0, len(s.clients))
	for _, c := range s.clients {
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) PutUser(_ context.Context, login string, u idpstore.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usersByID[u.ID] = u
	if login != "" {
		s.usersByLogin[login] = u.ID
	}
	return nil
}

func (s *Store) GetUser(_ context.Context, id string) (idpstore.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersByID[id]
	if !ok {
		return idpstore.User{}, idpstore.ErrNotFound
	}
	return u, nil
}

func (s *Store) GetUserByLogin(_ context.Context, login string) (idpstore.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.usersByLogin[login]
	if !ok {
		return idpstore.User{}, idpstore.ErrNotFound
	}
	u, ok := s.usersByID[id]
	if !ok {
		return idpstore.User{}, idpstore.ErrNotFound
	}
	return u, nil
}

func (s *Store) PutPasswordCredential(_ context.Context, credential idpstore.PasswordCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingUserID, ok := s.credentialsByLogin[credential.Login]; ok && existingUserID != credential.UserID {
		return idpstore.ErrDuplicate
	}
	if old, ok := s.credentialsByUser[credential.UserID]; ok && old.Login != credential.Login {
		delete(s.credentialsByLogin, old.Login)
	}
	s.credentialsByUser[credential.UserID] = credential
	s.credentialsByLogin[credential.Login] = credential.UserID
	return nil
}

func (s *Store) GetPasswordCredentialByLogin(_ context.Context, login string) (idpstore.PasswordCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	userID, ok := s.credentialsByLogin[login]
	if !ok {
		return idpstore.PasswordCredential{}, idpstore.ErrNotFound
	}
	credential, ok := s.credentialsByUser[userID]
	if !ok {
		return idpstore.PasswordCredential{}, idpstore.ErrNotFound
	}
	return credential, nil
}

func (s *Store) GetPasswordCredentialByUserID(_ context.Context, userID string) (idpstore.PasswordCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential, ok := s.credentialsByUser[userID]
	if !ok {
		return idpstore.PasswordCredential{}, idpstore.ErrNotFound
	}
	return credential, nil
}

func (s *Store) DeletePasswordCredential(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	credential, ok := s.credentialsByUser[userID]
	if !ok {
		return idpstore.ErrNotFound
	}
	delete(s.credentialsByUser, userID)
	delete(s.credentialsByLogin, credential.Login)
	return nil
}

func (s *Store) GetAccountSecurityState(_ context.Context, userID string) (idpstore.AccountSecurityState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.accountSecurity[userID]
	if !ok {
		return idpstore.AccountSecurityState{}, idpstore.ErrNotFound
	}
	return state, nil
}

func (s *Store) PutAccountSecurityState(_ context.Context, state idpstore.AccountSecurityState) error {
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

func (s *Store) CreateGrant(_ context.Context, grant idpstore.Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.grants[grant.ID]; ok {
		return idpstore.ErrDuplicate
	}
	s.grants[grant.ID] = grant
	return nil
}

func (s *Store) GetGrant(_ context.Context, id string) (idpstore.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[id]
	if !ok {
		return idpstore.Grant{}, idpstore.ErrNotFound
	}
	return g, nil
}

func (s *Store) RevokeGrant(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[id]
	if !ok {
		return idpstore.ErrNotFound
	}
	g.RevokedAt = &at
	s.grants[id] = g
	return nil
}

func (s *Store) CreateAuthorizationCode(_ context.Context, code idpstore.AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(code.CodeHash)
	if _, ok := s.codes[k]; ok {
		return idpstore.ErrDuplicate
	}
	s.codes[k] = code
	return nil
}

func (s *Store) ConsumeAuthorizationCode(_ context.Context, codeHash []byte, now time.Time) (idpstore.AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(codeHash)
	code, ok := s.codes[k]
	if !ok {
		return idpstore.AuthorizationCode{}, idpstore.ErrNotFound
	}
	if code.ConsumedAt != nil {
		return idpstore.AuthorizationCode{}, idpstore.ErrAlreadyConsumed
	}
	if !code.ExpiresAt.IsZero() && now.After(code.ExpiresAt) {
		return idpstore.AuthorizationCode{}, idpstore.ErrExpired
	}
	consumed := now
	code.ConsumedAt = &consumed
	s.codes[k] = code
	return code, nil
}

func (s *Store) CreateAccessToken(_ context.Context, token idpstore.AccessToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(token.TokenHash)
	if _, ok := s.access[k]; ok {
		return idpstore.ErrDuplicate
	}
	s.access[k] = token
	return nil
}

func (s *Store) GetAccessToken(_ context.Context, tokenHash []byte) (idpstore.AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.access[hashKey(tokenHash)]
	if !ok {
		return idpstore.AccessToken{}, idpstore.ErrNotFound
	}
	return t, nil
}

func (s *Store) RevokeAccessToken(_ context.Context, tokenHash []byte, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(tokenHash)
	t, ok := s.access[k]
	if !ok {
		return idpstore.ErrNotFound
	}
	t.RevokedAt = &at
	s.access[k] = t
	return nil
}

func (s *Store) CreateRefreshToken(_ context.Context, token idpstore.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(token.TokenHash)
	if _, ok := s.refresh[k]; ok {
		return idpstore.ErrDuplicate
	}
	s.refresh[k] = token
	return nil
}

func (s *Store) GetRefreshToken(_ context.Context, tokenHash []byte) (idpstore.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.refresh[hashKey(tokenHash)]
	if !ok {
		return idpstore.RefreshToken{}, idpstore.ErrNotFound
	}
	return t, nil
}

func (s *Store) RotateRefreshToken(_ context.Context, oldHash []byte, next idpstore.RefreshToken, now time.Time) (idpstore.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldKey := hashKey(oldHash)
	old, ok := s.refresh[oldKey]
	if !ok {
		return idpstore.RefreshToken{}, idpstore.ErrNotFound
	}
	if old.RevokedAt != nil || len(old.ReplacedByHash) > 0 || old.ReuseDetectedAt != nil {
		detected := now
		old.ReuseDetectedAt = &detected
		s.refresh[oldKey] = old
		s.revokeRefreshFamilyLocked(old.GrantID, now)
		return idpstore.RefreshToken{}, idpstore.ErrRefreshReuseDetected
	}
	if !old.ExpiresAt.IsZero() && now.After(old.ExpiresAt) {
		return idpstore.RefreshToken{}, idpstore.ErrExpired
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
		return idpstore.ErrNotFound
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
	return userID + "\x00" + clientID + "\x00" + strings.Join(idpstore.NormalizeScopes(scopes), " ")
}

func (s *Store) PutConsent(_ context.Context, consent idpstore.Consent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	consent.Scope = idpstore.NormalizeScopes(consent.Scope)
	s.consents[consentKey(consent.UserID, consent.ClientID, consent.Scope)] = consent
	return nil
}

func (s *Store) GetConsent(_ context.Context, userID, clientID string, scopes []string) (idpstore.Consent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.consents[consentKey(userID, clientID, scopes)]
	if !ok {
		return idpstore.Consent{}, idpstore.ErrNotFound
	}
	return c, nil
}

func (s *Store) RevokeConsent(_ context.Context, userID, clientID string, scopes []string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := consentKey(userID, clientID, scopes)
	c, ok := s.consents[k]
	if !ok {
		return idpstore.ErrNotFound
	}
	c.RevokedAt = &at
	s.consents[k] = c
	return nil
}

func (s *Store) CreateSession(_ context.Context, session idpstore.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(session.IDHash)
	if _, ok := s.sessions[k]; ok {
		return idpstore.ErrDuplicate
	}
	s.sessions[k] = session
	return nil
}

func (s *Store) GetSession(_ context.Context, idHash []byte) (idpstore.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[hashKey(idHash)]
	if !ok {
		return idpstore.Session{}, idpstore.ErrNotFound
	}
	return sess, nil
}

func (s *Store) RevokeSession(_ context.Context, idHash []byte, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(idHash)
	sess, ok := s.sessions[k]
	if !ok {
		return idpstore.ErrNotFound
	}
	sess.RevokedAt = &at
	s.sessions[k] = sess
	return nil
}

func (s *Store) CreateSigningKey(_ context.Context, key idpstore.SigningKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[key.ID]; ok {
		return idpstore.ErrDuplicate
	}
	s.keys[key.ID] = key
	return nil
}

func (s *Store) ActiveSigningKey(_ context.Context) (idpstore.SigningKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.keys {
		if k.Active {
			return k, nil
		}
	}
	return idpstore.SigningKey{}, idpstore.ErrNotFound
}

func (s *Store) VerificationKeys(_ context.Context) ([]idpstore.SigningKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]idpstore.SigningKey, 0, len(s.keys))
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
		return idpstore.ErrNotFound
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
		return idpstore.ErrNotFound
	}
	if k.Active {
		return idpstore.ErrLastSigningKey
	}
	k.Active = false
	if k.NotAfter.IsZero() {
		k.NotAfter = time.Now()
	}
	s.keys[kid] = k
	return nil
}

func (s *Store) View(ctx context.Context, fn func(idpstore.ReadStore) error) error {
	if fn == nil {
		return errors.New("view callback is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	snapshot := s.cloneLocked()
	s.mu.Unlock()
	return fn(snapshot)
}

func (s *Store) Update(ctx context.Context, fn func(idpstore.TxStore) error) error {
	if fn == nil {
		return errors.New("update callback is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.inTransaction {
		return idpstore.ErrNestedTransaction
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := s.cloneLocked()
	if err := fn(snapshot); err != nil {
		return err
	}
	s.replaceLocked(snapshot)
	return nil
}

func (s *Store) CreateUserWithCredential(ctx context.Context, login string, user idpstore.User, credential idpstore.PasswordCredential) error {
	return s.Update(ctx, func(tx idpstore.TxStore) error {
		if err := tx.PutUser(ctx, login, user); err != nil {
			return err
		}
		return tx.PutPasswordCredential(ctx, credential)
	})
}

func (s *Store) ReplacePasswordAndSecurityState(ctx context.Context, credential idpstore.PasswordCredential, state idpstore.AccountSecurityState) error {
	return s.Update(ctx, func(tx idpstore.TxStore) error {
		if err := tx.PutPasswordCredential(ctx, credential); err != nil {
			return err
		}
		if err := tx.PutAccountSecurityState(ctx, state); err != nil {
			return err
		}
		scoped, ok := tx.(*Store)
		if !ok {
			return errors.New("unexpected memory transaction implementation")
		}
		scoped.revokeUserSecurityArtifacts(credential.UserID, &credential.PasswordChangedAt)
		return nil
	})
}

func (s *Store) RevokeUserSecurityArtifacts(ctx context.Context, userID string, at time.Time) error {
	return s.Update(ctx, func(tx idpstore.TxStore) error {
		scoped, ok := tx.(*Store)
		if !ok {
			return errors.New("unexpected memory transaction implementation")
		}
		scoped.revokeUserSecurityArtifacts(userID, &at)
		return nil
	})
}

func (s *Store) revokeUserSecurityArtifacts(userID string, at *time.Time) {
	when := time.Now().UTC()
	if at != nil && !at.IsZero() {
		when = at.UTC()
	}
	for key, grant := range s.grants {
		if grant.UserID == userID && grant.RevokedAt == nil {
			grant.RevokedAt = &when
			s.grants[key] = grant
		}
	}
	for key, code := range s.codes {
		if code.UserID == userID && code.ConsumedAt == nil {
			code.ConsumedAt = &when
			s.codes[key] = code
		}
	}
	for key, token := range s.access {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &when
			s.access[key] = token
		}
	}
	for key, token := range s.refresh {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &when
			s.refresh[key] = token
		}
	}
	for key, session := range s.sessions {
		if session.UserID == userID && session.RevokedAt == nil {
			session.RevokedAt = &when
			s.sessions[key] = session
		}
	}
}

func (s *Store) RecordFailedLogin(ctx context.Context, userID string, now time.Time, policy idpstore.LockoutPolicy) (idpstore.AccountSecurityState, error) {
	var state idpstore.AccountSecurityState
	err := s.Update(ctx, func(tx idpstore.TxStore) error {
		loaded, loadErr := tx.GetAccountSecurityState(ctx, userID)
		if loadErr != nil && !errors.Is(loadErr, idpstore.ErrNotFound) {
			return loadErr
		}
		state = loaded
		state.UserID = userID
		if state.FirstFailedLoginAt == nil || (policy.Window > 0 && now.Sub(*state.FirstFailedLoginAt) > policy.Window) {
			state.FailedLoginCount = 0
			first := now
			state.FirstFailedLoginAt = &first
		}
		state.FailedLoginCount++
		last := now
		state.LastFailedLoginAt = &last
		if policy.Threshold > 0 && state.FailedLoginCount >= policy.Threshold {
			lockedUntil := now.Add(policy.Duration)
			state.LockedUntil = &lockedUntil
		}
		return tx.PutAccountSecurityState(ctx, state)
	})
	return state, err
}

func (s *Store) RecordSuccessfulLogin(ctx context.Context, userID string, now time.Time, session *idpstore.Session) error {
	return s.Update(ctx, func(tx idpstore.TxStore) error {
		if err := tx.ResetAccountSecurityState(ctx, userID, now); err != nil {
			return err
		}
		if session != nil {
			return tx.CreateSession(ctx, *session)
		}
		return nil
	})
}

func (s *Store) RotateSigningKey(ctx context.Context, next idpstore.SigningKey, now time.Time) (idpstore.RotationResult, error) {
	var result idpstore.RotationResult
	err := s.Update(ctx, func(tx idpstore.TxStore) error {
		old, oldErr := tx.ActiveSigningKey(ctx)
		if oldErr != nil && !errors.Is(oldErr, idpstore.ErrNotFound) {
			return oldErr
		}
		next.Active = true
		if err := tx.CreateSigningKey(ctx, next); err != nil {
			return err
		}
		if err := tx.ActivateSigningKey(ctx, next.ID); err != nil {
			return err
		}
		if oldErr == nil {
			retired := old
			retired.Active = false
			if retired.NotAfter.IsZero() {
				retired.NotAfter = now
			}
			if err := tx.RetireSigningKey(ctx, old.ID); err != nil {
				return err
			}
			result.Retired = &retired
		}
		result.Active = next
		return nil
	})
	return result, err
}

func (s *Store) cloneLocked() *Store {
	return &Store{
		inTransaction:      true,
		clients:            cloneMap(s.clients),
		usersByID:          cloneMap(s.usersByID),
		usersByLogin:       cloneMap(s.usersByLogin),
		credentialsByUser:  cloneMap(s.credentialsByUser),
		credentialsByLogin: cloneMap(s.credentialsByLogin),
		accountSecurity:    cloneMap(s.accountSecurity),
		grants:             cloneMap(s.grants),
		codes:              cloneMap(s.codes),
		access:             cloneMap(s.access),
		refresh:            cloneMap(s.refresh),
		consents:           cloneMap(s.consents),
		sessions:           cloneMap(s.sessions),
		keys:               cloneMap(s.keys),
	}
}

func (s *Store) replaceLocked(next *Store) {
	s.clients = next.clients
	s.usersByID = next.usersByID
	s.usersByLogin = next.usersByLogin
	s.credentialsByUser = next.credentialsByUser
	s.credentialsByLogin = next.credentialsByLogin
	s.accountSecurity = next.accountSecurity
	s.grants = next.grants
	s.codes = next.codes
	s.access = next.access
	s.refresh = next.refresh
	s.consents = next.consents
	s.sessions = next.sessions
	s.keys = next.keys
}

func cloneMap[K comparable, V any](source map[K]V) map[K]V {
	clone := make(map[K]V, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func EqualHash(a, b []byte) bool { return bytes.Equal(a, b) }

// Persistent reports whether the store survives process restarts. Memory never
// does, so production validation rejects it unless tests explicitly override.
func (s *Store) Persistent() bool { return false }
