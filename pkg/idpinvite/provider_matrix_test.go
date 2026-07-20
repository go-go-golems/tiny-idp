package idpinvite_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idpidentity"
	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
)

// TestProviderOutcomeMatrix keeps the security-relevant provider outcomes in
// one reviewable table. Package-specific tests cover wire encoding and runtime
// ownership in greater depth; this matrix verifies the cross-provider contract
// visible to the signup integration.
func TestProviderOutcomeMatrix(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	ring, err := idpinvite.NewKeyRing(map[string][]byte{"current": []byte("0123456789abcdef0123456789abcdef")})
	require.NoError(t, err)
	signed, err := ring.Sign("current", idpinvite.SignedClaims{ID: "signed-1", Issuer: "desk", Audience: "message-app", PolicyVersion: "v1", Email: "member@example.test", ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	durable, err := idpinvite.NewDurableService(memory.New(), []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	require.NoError(t, durable.Issue(ctx, idpinvite.DurableIssue{ID: "durable-1", Code: "one-time", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, durable.Issue(ctx, idpinvite.DurableIssue{ID: "durable-revoked", Code: "revoked", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, durable.Revoke(ctx, "revoked", now))
	capability, err := idpinvite.NewEligibilityCapability(func(_ context.Context, probe idpinvite.EligibilityProbe) (idpinvite.EligibilityDecision, error) {
		if probe.Email == "failure@example.test" {
			return idpinvite.EligibilityDecision{}, errors.New("directory unavailable")
		}
		return idpinvite.EligibilityDecision{Accepted: probe.Email == "member@example.test", EvidenceID: "directory:42"}, nil
	})
	require.NoError(t, err)
	deriver, err := idpidentity.NewSubjectDeriver([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	cases := []struct {
		name      string
		run       func() error
		wantError bool
	}{
		{name: "signed accepted", run: func() error {
			_, err := ring.Verify(signed, idpinvite.Verification{Issuer: "desk", Audience: "message-app", PolicyVersion: "v1", Email: "member@example.test", Now: now})
			return err
		}},
		{name: "signed expired", run: func() error {
			_, err := ring.Verify(signed, idpinvite.Verification{Issuer: "desk", Audience: "message-app", PolicyVersion: "v1", Now: now.Add(2 * time.Hour)})
			return err
		}, wantError: true},
		{name: "durable accepted", run: func() error {
			_, err := durable.Redeem(ctx, "one-time", "message-app", now)
			return err
		}},
		{name: "durable replay", run: func() error {
			_, err := durable.Redeem(ctx, "one-time", "message-app", now)
			return err
		}, wantError: true},
		{name: "durable revoked", run: func() error {
			_, err := durable.Redeem(ctx, "revoked", "message-app", now)
			return err
		}, wantError: true},
		{name: "computed accepted", run: func() error {
			_, err := capability.Invoke(ctx, json.RawMessage(`{"email":"member@example.test","audience":"message-app"}`))
			return err
		}},
		{name: "computed denied", run: func() error {
			result, err := capability.Invoke(ctx, json.RawMessage(`{"email":"other@example.test","audience":"message-app"}`))
			if err != nil {
				return err
			}
			var decision idpinvite.EligibilityDecision
			if err := json.Unmarshal(result, &decision); err != nil {
				return err
			}
			if decision.Accepted {
				return errors.New("computed provider accepted denied subject")
			}
			return nil
		}},
		{name: "computed malformed input", run: func() error {
			_, err := capability.Invoke(ctx, json.RawMessage(`{"email":"member@example.test","audience":"message-app","database":"forged"}`))
			return err
		}, wantError: true},
		{name: "computed capability failure", run: func() error {
			_, err := capability.Invoke(ctx, json.RawMessage(`{"email":"failure@example.test","audience":"message-app"}`))
			return err
		}, wantError: true},
		{name: "virtual identity protects protocol claims", run: func() error {
			candidate, err := idpidentity.NewVirtual(deriver, idpidentity.VirtualRequest{Namespace: "message-app", Seed: "verified-subject", Email: "member@example.test", EmailVerified: true})
			if err != nil {
				return err
			}
			claims := candidate.ProfileClaims()
			if candidate.Subject == "verified-subject" || claims["sub"] != nil || claims["iss"] != nil || claims["aud"] != nil {
				return errors.New("virtual identity leaked or overrode a protocol claim")
			}
			return nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if tc.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("durable one-time redemption is atomic", func(t *testing.T) {
		require.NoError(t, durable.Issue(ctx, idpinvite.DurableIssue{ID: "durable-race", Code: "race", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))
		var successful int
		var lock sync.Mutex
		var group sync.WaitGroup
		for range 12 {
			group.Add(1)
			go func() {
				defer group.Done()
				if _, err := durable.Redeem(ctx, "race", "message-app", now); err == nil {
					lock.Lock()
					successful++
					lock.Unlock()
				}
			}()
		}
		group.Wait()
		assert.Equal(t, 1, successful)
	})
}
