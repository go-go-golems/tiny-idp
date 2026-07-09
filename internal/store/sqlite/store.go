package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

var _ idpstore.Store = (*Store)(nil)

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	st := &Store{db: db}
	if err := st.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return st, nil
}

func (s *Store) Close() error     { return s.db.Close() }
func (s *Store) Persistent() bool { return true }

// SQLDB exposes the underlying database to adapter packages that need to store
// protocol-specific state while reusing the same SQLite file and transaction
// durability. Callers must not close the returned handle.
func (s *Store) SQLDB() *sql.DB { return s.db }

func MigrationNames() ([]string, error) {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	names, err := MigrationNames()
	if err != nil {
		return err
	}
	for _, name := range names {
		b, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, string(b)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func hashKey(b []byte) string        { return hex.EncodeToString(b) }
func enc(v any) ([]byte, error)      { return json.Marshal(v) }
func dec[T any](b []byte) (T, error) { var v T; err := json.Unmarshal(b, &v); return v, err }

func (s *Store) PutClient(ctx context.Context, c idpstore.Client) error {
	b, _ := enc(c)
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO clients(id,data) VALUES(?,?)`, c.ID, b)
	return err
}
func (s *Store) GetClient(ctx context.Context, id string) (idpstore.Client, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM clients WHERE id=?`, id).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.Client{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.Client{}, err
	}
	return dec[idpstore.Client](b)
}
func (s *Store) ListClients(ctx context.Context) ([]idpstore.Client, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []idpstore.Client
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		c, err := dec[idpstore.Client](b)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) PutUser(ctx context.Context, login string, u idpstore.User) error {
	b, _ := enc(u)
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO users(id,login,data) VALUES(?,?,?)`, u.ID, login, b)
	return err
}
func (s *Store) GetUser(ctx context.Context, id string) (idpstore.User, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM users WHERE id=?`, id).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.User{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.User{}, err
	}
	return dec[idpstore.User](b)
}
func (s *Store) GetUserByLogin(ctx context.Context, login string) (idpstore.User, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM users WHERE login=?`, login).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.User{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.User{}, err
	}
	return dec[idpstore.User](b)
}

func (s *Store) PutPasswordCredential(ctx context.Context, credential idpstore.PasswordCredential) error {
	if existing, err := s.GetPasswordCredentialByLogin(ctx, credential.Login); err == nil && existing.UserID != credential.UserID {
		return idpstore.ErrDuplicate
	} else if err != nil && err != idpstore.ErrNotFound {
		return err
	}
	b, _ := enc(credential)
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO password_credentials(user_id,login,data) VALUES(?,?,?)`, credential.UserID, credential.Login, b)
	return mapDup(err)
}
func (s *Store) GetPasswordCredentialByLogin(ctx context.Context, login string) (idpstore.PasswordCredential, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM password_credentials WHERE login=?`, login).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.PasswordCredential{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.PasswordCredential{}, err
	}
	return dec[idpstore.PasswordCredential](b)
}
func (s *Store) GetPasswordCredentialByUserID(ctx context.Context, userID string) (idpstore.PasswordCredential, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM password_credentials WHERE user_id=?`, userID).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.PasswordCredential{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.PasswordCredential{}, err
	}
	return dec[idpstore.PasswordCredential](b)
}
func (s *Store) DeletePasswordCredential(ctx context.Context, userID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM password_credentials WHERE user_id=?`, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return idpstore.ErrNotFound
	}
	return nil
}
func (s *Store) GetAccountSecurityState(ctx context.Context, userID string) (idpstore.AccountSecurityState, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM account_security_states WHERE user_id=?`, userID).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.AccountSecurityState{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.AccountSecurityState{}, err
	}
	return dec[idpstore.AccountSecurityState](b)
}
func (s *Store) PutAccountSecurityState(ctx context.Context, state idpstore.AccountSecurityState) error {
	b, _ := enc(state)
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO account_security_states(user_id,data) VALUES(?,?)`, state.UserID, b)
	return err
}
func (s *Store) ResetAccountSecurityState(ctx context.Context, userID string, now time.Time) error {
	state := idpstore.AccountSecurityState{UserID: userID, LastSuccessfulLoginAt: &now}
	return s.PutAccountSecurityState(ctx, state)
}

func (s *Store) CreateGrant(ctx context.Context, g idpstore.Grant) error {
	b, _ := enc(g)
	_, err := s.db.ExecContext(ctx, `INSERT INTO grants(id,data) VALUES(?,?)`, g.ID, b)
	return mapDup(err)
}
func (s *Store) GetGrant(ctx context.Context, id string) (idpstore.Grant, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM grants WHERE id=?`, id).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.Grant{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.Grant{}, err
	}
	return dec[idpstore.Grant](b)
}
func (s *Store) RevokeGrant(ctx context.Context, id string, at time.Time) error {
	g, err := s.GetGrant(ctx, id)
	if err != nil {
		return err
	}
	g.RevokedAt = &at
	b, _ := enc(g)
	_, err = s.db.ExecContext(ctx, `UPDATE grants SET data=? WHERE id=?`, b, id)
	return err
}

