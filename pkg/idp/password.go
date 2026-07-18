package idp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var ErrPasswordRejected = errors.New("password rejected by acceptance policy")

// PasswordBlocklist rejects a complete prospective password using common,
// compromised, or deployment-specific values. Implementations must not log or
// retain the supplied password.
type PasswordBlocklist interface {
	Blocked(ctx context.Context, normalizedPassword []byte, contextWords []string) (bool, error)
}

// PasswordAcceptancePolicy is independent of Argon2id encoding parameters.
// Character limits apply after NFC normalization; MaxBytes bounds hashing work.
type PasswordAcceptancePolicy struct {
	MinCharacters int
	MaxCharacters int
	MaxBytes      int
	Blocklist     PasswordBlocklist
}

// DefaultPasswordAcceptancePolicy follows the single-factor password length
// guidance in NIST SP 800-63B-4 and supplies a small baseline blocklist.
func DefaultPasswordAcceptancePolicy() PasswordAcceptancePolicy {
	return PasswordAcceptancePolicy{
		MinCharacters: 15,
		MaxCharacters: 1024,
		MaxBytes:      4096,
		Blocklist:     NewStaticPasswordBlocklist(defaultBlockedPasswords),
	}
}

// DevelopmentPasswordAcceptancePolicy preserves short scenario passwords for
// explicit development/test wiring. Production validation rejects it.
func DevelopmentPasswordAcceptancePolicy() PasswordAcceptancePolicy {
	return PasswordAcceptancePolicy{
		MinCharacters: 1,
		MaxCharacters: 1024,
		MaxBytes:      4096,
		Blocklist:     NewStaticPasswordBlocklist(nil),
	}
}

// NormalizeAndValidatePassword returns NFC-normalized bytes suitable for
// hashing or a non-secret rejection reason.
func (p PasswordAcceptancePolicy) NormalizeAndValidatePassword(ctx context.Context, password []byte, contextWords ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.MinCharacters <= 0 || p.MaxCharacters < p.MinCharacters || p.MaxBytes < p.MaxCharacters || p.Blocklist == nil {
		return nil, fmt.Errorf("%w: invalid_policy", ErrPasswordRejected)
	}
	if !utf8.Valid(password) {
		return nil, fmt.Errorf("%w: invalid_utf8", ErrPasswordRejected)
	}
	normalized := []byte(norm.NFC.String(string(password)))
	characters := utf8.RuneCount(normalized)
	if characters < p.MinCharacters {
		return nil, fmt.Errorf("%w: too_short", ErrPasswordRejected)
	}
	if characters > p.MaxCharacters || len(normalized) > p.MaxBytes {
		return nil, fmt.Errorf("%w: too_long", ErrPasswordRejected)
	}
	blocked, err := p.Blocklist.Blocked(ctx, normalized, contextWords)
	if err != nil {
		return nil, fmt.Errorf("password blocklist: %w", err)
	}
	if blocked {
		return nil, fmt.Errorf("%w: blocklisted", ErrPasswordRejected)
	}
	return normalized, nil
}

// NormalizePassword applies the same NFC transformation used at password
// establishment. It rejects invalid UTF-8 and oversized inputs before Argon2.
func (p PasswordAcceptancePolicy) NormalizePassword(password []byte) ([]byte, error) {
	if !utf8.Valid(password) {
		return nil, fmt.Errorf("%w: invalid_utf8", ErrPasswordRejected)
	}
	normalized := []byte(norm.NFC.String(string(password)))
	if p.MaxBytes <= 0 || len(normalized) > p.MaxBytes {
		return nil, fmt.Errorf("%w: too_long", ErrPasswordRejected)
	}
	return normalized, nil
}

// StaticPasswordBlocklist is an in-memory exact/context-derived blocklist.
type StaticPasswordBlocklist struct {
	values map[string]struct{}
}

func NewStaticPasswordBlocklist(values []string) *StaticPasswordBlocklist {
	list := &StaticPasswordBlocklist{values: make(map[string]struct{}, len(values))}
	for _, value := range values {
		value = canonicalPasswordWord(value)
		if value != "" {
			list.values[value] = struct{}{}
		}
	}
	return list
}

func (l *StaticPasswordBlocklist) Blocked(ctx context.Context, password []byte, contextWords []string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if l == nil {
		return false, fmt.Errorf("nil blocklist")
	}
	value := canonicalPasswordWord(string(password))
	if _, ok := l.values[value]; ok {
		return true, nil
	}
	base := strings.TrimRightFunc(value, func(r rune) bool {
		return unicode.IsDigit(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})
	for _, word := range contextWords {
		word = canonicalPasswordWord(word)
		if word != "" && (value == word || base == word) {
			return true, nil
		}
	}
	return false, nil
}

func canonicalPasswordWord(value string) string {
	return strings.ToLower(norm.NFC.String(strings.TrimSpace(value)))
}

var defaultBlockedPasswords = []string{
	"password", "password1", "password123", "123456789012345", "qwertyuiopasdfg",
	"letmeinletmeinlet", "administrator", "welcome123456789", "correcthorsebatterystaple",
	"iloveyouiloveyou", "monkeymonkeymon", "dragon123456789", "football12345678",
	"baseball12345678", "sunshinesunshine", "princessprincess", "trustno1trustno1",
	"changemechangeme", "temporarypassword", "tinyidptinyidpidp",
}

// PasswordWorkConfig bounds expensive password hashing and verification.
type PasswordWorkConfig struct {
	MaxConcurrent int
}

func DefaultPasswordWorkConfig() PasswordWorkConfig {
	return PasswordWorkConfig{MaxConcurrent: 2}
}

// PasswordWorkStats is a cumulative, non-secret snapshot of Argon2 capacity.
type PasswordWorkStats struct {
	Capacity      int
	InFlight      int64
	Waiting       int64
	Saturations   uint64
	Rejected      uint64
	Completed     uint64
	TotalWait     int64
	TotalDuration int64
}

// PasswordWorkReporter exposes Argon2 capacity metrics in nanoseconds.
type PasswordWorkReporter interface {
	PasswordWorkStats() PasswordWorkStats
}

// ProductionReadyReporter lets injected security controls declare whether
// their configuration is suitable for production construction.
type ProductionReadyReporter interface {
	ProductionReady() bool
}
