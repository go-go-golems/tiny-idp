package fositeadapter

import (
	"context"
	"reflect"
	"testing"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/ory/fosite"
)

func TestClientWithLifespansAppliesPerClientTokenTTLs(t *testing.T) {
	client := idpstore.Client{AccessTokenTTL: 17 * time.Minute, IDTokenTTL: 23 * time.Minute, RefreshTokenTTL: 37 * time.Hour}
	wrapped := clientWithLifespans(&fosite.DefaultClient{ID: "client"}, client)
	tests := []struct {
		grant fosite.GrantType
		token fosite.TokenType
		want  time.Duration
	}{
		{fosite.GrantTypeAuthorizationCode, fosite.AccessToken, client.AccessTokenTTL},
		{fosite.GrantTypeAuthorizationCode, fosite.IDToken, client.IDTokenTTL},
		{fosite.GrantTypeAuthorizationCode, fosite.RefreshToken, client.RefreshTokenTTL},
		{fosite.GrantTypeRefreshToken, fosite.AccessToken, client.AccessTokenTTL},
		{fosite.GrantTypeRefreshToken, fosite.IDToken, client.IDTokenTTL},
		{fosite.GrantTypeRefreshToken, fosite.RefreshToken, client.RefreshTokenTTL},
	}
	for _, test := range tests {
		if got := fosite.GetEffectiveLifespan(wrapped, test.grant, test.token, time.Second); got != test.want {
			t.Fatalf("%s/%s lifespan = %s, want %s", test.grant, test.token, got, test.want)
		}
	}
}

func TestFositeClientPreservesExplicitGrantCapabilities(t *testing.T) {
	configured := idpstore.Client{
		ID:                "device-cli",
		Public:            true,
		AllowedGrantTypes: []string{idpstore.GrantDeviceCode},
	}
	client, err := (&sqlFositeStore{}).toFositeClient(context.Background(), configured)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string(client.GetGrantTypes()); !reflect.DeepEqual(got, configured.AllowedGrantTypes) {
		t.Fatalf("Fosite grant types = %#v, want %#v", got, configured.AllowedGrantTypes)
	}
}

func TestPersistedRequesterPreservesPerClientTokenTTLs(t *testing.T) {
	client := idpstore.Client{AccessTokenTTL: 17 * time.Minute, IDTokenTTL: 23 * time.Minute, RefreshTokenTTL: 37 * time.Hour}
	wrapped := clientWithLifespans(&fosite.DefaultClient{ID: "client"}, client)
	encoded, err := persistRequester(&fosite.Request{ID: "request", RequestedAt: time.Now(), Client: wrapped})
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restoreRequester(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got := fosite.GetEffectiveLifespan(restored.GetClient(), fosite.GrantTypeRefreshToken, fosite.RefreshToken, time.Second); got != client.RefreshTokenTTL {
		t.Fatalf("restored refresh-token lifespan = %s, want %s", got, client.RefreshTokenTTL)
	}
}