func (s *Store) CreateAuthorizationCode(ctx context.Context, c idpstore.AuthorizationCode) error {
	b, _ := enc(c)
	_, err := s.db.ExecContext(ctx, `INSERT INTO authorization_codes(hash,data) VALUES(?,?)`, hashKey(c.CodeHash), b)
	return mapDup(err)
}
func (s *Store) ConsumeAuthorizationCode(ctx context.Context, codeHash []byte, now time.Time) (idpstore.AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(codeHash)
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM authorization_codes WHERE hash=?`, k).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.AuthorizationCode{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.AuthorizationCode{}, err
	}
	c, err := dec[idpstore.AuthorizationCode](b)
	if err != nil {
		return idpstore.AuthorizationCode{}, err
	}
	if c.ConsumedAt != nil {
		return idpstore.AuthorizationCode{}, idpstore.ErrAlreadyConsumed
	}
	if !c.ExpiresAt.IsZero() && now.After(c.ExpiresAt) {
		return idpstore.AuthorizationCode{}, idpstore.ErrExpired
	}
	c.ConsumedAt = &now
	nb, _ := enc(c)
	_, err = s.db.ExecContext(ctx, `UPDATE authorization_codes SET data=? WHERE hash=?`, nb, k)
	return c, err
}

func (s *Store) CreateAccessToken(ctx context.Context, t idpstore.AccessToken) error {
	b, _ := enc(t)
	_, err := s.db.ExecContext(ctx, `INSERT INTO access_tokens(hash,data) VALUES(?,?)`, hashKey(t.TokenHash), b)
	return mapDup(err)
}
func (s *Store) GetAccessToken(ctx context.Context, tokenHash []byte) (idpstore.AccessToken, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM access_tokens WHERE hash=?`, hashKey(tokenHash)).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.AccessToken{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.AccessToken{}, err
	}
	return dec[idpstore.AccessToken](b)
}
func (s *Store) RevokeAccessToken(ctx context.Context, tokenHash []byte, at time.Time) error {
	t, err := s.GetAccessToken(ctx, tokenHash)
	if err != nil {
		return err
	}
	t.RevokedAt = &at
	b, _ := enc(t)
	_, err = s.db.ExecContext(ctx, `UPDATE access_tokens SET data=? WHERE hash=?`, b, hashKey(tokenHash))
	return err
}

func (s *Store) CreateRefreshToken(ctx context.Context, t idpstore.RefreshToken) error {
	b, _ := enc(t)
	_, err := s.db.ExecContext(ctx, `INSERT INTO refresh_tokens(hash,grant_id,data) VALUES(?,?,?)`, hashKey(t.TokenHash), t.GrantID, b)
	return mapDup(err)
}
func (s *Store) GetRefreshToken(ctx context.Context, tokenHash []byte) (idpstore.RefreshToken, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM refresh_tokens WHERE hash=?`, hashKey(tokenHash)).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.RefreshToken{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.RefreshToken{}, err
	}
	return dec[idpstore.RefreshToken](b)
}
func (s *Store) RotateRefreshToken(ctx context.Context, oldHash []byte, next idpstore.RefreshToken, now time.Time) (idpstore.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, err := s.GetRefreshToken(ctx, oldHash)
	if err != nil {
		return idpstore.RefreshToken{}, err
	}
	if old.RevokedAt != nil || len(old.ReplacedByHash) > 0 || old.ReuseDetectedAt != nil {
		detected := now
		old.ReuseDetectedAt = &detected
		_ = s.putRefresh(ctx, old)
		_ = s.revokeFamily(ctx, old.GrantID, now)
		return idpstore.RefreshToken{}, idpstore.ErrRefreshReuseDetected
	}
	if !old.ExpiresAt.IsZero() && now.After(old.ExpiresAt) {
		return idpstore.RefreshToken{}, idpstore.ErrExpired
	}
	next.ParentTokenHash = append([]byte(nil), oldHash...)
	old.ReplacedByHash = append([]byte(nil), next.TokenHash...)
	if err := s.putRefresh(ctx, old); err != nil {
		return idpstore.RefreshToken{}, err
	}
	if err := s.CreateRefreshToken(ctx, next); err != nil {
		return idpstore.RefreshToken{}, err
	}
	return next, nil
}
func (s *Store) putRefresh(ctx context.Context, t idpstore.RefreshToken) error {
	b, _ := enc(t)
	_, err := s.db.ExecContext(ctx, `UPDATE refresh_tokens SET grant_id=?, data=? WHERE hash=?`, t.GrantID, b, hashKey(t.TokenHash))
	return err
}
func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, tokenHash []byte, at time.Time) error {
	t, err := s.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return err
	}
	return s.revokeFamily(ctx, t.GrantID, at)
}
func (s *Store) revokeFamily(ctx context.Context, grantID string, at time.Time) error {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM refresh_tokens WHERE grant_id=?`, grantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var toks []idpstore.RefreshToken
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return err
		}
		t, _ := dec[idpstore.RefreshToken](b)
		toks = append(toks, t)
	}
	for _, t := range toks {
		if t.RevokedAt == nil {
			t.RevokedAt = &at
			if err := s.putRefresh(ctx, t); err != nil {
				return err
			}
		}
	}
	return rows.Err()
}

