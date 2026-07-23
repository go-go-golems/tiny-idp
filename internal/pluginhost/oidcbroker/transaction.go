package oidcbroker

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

var (
	ErrTransactionNotFound       = errors.New("OIDC integration transaction not found")
	ErrTransactionExpired        = errors.New("OIDC integration transaction expired")
	ErrTransactionConsumed       = errors.New("OIDC integration transaction already consumed")
	ErrTransactionBinding        = errors.New("OIDC integration transaction browser binding mismatch")
	ErrTransactionPlugin         = errors.New("OIDC integration transaction plugin mismatch")
	ErrTransactionStateMalformed = errors.New("OIDC integration transaction state is malformed")
)

type transaction struct {
	StateHash          []byte
	PluginID           string
	ClientID           string
	CallbackPath       string
	NonceHash          []byte
	PKCEVerifierBox    []byte
	PluginStateBox     []byte
	BrowserBindingHash []byte
	CreatedAt          time.Time
	ExpiresAt          time.Time
	ConsumedAt         *time.Time
}

type ConsumedTransaction struct {
	ClientID     string
	CallbackPath string
	PKCEVerifier string
	PluginState  []byte
	NonceHash    []byte
}

type TransactionManager struct {
	db     *sql.DB
	aead   cipher.AEAD
	macKey [sha256.Size]byte
	random io.Reader
	now    func() time.Time
}

func NewTransactionManager(db *sql.DB, protectedKey []byte, random io.Reader, now func() time.Time) (*TransactionManager, error) {
	if db == nil || len(protectedKey) < 32 {
		return nil, errors.New("transaction database and at least 32 protected key bytes are required")
	}
	if random == nil {
		return nil, errors.New("transaction random source is required")
	}
	if now == nil {
		now = time.Now
	}
	encryptionKey := deriveKey(protectedKey, "tinyidp/plugin-oidc/transaction-encryption/v1")
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("construct transaction cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("construct transaction AEAD: %w", err)
	}
	manager := &TransactionManager{db: db, aead: aead, random: random, now: now}
	copy(manager.macKey[:], deriveKey(protectedKey, "tinyidp/plugin-oidc/transaction-mac/v1"))
	zeroBytes(encryptionKey)
	return manager, nil
}

type NewTransaction struct {
	PluginID       string
	ClientID       string
	CallbackPath   string
	PluginState    []byte
	BrowserBinding string
	TTL            time.Duration
}

type CreatedTransaction struct {
	State         string
	Nonce         string
	PKCEVerifier  string
	PKCEChallenge string
	ExpiresAt     time.Time
}

