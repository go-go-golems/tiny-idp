package fositeadapter

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/storage"
	fositejwt "github.com/ory/fosite/token/jwt"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

type sqlFositeStore struct {
	db            *sql.DB
	project       idpstore.Store
	config        *fosite.Config
	secrets       map[string]string
	authorizeHook func(string) error
	tokenHook     func(string) error
}

type sqlDBProvider interface {
	SQLDB() *sql.DB
}

func newSQLFositeStore(db *sql.DB, project idpstore.Store, config *fosite.Config, secrets map[string]string, authorizeHook, tokenHook func(string) error) (*sqlFositeStore, error) {
	return &sqlFositeStore{db: db, project: project, config: config, secrets: secrets, authorizeHook: authorizeHook, tokenHook: tokenHook}, nil
}

var _ storage.Transactional = (*sqlFositeStore)(nil)

type authorizeLifecycleContextKey struct{}

type authorizeLifecycle struct {
	tx *sql.Tx
}

func (s *sqlFositeStore) beginAuthorizeLifecycle(ctx context.Context, interaction *idpstore.InteractionRecord, now time.Time) (context.Context, func(bool) error, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ctx, nil, err
	}
	rollback := func(cause error) (context.Context, func(bool) error, error) {
		_ = tx.Rollback()
		return ctx, nil, cause
	}
	if interaction != nil {
		terminal := *interaction
		now = now.UTC()
		terminal.ConsumedAt = &now
		terminal.Outcome = idpstore.InteractionOutcomeApproved
		data, err := json.Marshal(terminal)
		if err != nil {
			return rollback(err)
		}
		result, err := tx.ExecContext(ctx, `UPDATE authorization_interactions SET consumed_at=?,data=? WHERE hash=? AND consumed_at IS NULL AND expires_at>?`, now, data, hex.EncodeToString(interaction.IDHash), now)
		if err != nil {
			return rollback(err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return rollback(err)
		}
		if count != 1 {
			return rollback(idpstore.ErrAlreadyConsumed)
		}
	}
	lifecycleContext := context.WithValue(ctx, authorizeLifecycleContextKey{}, &authorizeLifecycle{tx: tx})
	finish := func(commit bool) error {
		if !commit {
			return tx.Rollback()
		}
		if err := s.runAuthorizeHook("before_commit"); err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}
	return lifecycleContext, finish, nil
}

func (s *sqlFositeStore) runAuthorizeHook(point string) error {
	if s.authorizeHook == nil {
		return nil
	}
	return s.authorizeHook(point)
}

func (s *sqlFositeStore) authorizeExec(ctx context.Context, name, query string, args ...any) error {
	lifecycle, ok := ctx.Value(authorizeLifecycleContextKey{}).(*authorizeLifecycle)
	if !ok || lifecycle == nil || lifecycle.tx == nil {
		return fmt.Errorf("authorization persistence %s requires an active lifecycle transaction", name)
	}
	if err := s.runAuthorizeHook("before_" + name); err != nil {
		return err
	}
	if _, err := lifecycle.tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return s.runAuthorizeHook("after_" + name)
}

type tokenLifecycleContextKey struct{}

type tokenLifecycle struct {
	tx *sql.Tx
}

func (s *sqlFositeStore) BeginTX(ctx context.Context) (context.Context, error) {
	if lifecycle, ok := ctx.Value(tokenLifecycleContextKey{}).(*tokenLifecycle); ok && lifecycle != nil {
		return ctx, fmt.Errorf("nested Fosite token transaction")
	}
	if err := s.runTokenHook("before_begin_token"); err != nil {
		return ctx, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, tokenLifecycleContextKey{}, &tokenLifecycle{tx: tx}), nil
}

func (s *sqlFositeStore) Commit(ctx context.Context) error {
	lifecycle, err := tokenLifecycleFromContext(ctx)
	if err != nil {
		return err
	}
	if err := s.runTokenHook("before_commit_token"); err != nil {
		_ = lifecycle.tx.Rollback()
		return err
	}
	return lifecycle.tx.Commit()
}

func (s *sqlFositeStore) Rollback(ctx context.Context) error {
	lifecycle, err := tokenLifecycleFromContext(ctx)
	if err != nil {
		return err
	}
	return lifecycle.tx.Rollback()
}