func consentKey(userID, clientID string, scopes []string) string {
	parts := append([]string{userID, clientID}, idpstore.NormalizeScopes(scopes)...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func (s *Store) PutConsent(ctx context.Context, consent idpstore.Consent) error {
	consent.Scope = idpstore.NormalizeScopes(consent.Scope)
	b, _ := enc(consent)
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO consents(key,user_id,client_id,data) VALUES(?,?,?,?)`, consentKey(consent.UserID, consent.ClientID, consent.Scope), consent.UserID, consent.ClientID, b)
	return err
}
func (s *Store) GetConsent(ctx context.Context, userID, clientID string, scopes []string) (idpstore.Consent, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM consents WHERE key=?`, consentKey(userID, clientID, scopes)).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.Consent{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.Consent{}, err
	}
	return dec[idpstore.Consent](b)
}
func (s *Store) RevokeConsent(ctx context.Context, userID, clientID string, scopes []string, at time.Time) error {
	c, err := s.GetConsent(ctx, userID, clientID, scopes)
	if err != nil {
		return err
	}
	c.RevokedAt = &at
	b, _ := enc(c)
	_, err = s.db.ExecContext(ctx, `UPDATE consents SET data=? WHERE key=?`, b, consentKey(userID, clientID, scopes))
	return err
}

func (s *Store) CreateSession(ctx context.Context, sess idpstore.Session) error {
	b, _ := enc(sess)
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions(hash,data) VALUES(?,?)`, hashKey(sess.IDHash), b)
	return mapDup(err)
}
func (s *Store) GetSession(ctx context.Context, idHash []byte) (idpstore.Session, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM sessions WHERE hash=?`, hashKey(idHash)).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.Session{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.Session{}, err
	}
	return dec[idpstore.Session](b)
}
func (s *Store) RevokeSession(ctx context.Context, idHash []byte, at time.Time) error {
	sess, err := s.GetSession(ctx, idHash)
	if err != nil {
		return err
	}
	sess.RevokedAt = &at
	b, _ := enc(sess)
	_, err = s.db.ExecContext(ctx, `UPDATE sessions SET data=? WHERE hash=?`, b, hashKey(idHash))
	return err
}

func (s *Store) CreateSigningKey(ctx context.Context, k idpstore.SigningKey) error {
	b, _ := enc(k)
	active := 0
	if k.Active {
		active = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO signing_keys(id,active,data) VALUES(?,?,?)`, k.ID, active, b)
	return mapDup(err)
}
func (s *Store) ActiveSigningKey(ctx context.Context) (idpstore.SigningKey, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM signing_keys WHERE active=1 LIMIT 1`).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.SigningKey{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.SigningKey{}, err
	}
	return dec[idpstore.SigningKey](b)
}
func (s *Store) VerificationKeys(ctx context.Context) ([]idpstore.SigningKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM signing_keys ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []idpstore.SigningKey
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		k, err := dec[idpstore.SigningKey](b)
		if err != nil {
			return nil, err
		}
		if k.Active || !k.NotAfter.IsZero() {
			out = append(out, k)
		}
	}
	return out, rows.Err()
}
func (s *Store) ActivateSigningKey(ctx context.Context, kid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT id,data FROM signing_keys`)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	type row struct {
		id  string
		key idpstore.SigningKey
	}
	var all []row
	for rows.Next() {
		var id string
		var b []byte
		if err := rows.Scan(&id, &b); err != nil {
			return err
		}
		k, _ := dec[idpstore.SigningKey](b)
		k.Active = id == kid
		if id == kid {
			found = true
		}
		all = append(all, row{id, k})
	}
	if !found {
		return idpstore.ErrNotFound
	}
	for _, r := range all {
		b, _ := enc(r.key)
		active := 0
		if r.key.Active {
			active = 1
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE signing_keys SET active=?, data=? WHERE id=?`, active, b, r.id); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) RetireSigningKey(ctx context.Context, kid string) error {
	k, err := s.getSigningKey(ctx, kid)
	if err != nil {
		return err
	}
	k.Active = false
	if k.NotAfter.IsZero() {
		k.NotAfter = time.Now()
	}
	b, _ := enc(k)
	_, err = s.db.ExecContext(ctx, `UPDATE signing_keys SET active=0,data=? WHERE id=?`, b, kid)
	return err
}
func (s *Store) getSigningKey(ctx context.Context, kid string) (idpstore.SigningKey, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM signing_keys WHERE id=?`, kid).Scan(&b)
	if err == sql.ErrNoRows {
		return idpstore.SigningKey{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.SigningKey{}, err
	}
	return dec[idpstore.SigningKey](b)
}

func mapDup(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(fmt.Sprint(err), "UNIQUE") {
		return idpstore.ErrDuplicate
	}
	return err
}
