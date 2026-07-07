package fositeadapter

import (
	"errors"
	"strings"

	"github.com/ory/fosite"
)

func auditReason(err error) string {
	if err == nil {
		return ""
	}
	var rfc *fosite.RFC6749Error
	if errors.As(err, &rfc) {
		return cleanAuditReason(rfc.Error())
	}
	return "internal_error"
}

func cleanAuditReason(reason string) string {
	reason = strings.TrimSpace(strings.ToLower(reason))
	if reason == "" {
		return "internal_error"
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range reason {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "internal_error"
	}
	return out
}