func tokenLifecycleFromContext(ctx context.Context) (*tokenLifecycle, error) {
	lifecycle, ok := ctx.Value(tokenLifecycleContextKey{}).(*tokenLifecycle)
	if !ok || lifecycle == nil || lifecycle.tx == nil {
		return nil, fmt.Errorf("fosite token persistence requires an active transaction")
	}
	return lifecycle, nil
}

func (s *sqlFositeStore) runTokenHook(point string) error {
	if s.tokenHook == nil {
		return nil
	}
	return s.tokenHook(point)
}

func (s *sqlFositeStore) tokenExec(ctx context.Context, name, query string, args ...any) (sql.Result, error) {
	lifecycle, err := tokenLifecycleFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.runTokenHook("before_" + name); err != nil {
		return nil, err
	}
	result, err := lifecycle.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if err := s.runTokenHook("after_" + name); err != nil {
		return nil, err
	}
	return result, nil
}

// tinyidp:transaction-scoped -- uses the Fosite context transaction when present.
func (s *sqlFositeStore) tokenOrDirectExec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if lifecycle, ok := ctx.Value(tokenLifecycleContextKey{}).(*tokenLifecycle); ok && lifecycle != nil && lifecycle.tx != nil {
		return lifecycle.tx.ExecContext(ctx, query, args...)
	}
	return s.db.ExecContext(ctx, query, args...)
}

// consumeDeviceGrantInTokenTransaction makes RFC 8628 redemption part of the
// same SQL transaction that Fosite uses for access/refresh token persistence.
// It intentionally bypasses Store.Update: that callback would open a second
// transaction and break the all-or-nothing security invariant.
func (s *sqlFositeStore) consumeDeviceGrantInTokenTransaction(ctx context.Context, request idpstore.DeviceConsumeRequest) (idpstore.DeviceGrant, error) {
	if request.Now.IsZero() {
		return idpstore.DeviceGrant{}, idpstore.ErrInvalidDeviceGrant
	}
	lifecycle, err := tokenLifecycleFromContext(ctx)
	if err != nil {
		return idpstore.DeviceGrant{}, err
	}
	var data []byte
	err = lifecycle.tx.QueryRowContext(ctx, `SELECT data FROM device_grants WHERE device_code_hash=? AND client_id=?`, hex.EncodeToString(request.DeviceCodeHash), request.ClientID).Scan(&data)
	if err == sql.ErrNoRows {
		return idpstore.DeviceGrant{}, idpstore.ErrNotFound
	}
	if err != nil {
		return idpstore.DeviceGrant{}, err
	}
	var grant idpstore.DeviceGrant
	if err := json.Unmarshal(data, &grant); err != nil {
		return idpstore.DeviceGrant{}, err
	}
	now := request.Now.UTC()
	if !grant.ExpiresAt.IsZero() && !now.Before(grant.ExpiresAt) {
		return idpstore.DeviceGrant{}, idpstore.ErrExpired
	}
	if grant.Status == idpstore.DeviceGrantConsumed {
		return idpstore.DeviceGrant{}, idpstore.ErrAlreadyConsumed
	}
	if grant.Status != idpstore.DeviceGrantApproved {
		return idpstore.DeviceGrant{}, idpstore.ErrDeviceGrantNotApproved
	}
	grant.Status = idpstore.DeviceGrantConsumed
	grant.ConsumedAt = &now
	grant.Version++
	data, err = json.Marshal(grant)
	if err != nil {
		return idpstore.DeviceGrant{}, err
	}
	result, err := s.tokenExec(ctx, "consume_device_grant", `UPDATE device_grants SET status=?,data=? WHERE device_code_hash=? AND client_id=? AND status='approved' AND expires_at>?`, grant.Status, data, hex.EncodeToString(request.DeviceCodeHash), request.ClientID, now)
	if err != nil {
		return idpstore.DeviceGrant{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return idpstore.DeviceGrant{}, err
	}
	if count != 1 {
		return idpstore.DeviceGrant{}, idpstore.ErrDeviceGrantNotApproved
	}
	return grant, nil
}

func (s *sqlFositeStore) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	c, err := s.project.GetClient(ctx, id)
	if err != nil || c.Disabled {
		return nil, fosite.ErrNotFound
	}
	return s.toFositeClient(ctx, c)
}

