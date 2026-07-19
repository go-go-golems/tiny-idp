package idpinvite_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
)

func TestSignedInvitationVerifiesAllHostConstraints(t *testing.T) {
	ring, err := idpinvite.NewKeyRing(map[string][]byte{"current": []byte("0123456789abcdef0123456789abcdef")})
	require.NoError(t, err)
	now := time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC)
	token, err := ring.Sign("current", idpinvite.SignedClaims{ID: "invite-1", Issuer: "message-desk", Audience: "message-app", PolicyVersion: "v3", Email: "member@example.test", ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	verified, err := ring.Verify(token, idpinvite.Verification{Issuer: "message-desk", Audience: "message-app", PolicyVersion: "v3", Email: "MEMBER@example.test", Now: now})
	require.NoError(t, err)
	assert.Equal(t, "invite-1", verified.ID)
	_, err = ring.Verify(token, idpinvite.Verification{Issuer: "message-desk", Audience: "other-app", PolicyVersion: "v3", Now: now})
	require.Error(t, err)
	_, err = ring.Verify(token, idpinvite.Verification{Issuer: "message-desk", Audience: "message-app", PolicyVersion: "v3", Now: now.Add(2 * time.Hour)})
	require.Error(t, err)
}

func TestSignedInvitationRejectsTamperingAndKeyRemoval(t *testing.T) {
	ring, err := idpinvite.NewKeyRing(map[string][]byte{"current": []byte("0123456789abcdef0123456789abcdef")})
	require.NoError(t, err)
	now := time.Now().UTC()
	token, err := ring.Sign("current", idpinvite.SignedClaims{ID: "invite-1", Issuer: "issuer", Audience: "app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	_, err = ring.Verify(token+"x", idpinvite.Verification{Issuer: "issuer", Audience: "app", PolicyVersion: "v1", Now: now})
	require.Error(t, err)
	rotated, err := idpinvite.NewKeyRing(map[string][]byte{"next": []byte("abcdefghijklmnopqrstuvwxyz012345")})
	require.NoError(t, err)
	_, err = rotated.Verify(token, idpinvite.Verification{Issuer: "issuer", Audience: "app", PolicyVersion: "v1", Now: now})
	require.Error(t, err)
}
