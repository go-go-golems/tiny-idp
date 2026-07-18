package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type DoctorReport struct {
	Checks []Check `json:"checks"`
	OK     bool    `json:"ok"`
}

func (s *Service) Doctor(ctx context.Context) DoctorReport {
	report := DoctorReport{OK: true}
	add := func(name, status, message string) {
		if status != "ok" {
			report.OK = false
		}
		report.Checks = append(report.Checks, Check{Name: name, Status: status, Message: message})
	}
	if schema, ok := s.Store.(idpstore.SchemaReporter); ok {
		version, err := schema.SchemaVersion(ctx)
		if err != nil {
			add("schema.version", "error", err.Error())
		} else if version != schema.SupportedSchemaVersion() {
			add("schema.version", "error", fmt.Sprintf("database=%d supported=%d", version, schema.SupportedSchemaVersion()))
		} else {
			add("schema.version", "ok", fmt.Sprintf("version %d", version))
		}
	} else {
		add("schema.version", "error", "schema reporting unavailable")
	}
	clients, err := s.Store.ListClients(ctx)
	if err != nil {
		add("clients.load", "error", err.Error())
	} else {
		add("clients.load", "ok", fmt.Sprintf("%d clients", len(clients)))
		for _, c := range clients {
			if err := c.Validate(idpstore.ProductionMode); err != nil {
				add("client."+c.ID, "error", err.Error())
			}
		}
	}
	active, err := s.Store.ActiveSigningKey(ctx)
	if err != nil {
		if errors.Is(err, idpstore.ErrNotFound) {
			add("keys.active", "error", "no active signing key")
		} else {
			add("keys.active", "error", err.Error())
		}
	} else if !active.NotAfter.IsZero() && s.Clock().UTC().After(active.NotAfter) {
		add("keys.active", "error", "active signing key is expired")
	} else {
		add("keys.active", "ok", active.ID)
	}
	verificationKeys, err := s.Store.VerificationKeys(ctx)
	if err != nil {
		add("keys.verification", "error", err.Error())
	} else {
		add("keys.verification", "ok", fmt.Sprintf("%d verification keys", len(verificationKeys)))
		for _, key := range verificationKeys {
			if !key.NotAfter.IsZero() && s.Clock().UTC().After(key.NotAfter.Add(24*time.Hour)) {
				add("key."+key.ID, "warn", "retired key remains published after not_after grace period")
			}
		}
	}
	return report
}