func (s *sqlFositeStore) toFositeClient(ctx context.Context, c idpstore.Client) (fosite.Client, error) {
	fc := &fosite.DefaultClient{
		ID:            c.ID,
		Public:        c.Public,
		RedirectURIs:  append([]string(nil), c.RedirectURIs...),
		ResponseTypes: []string{"code"},
		GrantTypes:    append([]string(nil), c.AllowedGrantTypes...),
		Scopes:        append([]string(nil), c.AllowedScopes...),
		Audience:      append([]string(nil), c.AllowedAudiences...),
	}
	if !c.Public {
		if secret, ok := s.secrets[c.ID]; ok {
			hashed, err := (&fosite.BCrypt{Config: s.config}).Hash(ctx, []byte(secret))
			if err != nil {
				return nil, err
			}
			fc.Secret = hashed
		} else if len(c.SecretHash) > 0 && strings.HasPrefix(string(c.SecretHash), "$2") {
			fc.Secret = append([]byte(nil), c.SecretHash...)
		}
	}
	return clientWithLifespans(fc, c), nil
}

func clientWithLifespans(fc *fosite.DefaultClient, c idpstore.Client) fosite.Client {
	config := &fosite.ClientLifespanConfig{
		AuthorizationCodeGrantAccessTokenLifespan:  durationPointer(c.AccessTokenTTL),
		AuthorizationCodeGrantIDTokenLifespan:      durationPointer(c.IDTokenTTL),
		AuthorizationCodeGrantRefreshTokenLifespan: durationPointer(c.RefreshTokenTTL),
		RefreshTokenGrantAccessTokenLifespan:       durationPointer(c.AccessTokenTTL),
		RefreshTokenGrantIDTokenLifespan:           durationPointer(c.IDTokenTTL),
		RefreshTokenGrantRefreshTokenLifespan:      durationPointer(c.RefreshTokenTTL),
	}
	return &fosite.DefaultClientWithCustomTokenLifespans{DefaultClient: fc, TokenLifespans: config}
}

func durationPointer(value time.Duration) *time.Duration {
	if value <= 0 {
		return nil
	}
	return &value
}

type persistedRequest struct {
	ID                string           `json:"id"`
	RequestedAt       time.Time        `json:"requested_at"`
	Client            persistedClient  `json:"client"`
	RequestedScope    []string         `json:"requested_scope"`
	GrantedScope      []string         `json:"granted_scope"`
	Form              url.Values       `json:"form"`
	Session           persistedSession `json:"session"`
	RequestedAudience []string         `json:"requested_audience"`
	GrantedAudience   []string         `json:"granted_audience"`
}

type persistedClient struct {
	ID              string        `json:"id"`
	Secret          []byte        `json:"secret,omitempty"`
	RedirectURIs    []string      `json:"redirect_uris"`
	GrantTypes      []string      `json:"grant_types"`
	ResponseTypes   []string      `json:"response_types"`
	Scopes          []string      `json:"scopes"`
	Audience        []string      `json:"audience"`
	Public          bool          `json:"public"`
	AccessTokenTTL  time.Duration `json:"access_token_ttl,omitempty"`
	IDTokenTTL      time.Duration `json:"id_token_ttl,omitempty"`
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl,omitempty"`
}

type persistedSession struct {
	Claims    *fositejwt.IDTokenClaims       `json:"claims,omitempty"`
	Headers   *fositejwt.Headers             `json:"headers,omitempty"`
	ExpiresAt map[fosite.TokenType]time.Time `json:"expires_at,omitempty"`
	Username  string                         `json:"username,omitempty"`
	Subject   string                         `json:"subject,omitempty"`
}

