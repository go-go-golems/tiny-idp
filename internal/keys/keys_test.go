package keys

import (
	"strings"
	"testing"
	"time"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestPublicJWKSContainsPublicFields(t *testing.T) {
	key, err := GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	jwks, err := PublicJWKS([]idpstore.SigningKey{key})
	if err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0].Kid != "kid-1" || jwks.Keys[0].Alg != "RS256" {
		t.Fatalf("bad jwks: %#v", jwks)
	}
	if strings.Contains(jwks.Keys[0].N, "PRIVATE") || strings.Contains(jwks.Keys[0].E, "PRIVATE") {
		t.Fatalf("jwks leaked private material: %#v", jwks.Keys[0])
	}
}
