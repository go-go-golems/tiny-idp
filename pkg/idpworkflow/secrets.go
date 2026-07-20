package idpworkflow

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/pkg/errors"
)

// SecretHandle identifies one request-scoped secret without containing secret
// bytes. Its token is intentionally not exported as a struct field, so normal
// JSON encoding cannot turn it into JavaScript data.
type SecretHandle struct{ token string }

// Token is for trusted native effect validation only. JavaScript receives an
// identity-branded Goja object, never this Go value or its token.
func (h SecretHandle) Token() string { return h.token }

// SecretSet owns secret bytes collected from one browser submission. It is
// deliberately short lived: callers resolve a handle only for immediate native
// commit work and then call Destroy.
type SecretSet struct {
	values map[string][]byte
}

func newSecretSet(values map[FieldID][]byte) (*SecretSet, map[FieldID]SecretHandle, error) {
	set := &SecretSet{values: make(map[string][]byte, len(values))}
	handles := make(map[FieldID]SecretHandle, len(values))
	for field, value := range values {
		token, err := newSecretToken()
		if err != nil {
			set.Destroy()
			return nil, nil, err
		}
		set.values[token] = append([]byte(nil), value...)
		handles[field] = SecretHandle{token: token}
	}
	return set, handles, nil
}

func (s *SecretSet) Resolve(handle SecretHandle) ([]byte, bool) {
	if s == nil || handle.token == "" {
		return nil, false
	}
	value, ok := s.values[handle.token]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), value...), true
}

func (s *SecretSet) Destroy() {
	if s == nil {
		return
	}
	for token, value := range s.values {
		clear(value)
		delete(s.values, token)
	}
}

func newSecretToken() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", errors.Wrap(err, "generate workflow secret handle")
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