func persistRequester(req fosite.Requester) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("nil requester")
	}
	pc := persistedClient{}
	if c := req.GetClient(); c != nil {
		pc = persistedClient{ID: c.GetID(), Secret: append([]byte(nil), c.GetHashedSecret()...), RedirectURIs: append([]string(nil), c.GetRedirectURIs()...), GrantTypes: append([]string(nil), c.GetGrantTypes()...), ResponseTypes: append([]string(nil), c.GetResponseTypes()...), Scopes: append([]string(nil), c.GetScopes()...), Audience: append([]string(nil), c.GetAudience()...), Public: c.IsPublic(), AccessTokenTTL: fosite.GetEffectiveLifespan(c, fosite.GrantTypeAuthorizationCode, fosite.AccessToken, 0), IDTokenTTL: fosite.GetEffectiveLifespan(c, fosite.GrantTypeAuthorizationCode, fosite.IDToken, 0), RefreshTokenTTL: fosite.GetEffectiveLifespan(c, fosite.GrantTypeAuthorizationCode, fosite.RefreshToken, 0)}
	}
	ps := persistedSession{}
	if sess := req.GetSession(); sess != nil {
		ps.Subject = sess.GetSubject()
		ps.Username = sess.GetUsername()
		ps.ExpiresAt = map[fosite.TokenType]time.Time{}
		for _, tt := range []fosite.TokenType{fosite.AccessToken, fosite.RefreshToken, fosite.AuthorizeCode, fosite.IDToken} {
			if exp := sess.GetExpiresAt(tt); !exp.IsZero() {
				ps.ExpiresAt[tt] = exp
			}
		}
		if oidc, ok := sess.(*openid.DefaultSession); ok {
			ps.Claims = oidc.Claims
			ps.Headers = oidc.Headers
			ps.Subject = oidc.Subject
			ps.Username = oidc.Username
			if oidc.ExpiresAt != nil {
				ps.ExpiresAt = oidc.ExpiresAt
			}
		}
	}
	pr := persistedRequest{ID: req.GetID(), RequestedAt: req.GetRequestedAt(), Client: pc, RequestedScope: append([]string(nil), req.GetRequestedScopes()...), GrantedScope: append([]string(nil), req.GetGrantedScopes()...), Form: cloneValues(req.GetRequestForm()), Session: ps, RequestedAudience: append([]string(nil), req.GetRequestedAudience()...), GrantedAudience: append([]string(nil), req.GetGrantedAudience()...)}
	return json.Marshal(pr)
}

func restoreRequester(b []byte) (fosite.Requester, error) {
	var pr persistedRequest
	if err := json.Unmarshal(b, &pr); err != nil {
		return nil, err
	}
	fc := &fosite.DefaultClient{ID: pr.Client.ID, Secret: pr.Client.Secret, RedirectURIs: pr.Client.RedirectURIs, GrantTypes: pr.Client.GrantTypes, ResponseTypes: pr.Client.ResponseTypes, Scopes: pr.Client.Scopes, Audience: pr.Client.Audience, Public: pr.Client.Public}
	client := clientWithLifespans(fc, idpstore.Client{AccessTokenTTL: pr.Client.AccessTokenTTL, IDTokenTTL: pr.Client.IDTokenTTL, RefreshTokenTTL: pr.Client.RefreshTokenTTL})
	sess := &openid.DefaultSession{Claims: pr.Session.Claims, Headers: pr.Session.Headers, ExpiresAt: pr.Session.ExpiresAt, Username: pr.Session.Username, Subject: pr.Session.Subject}
	if sess.Claims == nil {
		sess.Claims = &fositejwt.IDTokenClaims{Subject: pr.Session.Subject, Extra: map[string]interface{}{}}
	}
	if sess.Headers == nil {
		sess.Headers = &fositejwt.Headers{}
	}
	if sess.ExpiresAt == nil {
		sess.ExpiresAt = map[fosite.TokenType]time.Time{}
	}
	return &fosite.Request{ID: pr.ID, RequestedAt: pr.RequestedAt, Client: client, RequestedScope: fosite.Arguments(pr.RequestedScope), GrantedScope: fosite.Arguments(pr.GrantedScope), Form: pr.Form, Session: sess, RequestedAudience: fosite.Arguments(pr.RequestedAudience), GrantedAudience: fosite.Arguments(pr.GrantedAudience)}, nil
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, values := range v {
		out[k] = append([]string(nil), values...)
	}
	return out
}