func (m *TransactionManager) Create(ctx context.Context, request NewTransaction) (CreatedTransaction, error) {
	if m == nil || m.db == nil || request.PluginID == "" || request.ClientID == "" ||
		request.CallbackPath == "" || request.BrowserBinding == "" || request.TTL <= 0 {
		return CreatedTransaction{}, errors.New("complete transaction creation fields are required")
	}
	state, err := m.randomToken(32)
	if err != nil {
		return CreatedTransaction{}, err
	}
	nonce, err := m.randomToken(32)
	if err != nil {
		return CreatedTransaction{}, err
	}
	verifier, err := m.randomToken(32)
	if err != nil {
		return CreatedTransaction{}, err
	}
	stateHash := m.keyedHash("state", state)
	nonceHash := m.keyedHash("nonce", nonce)
	bindingHash := m.keyedHash("browser", request.BrowserBinding)
	verifierBox, err := m.seal(stateHash, "pkce", []byte(verifier))
	if err != nil {
		return CreatedTransaction{}, err
	}
	stateBox, err := m.seal(stateHash, "plugin-state", request.PluginState)
	if err != nil {
		return CreatedTransaction{}, err
	}
	createdAt := m.now().UTC()
	expiresAt := createdAt.Add(request.TTL)
	_, err = m.db.ExecContext(ctx, `INSERT INTO integration_transactions
		(state_hash,plugin_id,client_id,callback_path,nonce_hash,pkce_verifier_box,plugin_state_box,browser_binding_hash,created_at_ns,expires_at_ns)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		stateHash, request.PluginID, request.ClientID, request.CallbackPath, nonceHash, verifierBox, stateBox, bindingHash,
		createdAt.UnixNano(), expiresAt.UnixNano())
	if err != nil {
		return CreatedTransaction{}, fmt.Errorf("create OIDC integration transaction: %w", err)
	}
	challenge := sha256.Sum256([]byte(verifier))
	return CreatedTransaction{
		State: state, Nonce: nonce, PKCEVerifier: verifier,
		PKCEChallenge: base64.RawURLEncoding.EncodeToString(challenge[:]), ExpiresAt: expiresAt,
	}, nil
}

func (m *TransactionManager) Consume(ctx context.Context, pluginID, browserBinding, rawState string) (ConsumedTransaction, error) {
	if m == nil || pluginID == "" || browserBinding == "" {
		return ConsumedTransaction{}, ErrTransactionStateMalformed
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(rawState); err != nil || len(decoded) != 32 {
		return ConsumedTransaction{}, ErrTransactionStateMalformed
	}
	stateHash := m.keyedHash("state", rawState)
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return ConsumedTransaction{}, fmt.Errorf("begin OIDC integration transaction consumption: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is a no-op after commit.
	record, err := loadTransaction(ctx, tx, stateHash)
	if err != nil {
		return ConsumedTransaction{}, err
	}
	now := m.now().UTC()
	switch {
	case record.ConsumedAt != nil:
		return ConsumedTransaction{}, ErrTransactionConsumed
	case !now.Before(record.ExpiresAt):
		return ConsumedTransaction{}, ErrTransactionExpired
	case record.PluginID != pluginID:
		return ConsumedTransaction{}, ErrTransactionPlugin
	case !hmac.Equal(record.BrowserBindingHash, m.keyedHash("browser", browserBinding)):
		return ConsumedTransaction{}, ErrTransactionBinding
	}
	result, err := tx.ExecContext(ctx, `UPDATE integration_transactions SET consumed_at_ns=?
		WHERE state_hash=? AND consumed_at_ns IS NULL`, now.UnixNano(), stateHash)
	if err != nil {
		return ConsumedTransaction{}, fmt.Errorf("consume OIDC integration transaction: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return ConsumedTransaction{}, ErrTransactionConsumed
	}
	verifier, err := m.open(stateHash, "pkce", record.PKCEVerifierBox)
	if err != nil {
		return ConsumedTransaction{}, err
	}
	pluginState, err := m.open(stateHash, "plugin-state", record.PluginStateBox)
	if err != nil {
		zeroBytes(verifier)
		return ConsumedTransaction{}, err
	}
	if err := tx.Commit(); err != nil {
		zeroBytes(verifier)
		zeroBytes(pluginState)
		return ConsumedTransaction{}, fmt.Errorf("commit OIDC integration transaction consumption: %w", err)
	}
	return ConsumedTransaction{
		ClientID: record.ClientID, CallbackPath: record.CallbackPath, PKCEVerifier: string(verifier),
		PluginState: pluginState, NonceHash: append([]byte(nil), record.NonceHash...),
	}, nil
}

func loadTransaction(ctx context.Context, tx *sql.Tx, stateHash []byte) (transaction, error) {
	var record transaction
	var createdAt, expiresAt int64
	var consumedAt sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT state_hash,plugin_id,client_id,callback_path,nonce_hash,
		pkce_verifier_box,plugin_state_box,browser_binding_hash,created_at_ns,expires_at_ns,consumed_at_ns
		FROM integration_transactions WHERE state_hash=?`, stateHash).Scan(
		&record.StateHash, &record.PluginID, &record.ClientID, &record.CallbackPath, &record.NonceHash,
		&record.PKCEVerifierBox, &record.PluginStateBox, &record.BrowserBindingHash, &createdAt, &expiresAt, &consumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return transaction{}, ErrTransactionNotFound
	}
	if err != nil {
		return transaction{}, fmt.Errorf("load OIDC integration transaction: %w", err)
	}
	record.CreatedAt = time.Unix(0, createdAt).UTC()
	record.ExpiresAt = time.Unix(0, expiresAt).UTC()
	if consumedAt.Valid {
		value := time.Unix(0, consumedAt.Int64).UTC()
		record.ConsumedAt = &value
	}
	return record, nil
}

func (m *TransactionManager) randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := io.ReadFull(m.random, value); err != nil {
		return "", fmt.Errorf("read transaction randomness: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (m *TransactionManager) keyedHash(domain, value string) []byte {
	mac := hmac.New(sha256.New, m.macKey[:])
	_, _ = mac.Write([]byte(domain))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (m *TransactionManager) nonceMatches(expected []byte, raw string) bool {
	return raw != "" && hmac.Equal(expected, m.keyedHash("nonce", raw))
}

func (m *TransactionManager) seal(stateHash []byte, domain string, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, m.aead.NonceSize())
	if _, err := io.ReadFull(m.random, nonce); err != nil {
		return nil, fmt.Errorf("read transaction encryption nonce: %w", err)
	}
	aad := append(append([]byte(nil), stateHash...), domain...)
	return m.aead.Seal(nonce, nonce, plaintext, aad), nil
}

func (m *TransactionManager) open(stateHash []byte, domain string, box []byte) ([]byte, error) {
	if len(box) < m.aead.NonceSize() {
		return nil, errors.New("OIDC integration transaction ciphertext is malformed")
	}
	aad := append(append([]byte(nil), stateHash...), domain...)
	plaintext, err := m.aead.Open(nil, box[:m.aead.NonceSize()], box[m.aead.NonceSize():], aad)
	if err != nil {
		return nil, errors.New("OIDC integration transaction ciphertext was not accepted")
	}
	return plaintext, nil
}

func deriveKey(key []byte, domain string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(domain))
	return mac.Sum(nil)
}

func zeroBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
