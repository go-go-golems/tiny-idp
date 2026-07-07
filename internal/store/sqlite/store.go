package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/storage"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

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

func (s *Store) Migrate(ctx context.Context) error {
	b, err := migrations.ReadFile("migrations/001_schema.sql")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, string(b))
	return err
}

func hashKey(b []byte) string        { return hex.EncodeToString(b) }
func enc(v any) ([]byte, error)      { return json.Marshal(v) }
func dec[T any](b []byte) (T, error) { var v T; err := json.Unmarshal(b, &v); return v, err }

func (s *Store) PutClient(ctx context.Context, c domain.Client) error {
	b, _ := enc(c)
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO clients(id,data) VALUES(?,?)`, c.ID, b)
	return err
}
func (s *Store) GetClient(ctx context.Context, id string) (domain.Client, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM clients WHERE id=?`, id).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.Client{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.Client{}, err
	}
	return dec[domain.Client](b)
}
func (s *Store) ListClients(ctx context.Context) ([]domain.Client, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Client
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		c, err := dec[domain.Client](b)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) PutUser(ctx context.Context, login string, u domain.User) error {
	b, _ := enc(u)
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO users(id,login,data) VALUES(?,?,?)`, u.ID, login, b)
	return err
}
func (s *Store) GetUser(ctx context.Context, id string) (domain.User, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM users WHERE id=?`, id).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.User{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return dec[domain.User](b)
}
func (s *Store) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM users WHERE login=?`, login).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.User{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return dec[domain.User](b)
}

func (s *Store) CreateGrant(ctx context.Context, g domain.Grant) error {
	b, _ := enc(g)
	_, err := s.db.ExecContext(ctx, `INSERT INTO grants(id,data) VALUES(?,?)`, g.ID, b)
	return mapDup(err)
}
func (s *Store) GetGrant(ctx context.Context, id string) (domain.Grant, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM grants WHERE id=?`, id).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.Grant{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.Grant{}, err
	}
	return dec[domain.Grant](b)
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

func (s *Store) CreateAuthorizationCode(ctx context.Context, c domain.AuthorizationCode) error {
	b, _ := enc(c)
	_, err := s.db.ExecContext(ctx, `INSERT INTO authorization_codes(hash,data) VALUES(?,?)`, hashKey(c.CodeHash), b)
	return mapDup(err)
}
func (s *Store) ConsumeAuthorizationCode(ctx context.Context, codeHash []byte, now time.Time) (domain.AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := hashKey(codeHash)
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM authorization_codes WHERE hash=?`, k).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.AuthorizationCode{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.AuthorizationCode{}, err
	}
	c, err := dec[domain.AuthorizationCode](b)
	if err != nil {
		return domain.AuthorizationCode{}, err
	}
	if c.ConsumedAt != nil {
		return domain.AuthorizationCode{}, storage.ErrAlreadyConsumed
	}
	if !c.ExpiresAt.IsZero() && now.After(c.ExpiresAt) {
		return domain.AuthorizationCode{}, storage.ErrExpired
	}
	c.ConsumedAt = &now
	nb, _ := enc(c)
	_, err = s.db.ExecContext(ctx, `UPDATE authorization_codes SET data=? WHERE hash=?`, nb, k)
	return c, err
}

func (s *Store) CreateAccessToken(ctx context.Context, t domain.AccessToken) error {
	b, _ := enc(t)
	_, err := s.db.ExecContext(ctx, `INSERT INTO access_tokens(hash,data) VALUES(?,?)`, hashKey(t.TokenHash), b)
	return mapDup(err)
}
func (s *Store) GetAccessToken(ctx context.Context, tokenHash []byte) (domain.AccessToken, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM access_tokens WHERE hash=?`, hashKey(tokenHash)).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.AccessToken{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.AccessToken{}, err
	}
	return dec[domain.AccessToken](b)
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

