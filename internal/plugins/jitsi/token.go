package jitsi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
)

type Signer struct {
	mu     sync.Mutex
	key    []byte
	appID  string
	domain string
	ttl    time.Duration
	clock  pluginapi.Clock
	closed bool
}

type IssueRequest struct {
	Identity pluginapi.Identity
	Room     string
	Decision Decision
}

func NewSigner(secret []byte, appID, domain string, ttl time.Duration, clock pluginapi.Clock) (*Signer, error) {
	if len(secret) < 32 || appID == "" || domain == "" || ttl <= 0 || ttl > maxTokenTTL || clock == nil {
		return nil, errors.New("complete bounded Jitsi signer configuration is required")
	}
	return &Signer{key: append([]byte(nil), secret...), appID: appID, domain: domain, ttl: ttl, clock: clock}, nil
}

func (s *Signer) Issue(request IssueRequest) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || len(s.key) == 0 {
		return "", errors.New("jitsi signer is closed")
	}
	if request.Identity.Subject == "" || !roomPattern.MatchString(request.Room) ||
		!request.Decision.Allowed || strings.TrimSpace(request.Decision.DisplayName) == "" {
		return "", errors.New("jitsi token request was not accepted")
	}
	now := s.clock.Now().UTC()
	user := map[string]any{
		"id": request.Identity.Subject, "name": request.Decision.DisplayName,
		"moderator": request.Decision.Moderator,
	}
	if request.Decision.IncludeEmail && request.Identity.Email != "" {
		user["email"] = request.Identity.Email
	}
	payload := map[string]any{
		"iss": s.appID, "aud": s.appID, "sub": s.domain, "room": request.Room,
		"iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(s.ttl).Unix(),
		"context": map[string]any{"user": user},
	}
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *Signer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	zeroSecret(s.key)
	s.key = nil
	return nil
}
