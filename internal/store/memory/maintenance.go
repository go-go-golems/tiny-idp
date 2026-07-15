package memory

import (
	"context"
	"fmt"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func (s *Store) Maintain(ctx context.Context, now time.Time, policy idpstore.MaintenancePolicy) (idpstore.MaintenanceReport, error) {
	if ctx == nil {
		return idpstore.MaintenanceReport{}, fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return idpstore.MaintenanceReport{}, err
	}
	if policy.RetainExpiredFor < 0 || policy.ProtocolStateRetention <= 0 || policy.SigningKeyRetention <= 0 {
		return idpstore.MaintenanceReport{}, fmt.Errorf("invalid maintenance retention policy")
	}
	now = now.UTC()
	report := idpstore.MaintenanceReport{StartedAt: now}
	cutoff := now.Add(-policy.RetainExpiredFor)
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, value := range s.grants {
		if expiredMemory(value.ExpiresAt, value.RevokedAt, cutoff) {
			delete(s.grants, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.codes {
		if expiredMemory(value.ExpiresAt, value.ConsumedAt, cutoff) {
			delete(s.codes, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.access {
		if expiredMemory(value.ExpiresAt, value.RevokedAt, cutoff) {
			delete(s.access, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.refresh {
		terminal := value.RevokedAt
		if terminal == nil {
			terminal = value.ReuseDetectedAt
		}
		if expiredMemory(value.ExpiresAt, terminal, cutoff) {
			delete(s.refresh, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.consents {
		if expiredMemory(value.ExpiresAt, value.RevokedAt, cutoff) {
			delete(s.consents, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.sessions {
		if expiredMemory(value.ExpiresAt, value.RevokedAt, cutoff) {
			delete(s.sessions, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.browserContexts {
		if expiredMemory(value.ExpiresAt, value.RevokedAt, cutoff) {
			delete(s.browserContexts, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.rememberedSessions {
		_, contextExists := s.browserContexts[hashKey(value.ContextIDHash)]
		_, sessionExists := s.sessions[hashKey(value.SessionIDHash)]
		if (value.RemovedAt != nil && value.RemovedAt.Before(cutoff)) || !contextExists || !sessionExists {
			delete(s.rememberedSessions, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.interactions {
		if expiredMemory(value.ExpiresAt, value.ConsumedAt, cutoff) {
			delete(s.interactions, key)
			report.DomainRecords++
		}
	}
	for key, value := range s.deviceGrants {
		terminal := value.ConsumedAt
		if terminal == nil && value.Status == idpstore.DeviceGrantDenied {
			terminal = value.DecidedAt
		}
		if expiredMemory(value.ExpiresAt, terminal, cutoff) {
			delete(s.deviceGrants, key)
			delete(s.deviceByUserCode, hashKey(value.UserCodeHash))
			report.DomainRecords++
		}
	}
	keyCutoff := now.Add(-policy.SigningKeyRetention)
	for key, value := range s.keys {
		if !value.Active && !value.NotAfter.IsZero() && value.NotAfter.Before(keyCutoff) {
			delete(s.keys, key)
			report.RetiredSigningKeys++
		}
	}
	report.FinishedAt = time.Now().UTC()
	return report, nil
}

func expiredMemory(expiresAt time.Time, terminalAt *time.Time, cutoff time.Time) bool {
	return (!expiresAt.IsZero() && expiresAt.Before(cutoff)) || (terminalAt != nil && terminalAt.Before(cutoff))
}