func (s *sqlFositeStore) CreateAuthorizeCodeSession(ctx context.Context, code string, req fosite.Requester) error {
	b, err := persistRequester(req)
	if err != nil {
		return err
	}
	return s.authorizeExec(ctx, "authorize_code", `INSERT INTO fosite_authorize_codes(signature,active,subject,request_json) VALUES(?,?,?,?)`, code, 1, requesterSubject(req), b)
}
func (s *sqlFositeStore) GetAuthorizeCodeSession(ctx context.Context, code string, _ fosite.Session) (fosite.Requester, error) {
	var active int
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT active,request_json FROM fosite_authorize_codes WHERE signature=?`, code).Scan(&active, &b)
	if err == sql.ErrNoRows {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	req, err := s.restoreActiveRequester(ctx, b)
	if err != nil {
		return nil, err
	}
	if active == 0 {
		return req, fosite.ErrInvalidatedAuthorizeCode
	}
	return req, nil
}
func (s *sqlFositeStore) InvalidateAuthorizeCodeSession(ctx context.Context, code string) error {
	res, err := s.tokenExec(ctx, "invalidate_authorize_code", `UPDATE fosite_authorize_codes SET active=0 WHERE signature=?`, code)
	if err != nil {
		return err
	}
	return notFoundIfZero(res)
}

func (s *sqlFositeStore) CreatePKCERequestSession(ctx context.Context, code string, req fosite.Requester) error {
	b, err := persistRequester(req)
	if err != nil {
		return err
	}
	return s.authorizeExec(ctx, "pkce", `INSERT INTO fosite_pkces(signature,subject,request_json) VALUES(?,?,?)`, code, requesterSubject(req), b)
}
func (s *sqlFositeStore) GetPKCERequestSession(ctx context.Context, code string, _ fosite.Session) (fosite.Requester, error) {
	return s.getRequester(ctx, `SELECT request_json FROM fosite_pkces WHERE signature=?`, code)
}
func (s *sqlFositeStore) DeletePKCERequestSession(ctx context.Context, code string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM fosite_pkces WHERE signature=?`, code)
	return err
}

func (s *sqlFositeStore) CreateOpenIDConnectSession(ctx context.Context, authorizeCode string, req fosite.Requester) error {
	b, err := persistRequester(req)
	if err != nil {
		return err
	}
	return s.authorizeExec(ctx, "oidc", `INSERT INTO fosite_oidc_sessions(signature,subject,request_json) VALUES(?,?,?)`, authorizeCode, requesterSubject(req), b)
}
func (s *sqlFositeStore) GetOpenIDConnectSession(ctx context.Context, authorizeCode string, requester fosite.Requester) (fosite.Requester, error) {
	return s.getRequester(ctx, `SELECT request_json FROM fosite_oidc_sessions WHERE signature=?`, authorizeCode)
}
func (s *sqlFositeStore) DeleteOpenIDConnectSession(ctx context.Context, authorizeCode string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM fosite_oidc_sessions WHERE signature=?`, authorizeCode)
	return err
}

func (s *sqlFositeStore) CreateAccessTokenSession(ctx context.Context, signature string, req fosite.Requester) error {
	b, err := persistRequester(req)
	if err != nil {
		return err
	}
	_, err = s.tokenExec(ctx, "create_access_token", `INSERT INTO fosite_access_tokens(signature,request_id,subject,request_json) VALUES(?,?,?,?)`, signature, req.GetID(), requesterSubject(req), b)
	return err
}
func (s *sqlFositeStore) GetAccessTokenSession(ctx context.Context, signature string, _ fosite.Session) (fosite.Requester, error) {
	return s.getRequester(ctx, `SELECT request_json FROM fosite_access_tokens WHERE signature=?`, signature)
}
func (s *sqlFositeStore) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM fosite_access_tokens WHERE signature=?`, signature)
	return err
}

func (s *sqlFositeStore) CreateRefreshTokenSession(ctx context.Context, signature, accessTokenSignature string, req fosite.Requester) error {
	b, err := persistRequester(req)
	if err != nil {
		return err
	}
	_, err = s.tokenExec(ctx, "create_refresh_token", `INSERT INTO fosite_refresh_tokens(signature,request_id,active,access_token_signature,subject,request_json) VALUES(?,?,?,?,?,?)`, signature, req.GetID(), 1, accessTokenSignature, requesterSubject(req), b)
	return err
}

