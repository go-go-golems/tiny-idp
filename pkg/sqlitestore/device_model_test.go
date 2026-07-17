package sqlitestore_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

// TestDeviceGrantGeneratedActionSequencesAgreeWithReferenceModel is a small
// stateful model-checking harness. It generates sequences against an
// independently implemented, pure transition model and compares each observed
// SQLite result. The model deliberately uses only the public named operations;
// it cannot reach into the SQLite tables or rely on implementation internals.
func TestDeviceGrantGeneratedActionSequencesAgreeWithReferenceModel(t *testing.T) {
	for seed := int64(0); seed < 64; seed++ {
		t.Run(fmt.Sprintf("seed-%02d", seed), func(t *testing.T) {
			ctx := context.Background()
			base := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
			grant := modelGrant(base)
			store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err := store.PutClient(ctx, idpstore.Client{ID: grant.ClientID}); err != nil {
				t.Fatal(err)
			}
			if err := store.CreateDeviceGrant(ctx, grant); err != nil {
				t.Fatal(err)
			}
			// CreateDeviceGrant assigns the initial persisted version. The pure
			// model starts from the same post-create public state.
			grant.Version = 1
			model := newDeviceGrantModel(grant)
			rng := rand.New(rand.NewSource(seed))
			now := base
			for step := 0; step < 48; step++ {
				now = now.Add(time.Duration(rng.Intn(5)) * time.Second)
				switch rng.Intn(5) {
				case 0, 1:
					clientID := grant.ClientID
					if rng.Intn(5) == 0 {
						clientID = "wrong-client"
					}
					request := idpstore.DevicePollRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: clientID, Now: now}
					want, wantErr := model.poll(request)
					got, gotErr := store.PollDeviceGrant(ctx, request)
					assertModelError(t, seed, step, "poll", gotErr, wantErr)
					if gotErr == nil {
						assertModelPoll(t, seed, step, got, want)
					}
				case 2:
					decision := idpstore.DeviceGrantApprove
					if rng.Intn(3) == 0 {
						decision = idpstore.DeviceGrantDeny
					}
					request := idpstore.DeviceDecisionRequest{UserCodeHash: grant.UserCodeHash, Decision: decision, Now: now}
					if decision == idpstore.DeviceGrantApprove {
						request.UserID = "u1"
						request.Subject = "subject-1"
						request.AuthTime = now
						request.AuthenticationMethods = []string{"pwd"}
						request.ApprovedScopes = []string{"openid"}
					}
					want, wantErr := model.decide(request)
					got, gotErr := store.DecideDeviceGrant(ctx, request)
					assertModelError(t, seed, step, "decide", gotErr, wantErr)
					if gotErr == nil {
						assertModelGrant(t, seed, step, got, want)
					}
				default:
					clientID := grant.ClientID
					if rng.Intn(5) == 0 {
						clientID = "wrong-client"
					}
					request := idpstore.DeviceConsumeRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: clientID, Now: now}
					want, wantErr := model.consume(request)
					got, gotErr := store.ConsumeDeviceGrant(ctx, request)
					assertModelError(t, seed, step, "consume", gotErr, wantErr)
					if gotErr == nil {
						assertModelGrant(t, seed, step, got, want)
					}
				}
			}
		})
	}
}

type deviceGrantModel struct{ grant idpstore.DeviceGrant }

func newDeviceGrantModel(grant idpstore.DeviceGrant) *deviceGrantModel {
	return &deviceGrantModel{grant: cloneModelGrant(grant)}
}

func (m *deviceGrantModel) poll(request idpstore.DevicePollRequest) (idpstore.DevicePollResult, error) {
	if request.ClientID != m.grant.ClientID || string(request.DeviceCodeHash) != string(m.grant.DeviceCodeHash) {
		return idpstore.DevicePollResult{}, idpstore.ErrNotFound
	}
	now := request.Now.UTC()
	if !now.Before(m.grant.ExpiresAt) {
		return idpstore.DevicePollResult{Outcome: idpstore.DevicePollExpired, Grant: cloneModelGrant(m.grant)}, nil
	}
	switch m.grant.Status {
	case idpstore.DeviceGrantDenied:
		return idpstore.DevicePollResult{Outcome: idpstore.DevicePollDenied, Grant: cloneModelGrant(m.grant)}, nil
	case idpstore.DeviceGrantConsumed:
		return idpstore.DevicePollResult{Outcome: idpstore.DevicePollConsumed, Grant: cloneModelGrant(m.grant)}, nil
	case idpstore.DeviceGrantPending, idpstore.DeviceGrantApproved:
	default:
		return idpstore.DevicePollResult{}, idpstore.ErrInvalidDeviceGrant
	}
	if now.Before(m.grant.NextPollAt) {
		m.grant.PollInterval += 5 * time.Second
		m.grant.NextPollAt = now.Add(m.grant.PollInterval)
		m.grant.SlowDownCount++
		m.grant.Version++
		return idpstore.DevicePollResult{Outcome: idpstore.DevicePollSlowDown, Grant: cloneModelGrant(m.grant)}, nil
	}
	if m.grant.Status == idpstore.DeviceGrantApproved {
		return idpstore.DevicePollResult{Outcome: idpstore.DevicePollApproved, Grant: cloneModelGrant(m.grant)}, nil
	}
	m.grant.NextPollAt = now.Add(m.grant.PollInterval)
	m.grant.Version++
	return idpstore.DevicePollResult{Outcome: idpstore.DevicePollPending, Grant: cloneModelGrant(m.grant)}, nil
}

