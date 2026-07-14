package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type cleanupReport struct {
	LoginAttempts        int64
	RegistrationAttempts int64
	Sessions             int64
}

func (s *appStore) cleanup(ctx context.Context, now time.Time, retainTerminalFor time.Duration) (cleanupReport, error) {
	if retainTerminalFor < 0 {
		return cleanupReport{}, errors.New("terminal retention must not be negative")
	}
	cutoff := formatAppTime(now.Add(-retainTerminalFor))
	var report cleanupReport
	operations := []struct {
		query string
		count *int64
	}{
		{`DELETE FROM oidc_login_attempts WHERE expires_at <= ? OR (consumed_at IS NOT NULL AND consumed_at <= ?)`, &report.LoginAttempts},
		{`DELETE FROM registration_attempts WHERE expires_at <= ? OR (consumed_at IS NOT NULL AND consumed_at <= ?)`, &report.RegistrationAttempts},
		{`DELETE FROM app_sessions WHERE expires_at <= ? OR (revoked_at IS NOT NULL AND revoked_at <= ?)`, &report.Sessions},
	}
	for _, operation := range operations {
		result, err := s.db.ExecContext(ctx, operation.query, cutoff, cutoff)
		if err != nil {
			return cleanupReport{}, errors.Wrap(err, "clean application protocol state")
		}
		if *operation.count, err = result.RowsAffected(); err != nil {
			return cleanupReport{}, errors.Wrap(err, "read application cleanup result")
		}
	}
	return report, nil
}
