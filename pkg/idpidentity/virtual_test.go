package idpidentity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpidentity"
)

func TestVirtualSubjectIsStablePairwiseAndProjectsOnlyProfileClaims(t *testing.T) {
	deriver, err := idpidentity.NewSubjectDeriver([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	first, err := idpidentity.NewVirtual(deriver, idpidentity.VirtualRequest{Namespace: "message-desk", Seed: "verified@example.test", Email: "verified@example.test", EmailVerified: true, DisplayName: "Verified", Roles: []string{"member"}})
	require.NoError(t, err)
	second, err := idpidentity.NewVirtual(deriver, idpidentity.VirtualRequest{Namespace: "message-desk", Seed: "verified@example.test"})
	require.NoError(t, err)
	different, err := idpidentity.NewVirtual(deriver, idpidentity.VirtualRequest{Namespace: "other-app", Seed: "verified@example.test"})
	require.NoError(t, err)
	assert.Equal(t, first.Subject, second.Subject)
	assert.NotEqual(t, first.Subject, different.Subject)
	assert.NotContains(t, first.Subject, "verified@example.test")
	claims := first.ProfileClaims()
	assert.Equal(t, "Verified", claims["name"])
	assert.Equal(t, "verified@example.test", claims["email"])
	assert.NotContains(t, claims, "sub")
	assert.NotContains(t, claims, "iss")
	assert.NotContains(t, claims, "aud")
}

func TestVirtualSubjectRejectsShortKeyAndEmptySeed(t *testing.T) {
	_, err := idpidentity.NewSubjectDeriver([]byte("too short"))
	require.Error(t, err)
	deriver, err := idpidentity.NewSubjectDeriver([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	_, err = idpidentity.NewVirtual(deriver, idpidentity.VirtualRequest{Namespace: "message-desk"})
	require.Error(t, err)
}
