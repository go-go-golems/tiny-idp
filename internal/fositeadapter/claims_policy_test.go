package fositeadapter

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/internal/oidcmeta"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestClaimsPolicyPersistsOnlyAdditionalClaimsInOIDCSession(t *testing.T) {
	issuer, err := oidcmeta.ParseIssuer("https://issuer.example.test")
	require.NoError(t, err)
	provider := &Provider{issuer: issuer, store: memory.New(), clock: time.Now, claims: staticClaimsPolicy{output: idp.ClaimsOutput{Additional: map[string]json.RawMessage{"community_role": json.RawMessage(`"member"`)}}}}
	request := &fosite.Request{Client: clientWithLifespans(&fosite.DefaultClient{ID: "spa", Public: true}, idpstore.Client{ID: "spa", Public: true}), GrantedScope: fosite.Arguments{"openid", "profile"}, Form: url.Values{"nonce": {"nonce"}}}
	session, err := provider.newOIDCSession(context.Background(), idpstore.User{Sub: "user-ada", Name: "Ada"}, request, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "user-ada", session.Claims.Subject)
	assert.Equal(t, "Ada", session.Claims.Extra["name"])
	assert.Equal(t, "member", session.Claims.Extra["community_role"])

	provider.claims = staticClaimsPolicy{output: idp.ClaimsOutput{Additional: map[string]json.RawMessage{"sub": json.RawMessage(`"other"`)}}}
	_, err = provider.newOIDCSession(context.Background(), idpstore.User{Sub: "user-ada"}, request, time.Now())
	require.Error(t, err)
}

type staticClaimsPolicy struct{ output idp.ClaimsOutput }

func (p staticClaimsPolicy) Claims(context.Context, idp.ClaimsInput) (idp.ClaimsOutput, error) {
	return p.output, nil
}
