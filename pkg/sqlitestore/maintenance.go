package sqlitestore

import (
	"context"
	"fmt"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

// Maintain removes terminal domain records, old Fosite protocol state, expired
// JTIs, and retired signing keys in one SQLite transaction.
func (s *Store) Maintain(ctx context.Context, now time.Time, policy idpstore.MaintenancePolicy) (idpstore.MaintenanceReport, error) {
	if ctx == nil {
		return idpstore.MaintenanceReport{}, fmt.Errorf("context is required")
	}
	if policy.RetainExpiredFor < 0 || policy.ProtocolStateRetention <= 0 || policy.SigningKeyRetention <= 0 {
		return idpstore.MaintenanceReport{}, fmt.Errorf("invalid maintenance retention policy")
	}
	now = now.UTC()
	report := idpstore.MaintenanceReport{StartedAt: now}
	err := s.Update(ctx, func(tx idpstore.TxStore) error {
		scoped, ok := tx.(*Store)
		if !ok {
			return fmt.Errorf("unexpected SQLite transaction implementation")
		}
		domainCutoff := now.Add(-policy.RetainExpiredFor)
		var err error
		add := func(n int64, e error) error {
			report.DomainRecords += n
			return e
		}
		if err = add(deleteJSONRecords[idpstore.Grant](ctx, scoped.conn(), "grants", "id", func(v idpstore.Grant) bool {
			return expiredOrTerminal(v.ExpiresAt, v.RevokedAt, domainCutoff)
		})); err != nil {
			return err
		}
		if err = add(deleteJSONRecords[idpstore.AuthorizationCode](ctx, scoped.conn(), "authorization_codes", "hash", func(v idpstore.AuthorizationCode) bool {
			return expiredOrTerminal(v.ExpiresAt, v.ConsumedAt, domainCutoff)
		})); err != nil {
			return err
		}
		if err = add(deleteJSONRecords[idpstore.AccessToken](ctx, scoped.conn(), "access_tokens", "hash", func(v idpstore.AccessToken) bool {
			return expiredOrTerminal(v.ExpiresAt, v.RevokedAt, domainCutoff)
		})); err != nil {
			return err
		}
		if err = add(deleteJSONRecords[idpstore.RefreshToken](ctx, scoped.conn(), "refresh_tokens", "hash", func(v idpstore.RefreshToken) bool {
			terminal := v.RevokedAt
			if terminal == nil {
				terminal = v.ReuseDetectedAt
			}
			return expiredOrTerminal(v.ExpiresAt, terminal, domainCutoff)
		})); err != nil {
			return err
		}
		if err = add(deleteJSONRecords[idpstore.Consent](ctx, scoped.conn(), "consents", "key", func(v idpstore.Consent) bool {
			return expiredOrTerminal(v.ExpiresAt, v.RevokedAt, domainCutoff)
		})); err != nil {
			return err
		}
		if err = add(deleteJSONRecords[idpstore.Session](ctx, scoped.conn(), "sessions", "hash", func(v idpstore.Session) bool {
			return expiredOrTerminal(v.ExpiresAt, v.RevokedAt, domainCutoff)
		})); err != nil {
			return err
		}
		if err = add(deleteJSONRecords[idpstore.InteractionRecord](ctx, scoped.conn(), "authorization_interactions", "hash", func(v idpstore.InteractionRecord) bool {
			return expiredOrTerminal(v.ExpiresAt, v.ConsumedAt, domainCutoff)
		})); err != nil {
			return err
		}

		protocolCutoff := now.Add(-policy.ProtocolStateRetention)
		for _, table := range []string{"fosite_authorize_codes", "fosite_pkces", "fosite_oidc_sessions", "fosite_access_tokens", "fosite_refresh_tokens"} {
			result, err := scoped.conn().ExecContext(ctx, `DELETE FROM `+table+` WHERE created_at < ?`, protocolCutoff)
			if err != nil {
				return fmt.Errorf("maintain %s: %w", table, err)
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			report.ProtocolRecords += count
		}
		result, err := scoped.conn().ExecContext(ctx, `DELETE FROM fosite_jtis WHERE expires_at < ?`, domainCutoff)
		if err != nil {
			return fmt.Errorf("maintain fosite_jtis: %w", err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		report.ProtocolRecords += count

		keyCutoff := now.Add(-policy.SigningKeyRetention)
		count, err = deleteJSONRecords[idpstore.SigningKey](ctx, scoped.conn(), "signing_keys", "id", func(v idpstore.SigningKey) bool {
			return !v.Active && !v.NotAfter.IsZero() && v.NotAfter.Before(keyCutoff)
		})
		if err != nil {
			return err
		}
		report.RetiredSigningKeys += count
		return nil
	})
	report.FinishedAt = time.Now().UTC()
	return report, err
}

func expiredOrTerminal(expiresAt time.Time, terminalAt *time.Time, cutoff time.Time) bool {
	return (!expiresAt.IsZero() && expiresAt.Before(cutoff)) || (terminalAt != nil && terminalAt.Before(cutoff))
}

// deleteJSONRecords is called only with a transaction-scoped runner.
//
// tinyidp:transaction-scoped
func deleteJSONRecords[T any](ctx context.Context, runner sqlRunner, table, keyColumn string, remove func(T) bool) (int64, error) {
	rows, err := runner.QueryContext(ctx, `SELECT `+keyColumn+`,data FROM `+table)
	if err != nil {
		return 0, err
	}
	var keys []string
	for rows.Next() {
		var key string
		var data []byte
		if err := rows.Scan(&key, &data); err != nil {
			_ = rows.Close()
			return 0, err
		}
		value, err := dec[T](data)
		if err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("decode %s maintenance record: %w", table, err)
		}
		if remove(value) {
			keys = append(keys, key)
		}
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, key := range keys {
		if _, err := runner.ExecContext(ctx, `DELETE FROM `+table+` WHERE `+keyColumn+`=?`, key); err != nil {
			return 0, err
		}
	}
	return int64(len(keys)), nil
}
