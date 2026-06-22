// Package user provides synthetic user derivation for the mock IdP.
//
// A user is derived deterministically from a typed login so that logging in
// as the same string always yields the same OIDC subject (sub), without any
// persistent storage. This is what lets the mock IdP model "different
// authenticated principals" without an account database.
package user

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// User is a synthetic OIDC principal.
type User struct {
	Sub   string
	Email string
	Name  string
}

// Normalize lowercases and trims a login so "Alice", " alice ", and "alice"
// all resolve to the same subject.
func Normalize(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

// FromLogin derives a stable synthetic user from any typed login.
//
//	sub = "user-" + base64url(sha256("tinyidp:user:" + normalize(login))[:16])
//
// The salt prefix prevents trivially guessing other logins' subs and keeps
// the sub from being the raw login. email defaults to <login>@example.test
// when no domain is given; name is the local part of the login.
func FromLogin(login string) User {
	login = Normalize(login)

	sum := sha256.Sum256([]byte("tinyidp:user:" + login))
	sub := "user-" + base64.RawURLEncoding.EncodeToString(sum[:16])

	email := login
	if !strings.Contains(email, "@") {
		email = login + "@example.test"
	}

	name := login
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}

	return User{Sub: sub, Email: email, Name: name}
}