func (m *deviceGrantModel) decide(request idpstore.DeviceDecisionRequest) (idpstore.DeviceGrant, error) {
	if !request.Decision.Valid() || request.Now.IsZero() {
		return idpstore.DeviceGrant{}, idpstore.ErrInvalidDeviceDecision
	}
	if string(request.UserCodeHash) != string(m.grant.UserCodeHash) {
		return idpstore.DeviceGrant{}, idpstore.ErrNotFound
	}
	now := request.Now.UTC()
	if !now.Before(m.grant.ExpiresAt) {
		return idpstore.DeviceGrant{}, idpstore.ErrExpired
	}
	if m.grant.Status != idpstore.DeviceGrantPending {
		return idpstore.DeviceGrant{}, idpstore.ErrDeviceGrantNotPending
	}
	if request.Decision == idpstore.DeviceGrantApprove && (request.UserID == "" || request.Subject == "" || request.AuthTime.IsZero()) {
		return idpstore.DeviceGrant{}, idpstore.ErrInvalidDeviceDecision
	}
	m.grant.Status = idpstore.DeviceGrantDenied
	m.grant.DecidedAt = &now
	if request.Decision == idpstore.DeviceGrantApprove {
		m.grant.Status = idpstore.DeviceGrantApproved
		m.grant.UserID, m.grant.Subject, m.grant.AuthTime = request.UserID, request.Subject, request.AuthTime.UTC()
		m.grant.AuthenticationMethods = append([]string(nil), request.AuthenticationMethods...)
		m.grant.ApprovedScopes = append([]string(nil), request.ApprovedScopes...)
		m.grant.ApprovedAudiences = append([]string(nil), request.ApprovedAudiences...)
	}
	m.grant.Version++
	return cloneModelGrant(m.grant), nil
}

func (m *deviceGrantModel) consume(request idpstore.DeviceConsumeRequest) (idpstore.DeviceGrant, error) {
	if request.ClientID != m.grant.ClientID || string(request.DeviceCodeHash) != string(m.grant.DeviceCodeHash) {
		return idpstore.DeviceGrant{}, idpstore.ErrNotFound
	}
	now := request.Now.UTC()
	if !now.Before(m.grant.ExpiresAt) {
		return idpstore.DeviceGrant{}, idpstore.ErrExpired
	}
	if m.grant.Status == idpstore.DeviceGrantConsumed {
		return idpstore.DeviceGrant{}, idpstore.ErrAlreadyConsumed
	}
	if m.grant.Status != idpstore.DeviceGrantApproved {
		return idpstore.DeviceGrant{}, idpstore.ErrDeviceGrantNotApproved
	}
	m.grant.Status = idpstore.DeviceGrantConsumed
	m.grant.ConsumedAt = &now
	m.grant.Version++
	return cloneModelGrant(m.grant), nil
}

func modelGrant(now time.Time) idpstore.DeviceGrant {
	return idpstore.DeviceGrant{ID: "model-grant", DeviceCodeHash: []byte("device-hash"), UserCodeHash: []byte("user-hash"), ClientID: "device-client", RequestedScopes: []string{"openid"}, Status: idpstore.DeviceGrantPending, CreatedAt: now, ExpiresAt: now.Add(30 * time.Second), PollInterval: 5 * time.Second, NextPollAt: now}
}

func cloneModelGrant(in idpstore.DeviceGrant) idpstore.DeviceGrant {
	out := in
	out.DeviceCodeHash = append([]byte(nil), in.DeviceCodeHash...)
	out.UserCodeHash = append([]byte(nil), in.UserCodeHash...)
	out.RequestedScopes = append([]string(nil), in.RequestedScopes...)
	out.AuthenticationMethods = append([]string(nil), in.AuthenticationMethods...)
	out.ApprovedScopes = append([]string(nil), in.ApprovedScopes...)
	out.ApprovedAudiences = append([]string(nil), in.ApprovedAudiences...)
	if in.DecidedAt != nil {
		value := *in.DecidedAt
		out.DecidedAt = &value
	}
	if in.ConsumedAt != nil {
		value := *in.ConsumedAt
		out.ConsumedAt = &value
	}
	return out
}

func assertModelError(t *testing.T, seed int64, step int, operation string, got, want error) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if want == nil || got == nil || !errors.Is(got, want) {
		t.Fatalf("seed=%d step=%d %s error=%v, want %v", seed, step, operation, got, want)
	}
}

func assertModelPoll(t *testing.T, seed int64, step int, got, want idpstore.DevicePollResult) {
	t.Helper()
	if got.Outcome != want.Outcome {
		t.Fatalf("seed=%d step=%d poll outcome=%s, want %s", seed, step, got.Outcome, want.Outcome)
	}
	assertModelGrant(t, seed, step, got.Grant, want.Grant)
}

func assertModelGrant(t *testing.T, seed int64, step int, got, want idpstore.DeviceGrant) {
	t.Helper()
	if got.Status != want.Status || got.Version != want.Version || got.PollInterval != want.PollInterval || !got.NextPollAt.Equal(want.NextPollAt) || got.SlowDownCount != want.SlowDownCount || !sameTime(got.DecidedAt, want.DecidedAt) || !sameTime(got.ConsumedAt, want.ConsumedAt) {
		t.Fatalf("seed=%d step=%d grant=%#v, want %#v", seed, step, got, want)
	}
}

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}