func requesterSubject(req fosite.Requester) string {
	if req == nil || req.GetSession() == nil {
		return ""
	}
	return req.GetSession().GetSubject()
}
func (s *sqlFositeStore) GetRefreshTokenSession(ctx context.Context, signature string, _ fosite.Session) (fosite.Requester, error) {
	var active int
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT active,request_json FROM fosite_refresh_tokens WHERE signature=?`, signature).Scan(&active, &b)
	if err == sql.ErrNoRows {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	req, err := s.restoreActiveRequester(ctx, b)
	if err != nil {
		return nil, err
	}
	if active == 0 {
		return req, fosite.ErrInactiveToken
	}
	return req, nil
}
func (s *sqlFositeStore) DeleteRefreshTokenSession(ctx context.Context, signature string) error {
	_, err := s.tokenOrDirectExec(ctx, `DELETE FROM fosite_refresh_tokens WHERE signature=?`, signature)
	return err
}

func (s *sqlFositeStore) RevokeRefreshToken(ctx context.Context, requestID string) error {
	_, err := s.tokenOrDirectExec(ctx, `UPDATE fosite_refresh_tokens SET active=0 WHERE request_id=?`, requestID)
	return err
}
func (s *sqlFositeStore) RevokeAccessToken(ctx context.Context, requestID string) error {
	_, err := s.tokenOrDirectExec(ctx, `DELETE FROM fosite_access_tokens WHERE request_id=?`, requestID)
	return err
}

// tinyidp:transaction-scoped -- Fosite BeginTX owns commit and rollback.
func (s *sqlFositeStore) RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) error {
	lifecycle, err := tokenLifecycleFromContext(ctx)
	if err != nil {
		return err
	}
	if err := s.runTokenHook("before_rotate_refresh"); err != nil {
		return err
	}
	result, err := lifecycle.tx.ExecContext(ctx, `UPDATE fosite_refresh_tokens SET active=0 WHERE request_id=? AND signature=? AND active=1`, requestID, refreshTokenSignature)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fosite.ErrSerializationFailure
	}
	if err := s.runTokenHook("after_revoke_refresh"); err != nil {
		return err
	}
	if _, err := lifecycle.tx.ExecContext(ctx, `DELETE FROM fosite_access_tokens WHERE request_id=?`, requestID); err != nil {
		return err
	}
	if err := s.runTokenHook("after_delete_old_access"); err != nil {
		return err
	}
	return s.runTokenHook("after_rotate_refresh")
}

func (s *sqlFositeStore) getRequester(ctx context.Context, query, signature string) (fosite.Requester, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, query, signature).Scan(&b)
	if err == sql.ErrNoRows {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.restoreActiveRequester(ctx, b)
}

func (s *sqlFositeStore) restoreActiveRequester(ctx context.Context, b []byte) (fosite.Requester, error) {
	req, err := restoreRequester(b)
	if err != nil {
		return nil, err
	}
	client := req.GetClient()
	if client == nil || client.GetID() == "" {
		return nil, fosite.ErrNotFound
	}
	domainClient, err := s.project.GetClient(ctx, client.GetID())
	if err != nil || domainClient.Disabled {
		return nil, fosite.ErrNotFound
	}
	return req, nil
}

func (s *sqlFositeStore) Authenticate(_ context.Context, name string, secret string) (string, error) {
	return "", fosite.ErrNotFound
}
func (s *sqlFositeStore) ClientAssertionJWTValid(ctx context.Context, jti string) error {
	used, err := s.IsJWTUsed(ctx, jti)
	if err != nil {
		return err
	}
	if used {
		return fosite.ErrJTIKnown
	}
	return nil
}
func (s *sqlFositeStore) SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error {
	return s.MarkJWTUsedForTime(ctx, jti, exp)
}
func (s *sqlFositeStore) IsJWTUsed(ctx context.Context, jti string) (bool, error) {
	var exp time.Time
	err := s.db.QueryRowContext(ctx, `SELECT expires_at FROM fosite_jtis WHERE jti=?`, jti).Scan(&exp)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if time.Now().After(exp) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM fosite_jtis WHERE jti=?`, jti)
		return false, nil
	}
	return true, nil
}
func (s *sqlFositeStore) MarkJWTUsedForTime(ctx context.Context, jti string, exp time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO fosite_jtis(jti,expires_at) VALUES(?,?)`, jti, exp)
	return err
}

func notFoundIfZero(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fosite.ErrNotFound
	}
	return nil
}