func (s *Store) CreateRefreshToken(ctx context.Context, t domain.RefreshToken) error {
	b, _ := enc(t)
	_, err := s.db.ExecContext(ctx, `INSERT INTO refresh_tokens(hash,grant_id,data) VALUES(?,?,?)`, hashKey(t.TokenHash), t.GrantID, b)
	return mapDup(err)
}
func (s *Store) GetRefreshToken(ctx context.Context, tokenHash []byte) (domain.RefreshToken, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM refresh_tokens WHERE hash=?`, hashKey(tokenHash)).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.RefreshToken{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.RefreshToken{}, err
	}
	return dec[domain.RefreshToken](b)
}
func (s *Store) RotateRefreshToken(ctx context.Context, oldHash []byte, next domain.RefreshToken, now time.Time) (domain.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, err := s.GetRefreshToken(ctx, oldHash)
	if err != nil {
		return domain.RefreshToken{}, err
	}
	if old.RevokedAt != nil || len(old.ReplacedByHash) > 0 || old.ReuseDetectedAt != nil {
		detected := now
		old.ReuseDetectedAt = &detected
		_ = s.putRefresh(ctx, old)
		_ = s.revokeFamily(ctx, old.GrantID, now)
		return domain.RefreshToken{}, storage.ErrRefreshReuseDetected
	}
	if !old.ExpiresAt.IsZero() && now.After(old.ExpiresAt) {
		return domain.RefreshToken{}, storage.ErrExpired
	}
	next.ParentTokenHash = append([]byte(nil), oldHash...)
	old.ReplacedByHash = append([]byte(nil), next.TokenHash...)
	if err := s.putRefresh(ctx, old); err != nil {
		return domain.RefreshToken{}, err
	}
	if err := s.CreateRefreshToken(ctx, next); err != nil {
		return domain.RefreshToken{}, err
	}
	return next, nil
}
func (s *Store) putRefresh(ctx context.Context, t domain.RefreshToken) error {
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
	var toks []domain.RefreshToken
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return err
		}
		t, _ := dec[domain.RefreshToken](b)
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

func (s *Store) CreateSession(ctx context.Context, sess domain.Session) error {
	b, _ := enc(sess)
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions(hash,data) VALUES(?,?)`, hashKey(sess.IDHash), b)
	return mapDup(err)
}
func (s *Store) GetSession(ctx context.Context, idHash []byte) (domain.Session, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM sessions WHERE hash=?`, hashKey(idHash)).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.Session{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.Session{}, err
	}
	return dec[domain.Session](b)
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

func (s *Store) CreateSigningKey(ctx context.Context, k domain.SigningKey) error {
	b, _ := enc(k)
	active := 0
	if k.Active {
		active = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO signing_keys(id,active,data) VALUES(?,?,?)`, k.ID, active, b)
	return mapDup(err)
}
func (s *Store) ActiveSigningKey(ctx context.Context) (domain.SigningKey, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM signing_keys WHERE active=1 LIMIT 1`).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.SigningKey{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.SigningKey{}, err
	}
	return dec[domain.SigningKey](b)
}
func (s *Store) VerificationKeys(ctx context.Context) ([]domain.SigningKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM signing_keys ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SigningKey
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		k, err := dec[domain.SigningKey](b)
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
		key domain.SigningKey
	}
	var all []row
	for rows.Next() {
		var id string
		var b []byte
		if err := rows.Scan(&id, &b); err != nil {
			return err
		}
		k, _ := dec[domain.SigningKey](b)
		k.Active = id == kid
		if id == kid {
			found = true
		}
		all = append(all, row{id, k})
	}
	if !found {
		return storage.ErrNotFound
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
func (s *Store) getSigningKey(ctx context.Context, kid string) (domain.SigningKey, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM signing_keys WHERE id=?`, kid).Scan(&b)
	if err == sql.ErrNoRows {
		return domain.SigningKey{}, storage.ErrNotFound
	}
	if err != nil {
		return domain.SigningKey{}, err
	}
	return dec[domain.SigningKey](b)
}

func mapDup(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(fmt.Sprint(err), "UNIQUE") {
		return storage.ErrDuplicate
	}
	return err
}
