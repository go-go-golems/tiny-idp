package jitsi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
)

type fixedClock struct{ value time.Time }

func (c fixedClock) Now() time.Time { return c.value }

func TestSignerIssuesExactShortRoomBoundHS256Claims(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	secret := []byte("0123456789abcdef0123456789abcdef")
	signer, err := NewSigner(secret, "tinyidp-jitsi", "meet.example.test", 5*time.Minute, fixedClock{now})
	if err != nil {
		t.Fatal(err)
	}
	defer signer.Close()
	token, err := signer.Issue(IssueRequest{
		Identity: pluginapi.Identity{Subject: "user-123", Email: "user@example.test"},
		Room:     "engineering",
		Decision: Decision{Allowed: true, DisplayName: "Test User", IncludeEmail: true, Moderator: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d", len(parts))
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		t.Fatal("token signature was not accepted with configured secret")
	}
	wrongMAC := hmac.New(sha256.New, []byte("abcdef0123456789abcdef0123456789"))
	_, _ = wrongMAC.Write([]byte(parts[0] + "." + parts[1]))
	if hmac.Equal(signature, wrongMAC.Sum(nil)) {
		t.Fatal("token signature was accepted with wrong secret")
	}
	body, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	if err := json.Unmarshal(body, &claims); err != nil {
		t.Fatal(err)
	}
	if claims["iss"] != "tinyidp-jitsi" || claims["aud"] != "tinyidp-jitsi" ||
		claims["sub"] != "meet.example.test" || claims["room"] != "engineering" ||
		int64(claims["exp"].(float64)) != now.Add(5*time.Minute).Unix() {
		t.Fatalf("claims = %#v", claims)
	}
	contextClaims := claims["context"].(map[string]any)
	user := contextClaims["user"].(map[string]any)
	if user["id"] != "user-123" || user["name"] != "Test User" || user["email"] != "user@example.test" || user["moderator"] != true {
		t.Fatalf("user claims = %#v", user)
	}
}

func TestSignerRejectsWrongRoomAndOmitsPrivateEmail(t *testing.T) {
	signer, err := NewSigner([]byte("0123456789abcdef0123456789abcdef"), "app", "meet.example.test", time.Minute, fixedClock{time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	for _, room := range []string{"", "*", "../admin", "with space"} {
		if _, err := signer.Issue(IssueRequest{
			Identity: pluginapi.Identity{Subject: "user-123"}, Room: room,
			Decision: Decision{Allowed: true, DisplayName: "User"},
		}); err == nil {
			t.Fatalf("room %q accepted", room)
		}
	}
	token, err := signer.Issue(IssueRequest{
		Identity: pluginapi.Identity{Subject: "user-123", Email: "private@example.test"}, Room: "safe-room",
		Decision: Decision{Allowed: true, DisplayName: "User", IncludeEmail: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(token, "private@example.test") {
		t.Fatal("plaintext email appeared in encoded token")
	}
	parts := strings.Split(token, ".")
	body, _ := base64.RawURLEncoding.DecodeString(parts[1])
	if strings.Contains(string(body), "private@example.test") || strings.Contains(string(body), `"email"`) {
		t.Fatalf("email leaked into token body: %s", body)
	}
	if err := signer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Issue(IssueRequest{
		Identity: pluginapi.Identity{Subject: "user-123"}, Room: "safe-room",
		Decision: Decision{Allowed: true, DisplayName: "User"},
	}); err == nil {
		t.Fatal("closed signer issued token")
	}
}

func TestProsodyContractRejectsExpiredWrongAppDomainAndRoom(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	secret := []byte("0123456789abcdef0123456789abcdef")
	issue := func(appID, domain, room string, issuedAt time.Time) string {
		signer, err := NewSigner(secret, appID, domain, time.Minute, fixedClock{issuedAt})
		if err != nil {
			t.Fatal(err)
		}
		defer signer.Close()
		token, err := signer.Issue(IssueRequest{
			Identity: pluginapi.Identity{Subject: "user-123"}, Room: room,
			Decision: Decision{Allowed: true, DisplayName: "User"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	valid := issue("app", "meet.example.test", "engineering", now)
	if err := verifyProsodyContract(valid, secret, "app", "meet.example.test", "engineering", now); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	cases := []struct {
		name  string
		token string
	}{
		{name: "expired", token: issue("app", "meet.example.test", "engineering", now.Add(-2*time.Minute))},
		{name: "wrong app", token: issue("other-app", "meet.example.test", "engineering", now)},
		{name: "wrong domain", token: issue("app", "other.example.test", "engineering", now)},
		{name: "wrong room", token: issue("app", "meet.example.test", "other-room", now)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := verifyProsodyContract(test.token, secret, "app", "meet.example.test", "engineering", now); err == nil {
				t.Fatal("invalid Prosody contract accepted")
			}
		})
	}
}

func verifyProsodyContract(token string, secret []byte, appID, domain, room string, now time.Time) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("malformed token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return errors.New("signature mismatch")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	var claims struct {
		Issuer   string `json:"iss"`
		Audience string `json:"aud"`
		Subject  string `json:"sub"`
		Room     string `json:"room"`
		Expiry   int64  `json:"exp"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return err
	}
	if claims.Issuer != appID || claims.Audience != appID || claims.Subject != domain ||
		claims.Room != room || claims.Expiry <= now.Unix() {
		return errors.New("claim contract mismatch")
	}
	return nil
}
