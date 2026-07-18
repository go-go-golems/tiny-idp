package idpstore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// RunStoreSuite verifies invariants every store implementation must satisfy.
func RunStoreSuite(t *testing.T, newStore func(t *testing.T) Store) {
	t.Helper()
	t.Run("device grants use named durable transitions", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Date(2026, time.July, 15, 14, 0, 0, 0, time.UTC)
		if err := st.PutClient(ctx, Client{ID: "device-client"}); err != nil {
			t.Fatal(err)
		}
		grant := DeviceGrant{
			ID: "grant-1", DeviceCodeHash: []byte("device-code-hash"), UserCodeHash: []byte("user-code-hash"), ClientID: "device-client",
			RequestedScopes: []string{"openid", "profile"}, RequestedAudiences: []string{"https://api.example.test/messages"}, Status: DeviceGrantPending,
			CreatedAt: now, ExpiresAt: now.Add(time.Minute), PollInterval: 5 * time.Second, NextPollAt: now,
		}
		if err := st.CreateDeviceGrant(ctx, grant); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateDeviceGrant(ctx, grant); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("duplicate device grant error = %v", err)
		}
		if _, err := st.InspectDeviceGrantByDeviceCodeHash(ctx, grant.DeviceCodeHash, "wrong-client"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("wrong-client inspect error = %v", err)
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := st.PollDeviceGrant(canceled, DevicePollRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now}); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled poll error = %v", err)
		}
		pending, err := st.PollDeviceGrant(ctx, DevicePollRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now})
		if err != nil || pending.Outcome != DevicePollPending || !pending.Grant.NextPollAt.Equal(now.Add(5*time.Second)) {
			t.Fatalf("first poll = %#v, %v", pending, err)
		}
		slow, err := st.PollDeviceGrant(ctx, DevicePollRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(time.Second)})
		if err != nil || slow.Outcome != DevicePollSlowDown || slow.Grant.PollInterval != 10*time.Second || slow.Grant.SlowDownCount != 1 {
			t.Fatalf("early poll = %#v, %v", slow, err)
		}
		approved, err := st.DecideDeviceGrant(ctx, DeviceDecisionRequest{UserCodeHash: grant.UserCodeHash, Decision: DeviceGrantApprove, UserID: "u1", Subject: "subject-1", AuthTime: now.Add(2 * time.Second), AuthenticationMethods: []string{"pwd"}, ApprovedScopes: []string{"openid"}, ApprovedAudiences: []string{"https://api.example.test/messages"}, Now: now.Add(2 * time.Second)})
		if err != nil || approved.Status != DeviceGrantApproved || approved.DecidedAt == nil || approved.UserID != "u1" || len(approved.RequestedAudiences) != 1 || len(approved.ApprovedAudiences) != 1 {
			t.Fatalf("approve = %#v, %v", approved, err)
		}
		earlyApproved, err := st.PollDeviceGrant(ctx, DevicePollRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(3 * time.Second)})
		if err != nil || earlyApproved.Outcome != DevicePollSlowDown || earlyApproved.Grant.PollInterval != 15*time.Second {
			t.Fatalf("early approved poll = %#v, %v", earlyApproved, err)
		}
		ready, err := st.PollDeviceGrant(ctx, DevicePollRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(20 * time.Second)})
		if err != nil || ready.Outcome != DevicePollApproved {
			t.Fatalf("approved poll = %#v, %v", ready, err)
		}
		if err := st.Update(ctx, func(tx TxStore) error {
			if _, err := tx.ConsumeDeviceGrant(ctx, DeviceConsumeRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(21 * time.Second)}); err != nil {
				return err
			}
			return errors.New("rollback device consumption")
		}); err == nil {
			t.Fatal("rollback transaction returned nil")
		}
		afterRollback, err := st.InspectDeviceGrantByDeviceCodeHash(ctx, grant.DeviceCodeHash, grant.ClientID)
		if err != nil || afterRollback.Status != DeviceGrantApproved {
			t.Fatalf("rollback left grant = %#v, %v", afterRollback, err)
		}
		consumed, err := st.ConsumeDeviceGrant(ctx, DeviceConsumeRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(21 * time.Second)})
		if err != nil || consumed.Status != DeviceGrantConsumed || consumed.ConsumedAt == nil {
			t.Fatalf("consume = %#v, %v", consumed, err)
		}
		if _, err := st.ConsumeDeviceGrant(ctx, DeviceConsumeRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(22 * time.Second)}); !errors.Is(err, ErrAlreadyConsumed) {
			t.Fatalf("replay consume error = %v", err)
		}
	})
	t.Run("device grants expire before decision and consumption", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Date(2026, time.July, 15, 14, 0, 0, 0, time.UTC)
		if err := st.PutClient(ctx, Client{ID: "device-client"}); err != nil {
			t.Fatal(err)
		}
		grant := DeviceGrant{ID: "expired", DeviceCodeHash: []byte("expired-device"), UserCodeHash: []byte("expired-user"), ClientID: "device-client", Status: DeviceGrantPending, CreatedAt: now, ExpiresAt: now.Add(time.Second), PollInterval: time.Second, NextPollAt: now}
		if err := st.CreateDeviceGrant(ctx, grant); err != nil {
			t.Fatal(err)
		}
		if result, err := st.PollDeviceGrant(ctx, DevicePollRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(time.Second)}); err != nil || result.Outcome != DevicePollExpired {
			t.Fatalf("expired poll = %#v, %v", result, err)
		}
		if _, err := st.DecideDeviceGrant(ctx, DeviceDecisionRequest{UserCodeHash: grant.UserCodeHash, Decision: DeviceGrantDeny, Now: now.Add(time.Second)}); !errors.Is(err, ErrExpired) {
			t.Fatalf("expired decision error = %v", err)
		}
		if _, err := st.ConsumeDeviceGrant(ctx, DeviceConsumeRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(time.Second)}); !errors.Is(err, ErrExpired) {
			t.Fatalf("expired consume error = %v", err)
		}
	})
	t.Run("maintenance removes expired browser contexts and orphaned remembered sessions", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		maintenance, ok := st.(MaintenanceStore)
		if !ok {
			t.Fatal("store does not implement maintenance")
		}
		now := time.Date(2026, time.July, 14, 18, 0, 0, 0, time.UTC)
		contextHash := []byte("expired-browser-context")
		sessionHash := []byte("expired-source-session")
		if err := st.CreateBrowserContext(ctx, BrowserContext{IDHash: contextHash, CreatedAt: now.Add(-72 * time.Hour), ExpiresAt: now.Add(-48 * time.Hour)}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateSession(ctx, Session{IDHash: sessionHash, UserID: "u1", ExpiresAt: now.Add(-48 * time.Hour)}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateRememberedBrowserSession(ctx, RememberedBrowserSession{IDHash: []byte("orphaned-remembered-entry"), ContextIDHash: contextHash, SessionIDHash: sessionHash, UserID: "u1", CreatedAt: now.Add(-72 * time.Hour), LastUsedAt: now.Add(-72 * time.Hour)}); err != nil {
			t.Fatal(err)
		}
		report, err := maintenance.Maintain(ctx, now, MaintenancePolicy{RetainExpiredFor: 24 * time.Hour, ProtocolStateRetention: 24 * time.Hour, SigningKeyRetention: 24 * time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		if report.DomainRecords < 3 {
			t.Fatalf("maintenance domain records = %d, want at least 3", report.DomainRecords)
		}
		if _, err := st.GetBrowserContext(ctx, contextHash); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expired browser context survived maintenance: %v", err)
		}
		if _, err := st.GetSession(ctx, sessionHash); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expired source session survived maintenance: %v", err)
		}
	})
	t.Run("remembered browser session activation is context-bound and fresh", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Date(2026, time.July, 14, 18, 0, 0, 0, time.UTC)
		contextA := []byte("browser-context-a")
		contextB := []byte("browser-context-b")
		sourceHash := []byte("source-session")
		entryHash := []byte("remembered-entry")
		freshHash := []byte("fresh-active-session")
		if err := st.PutUser(ctx, "alice", User{ID: "u1", Sub: "subject-1", Name: "Alice"}); err != nil {
			t.Fatal(err)
		}
		source := Session{IDHash: sourceHash, UserID: "u1", AuthTime: now.Add(-10 * time.Minute), CreatedAt: now.Add(-10 * time.Minute), LastSeenAt: now.Add(-10 * time.Minute), ExpiresAt: now.Add(time.Hour), ACR: "urn:tinyidp:password", AMR: []string{"pwd"}}
		if err := st.CreateSession(ctx, source); err != nil {
			t.Fatal(err)
		}
		for _, browserContext := range []BrowserContext{
			{IDHash: contextA, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(24 * time.Hour)},
			{IDHash: contextB, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(24 * time.Hour)},
		} {
			if err := st.CreateBrowserContext(ctx, browserContext); err != nil {
				t.Fatal(err)
			}
		}
		remembered := RememberedBrowserSession{IDHash: entryHash, ContextIDHash: contextA, SessionIDHash: sourceHash, UserID: "u1", DisplayLabel: "Alice", CreatedAt: now, LastUsedAt: now}
		if err := st.CreateRememberedBrowserSession(ctx, remembered); err != nil {
			t.Fatal(err)
		}
		entries, err := st.ListRememberedBrowserSessions(ctx, contextA, now)
		if err != nil || len(entries) != 1 || entries[0].DisplayLabel != "Alice" {
			t.Fatalf("list remembered entries = %#v, %v", entries, err)
		}
		entries[0].DisplayLabel = "mutated caller copy"
		again, err := st.ListRememberedBrowserSessions(ctx, contextA, now)
		if err != nil || again[0].DisplayLabel != "Alice" {
			t.Fatalf("remembered entry aliases store state: %#v, %v", again, err)
		}
		if _, _, err := st.ActivateRememberedSession(ctx, contextB, entryHash, []byte("cross-context"), now.Add(time.Second)); !errors.Is(err, ErrNotFound) {
			t.Fatalf("cross-context activation error = %v, want %v", err, ErrNotFound)
		}
		if _, err := st.GetSession(ctx, []byte("cross-context")); !errors.Is(err, ErrNotFound) {
			t.Fatalf("cross-context activation created a session: %v", err)
		}
		active, user, err := st.ActivateRememberedSession(ctx, contextA, entryHash, freshHash, now.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if user.ID != "u1" || string(active.IDHash) != string(freshHash) || !active.AuthTime.Equal(source.AuthTime) || !active.ExpiresAt.Equal(source.ExpiresAt) || active.RevokedAt != nil {
			t.Fatalf("activated session/user = %#v / %#v", active, user)
		}
		if _, err := st.GetSession(ctx, sourceHash); err != nil {
			t.Fatalf("activation replaced source session: %v", err)
		}
		entries, err = st.ListRememberedBrowserSessions(ctx, contextA, now.Add(time.Minute))
		if err != nil || len(entries) != 1 || !entries[0].LastUsedAt.Equal(now.Add(time.Minute)) {
			t.Fatalf("activation did not refresh entry use: %#v, %v", entries, err)
		}
		if err := st.RevokeSession(ctx, sourceHash, now.Add(90*time.Second)); err != nil {
			t.Fatal(err)
		}
		entries, err = st.ListRememberedBrowserSessions(ctx, contextA, now.Add(2*time.Minute))
		if err != nil || len(entries) != 0 {
			t.Fatalf("revoked source session remains selectable: %#v, %v", entries, err)
		}
		if _, _, err := st.ActivateRememberedSession(ctx, contextA, entryHash, []byte("revoked-source"), now.Add(2*time.Minute)); !errors.Is(err, ErrNotFound) {
			t.Fatalf("revoked source activation error = %v, want %v", err, ErrNotFound)
		}
		if err := st.RemoveRememberedBrowserSession(ctx, contextA, entryHash, now.Add(2*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := st.ActivateRememberedSession(ctx, contextA, entryHash, []byte("removed-entry"), now.Add(2*time.Minute)); !errors.Is(err, ErrNotFound) {
			t.Fatalf("removed entry activation error = %v, want %v", err, ErrNotFound)
		}
		if err := st.RevokeBrowserContext(ctx, contextA, now.Add(3*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if _, err := st.ListRememberedBrowserSessions(ctx, contextA, now.Add(3*time.Minute)); !errors.Is(err, ErrNotFound) {
			t.Fatalf("revoked context list error = %v, want %v", err, ErrNotFound)
		}
	})
	t.Run("nested transactions are rejected", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		err := st.Update(ctx, func(tx TxStore) error {
			nested, ok := tx.(interface {
				Update(context.Context, func(TxStore) error) error
			})
			if !ok {
				t.Fatal("transaction implementation does not expose its nested-operation guard")
			}
			return nested.Update(ctx, func(TxStore) error { return nil })
		})
		if !errors.Is(err, ErrNestedTransaction) {
			t.Fatalf("nested Update error = %v, want %v", err, ErrNestedTransaction)
		}
	})

	t.Run("password security artifact revocation is user scoped", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now().UTC()
		if err := st.PutUser(ctx, "alice", User{ID: "u1", Sub: "subject-1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateGrant(ctx, Grant{ID: "g1", UserID: "u1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateAuthorizationCode(ctx, AuthorizationCode{CodeHash: []byte("user-code"), UserID: "u1", ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateAccessToken(ctx, AccessToken{TokenHash: []byte("user-access"), UserID: "u1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateRefreshToken(ctx, RefreshToken{TokenHash: []byte("user-refresh"), GrantID: "g1", UserID: "u1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateSession(ctx, Session{IDHash: []byte("user-session"), UserID: "u1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.RevokeUserSecurityArtifacts(ctx, "u1", now); err != nil {
			t.Fatal(err)
		}
		grant, _ := st.GetGrant(ctx, "g1")
		access, _ := st.GetAccessToken(ctx, []byte("user-access"))
		refresh, _ := st.GetRefreshToken(ctx, []byte("user-refresh"))
		session, _ := st.GetSession(ctx, []byte("user-session"))
		if grant.RevokedAt == nil || access.RevokedAt == nil || refresh.RevokedAt == nil || session.RevokedAt == nil {
			t.Fatalf("artifacts not revoked: grant=%#v access=%#v refresh=%#v session=%#v", grant, access, refresh, session)
		}
		if _, err := st.ConsumeAuthorizationCode(ctx, []byte("user-code"), now); !errors.Is(err, ErrAlreadyConsumed) {
			t.Fatalf("authorization code after password revocation = %v", err)
		}
	})
	t.Run("OIDC subjects are unique at the store boundary", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		if err := st.PutUser(ctx, "alice", User{ID: "u1", Sub: "subject-1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.PutUser(ctx, "bob", User{ID: "u2", Sub: "subject-1"}); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("duplicate subject error = %v, want %v", err, ErrDuplicate)
		}
		user, err := st.GetUserBySubject(ctx, "subject-1")
		if err != nil || user.ID != "u1" {
			t.Fatalf("GetUserBySubject = %#v, %v", user, err)
		}
	})
	t.Run("authorization code can be consumed once", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		codeHash := []byte("code-1")
		if err := st.CreateAuthorizationCode(ctx, AuthorizationCode{CodeHash: codeHash, ClientID: "c", ExpiresAt: now.Add(time.Minute)}); err != nil {
			t.Fatalf("create code: %v", err)
		}
		if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); err != nil {
			t.Fatalf("consume code: %v", err)
		}
		if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); !errors.Is(err, ErrAlreadyConsumed) {
			t.Fatalf("second consume got %v, want %v", err, ErrAlreadyConsumed)
		}
	})

	t.Run("parallel authorization code consumption has one winner", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		codeHash := []byte("code-race")
		if err := st.CreateAuthorizationCode(ctx, AuthorizationCode{CodeHash: codeHash, ClientID: "c", ExpiresAt: now.Add(time.Minute)}); err != nil {
			t.Fatalf("create code: %v", err)
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		success := 0
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); err == nil {
					mu.Lock()
					success++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if success != 1 {
			t.Fatalf("success count = %d, want 1", success)
		}
	})

	t.Run("expired authorization code is rejected", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		codeHash := []byte("code-expired")
		if err := st.CreateAuthorizationCode(ctx, AuthorizationCode{CodeHash: codeHash, ExpiresAt: now.Add(-time.Minute)}); err != nil {
			t.Fatalf("create code: %v", err)
		}
		if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); !errors.Is(err, ErrExpired) {
			t.Fatalf("got %v, want expired", err)
		}
	})

	t.Run("authorization interaction is isolated and consumed once", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now().UTC()
		idHash := []byte("interaction-1")
		original := InteractionRecord{
			IDHash:           idHash,
			CanonicalRequest: map[string][]string{"state": {"original"}, "scope": {"openid", "email"}},
			RequestDigest:    []byte("digest"),
			ClientID:         "client",
			RequiredActions:  InteractionRequireFreshLogin | InteractionRequireConsent,
			CreatedAt:        now,
			ExpiresAt:        now.Add(time.Minute),
		}
		if err := st.CreateInteraction(ctx, original); err != nil {
			t.Fatalf("create interaction: %v", err)
		}
		original.CanonicalRequest["state"][0] = "mutated-after-create"
		original.RequestDigest[0] = 'X'
		stored, err := st.GetInteraction(ctx, idHash)
		if err != nil {
			t.Fatalf("get interaction: %v", err)
		}
		if stored.CanonicalRequest["state"][0] != "original" || string(stored.RequestDigest) != "digest" {
			t.Fatalf("interaction was not copy-isolated: %#v", stored)
		}
		stored.CanonicalRequest["state"][0] = "mutated-after-get"
		again, err := st.GetInteraction(ctx, idHash)
		if err != nil {
			t.Fatal(err)
		}
		if again.CanonicalRequest["state"][0] != "original" {
			t.Fatalf("returned interaction aliases store state: %#v", again.CanonicalRequest)
		}
		consumed, err := st.ConsumeInteraction(ctx, idHash, now.Add(time.Second), InteractionOutcomeApproved)
		if err != nil {
			t.Fatalf("consume interaction: %v", err)
		}
		if consumed.ConsumedAt == nil || consumed.Outcome != InteractionOutcomeApproved {
			t.Fatalf("bad consumed interaction: %#v", consumed)
		}
		if _, err := st.ConsumeInteraction(ctx, idHash, now.Add(2*time.Second), InteractionOutcomeApproved); !errors.Is(err, ErrAlreadyConsumed) {
			t.Fatalf("second consume got %v, want %v", err, ErrAlreadyConsumed)
		}
	})

	t.Run("authorization interaction expiration and outcomes are enforced", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now().UTC()
		idHash := []byte("interaction-expired")
		if err := st.CreateInteraction(ctx, InteractionRecord{IDHash: idHash, CreatedAt: now.Add(-time.Minute), ExpiresAt: now}); err != nil {
			t.Fatal(err)
		}
		if _, err := st.ConsumeInteraction(ctx, idHash, now, InteractionOutcomeApproved); !errors.Is(err, ErrExpired) {
			t.Fatalf("expired consume got %v, want %v", err, ErrExpired)
		}
		if _, err := st.ConsumeInteraction(ctx, idHash, now.Add(-time.Second), InteractionOutcome("unknown")); !errors.Is(err, ErrInvalidInteractionOutcome) {
			t.Fatalf("invalid outcome got %v, want %v", err, ErrInvalidInteractionOutcome)
		}
	})

	t.Run("parallel authorization interaction consumption has one winner", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now().UTC()
		idHash := []byte("interaction-race")
		if err := st.CreateInteraction(ctx, InteractionRecord{IDHash: idHash, CreatedAt: now, ExpiresAt: now.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		success := 0
		for range 16 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := st.ConsumeInteraction(ctx, idHash, now, InteractionOutcomeApproved); err == nil {
					mu.Lock()
					success++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if success != 1 {
			t.Fatalf("success count = %d, want 1", success)
		}
	})

	t.Run("refresh token rotation and reuse detection", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		oldHash := []byte("refresh-old")
		newHash := []byte("refresh-new")
		if err := st.CreateRefreshToken(ctx, RefreshToken{TokenHash: oldHash, GrantID: "g", ClientID: "c", UserID: "u", ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatalf("create refresh: %v", err)
		}
		if _, err := st.RotateRefreshToken(ctx, oldHash, RefreshToken{TokenHash: newHash, GrantID: "g", ClientID: "c", UserID: "u", ExpiresAt: now.Add(time.Hour)}, now); err != nil {
			t.Fatalf("rotate refresh: %v", err)
		}
		old, err := st.GetRefreshToken(ctx, oldHash)
		if err != nil {
			t.Fatalf("get old: %v", err)
		}
		if string(old.ReplacedByHash) != string(newHash) {
			t.Fatalf("old token not linked to replacement")
		}
		if _, err := st.RotateRefreshToken(ctx, oldHash, RefreshToken{TokenHash: []byte("other"), GrantID: "g"}, now); !errors.Is(err, ErrRefreshReuseDetected) {
			t.Fatalf("reuse got %v, want reuse detected", err)
		}
		newToken, err := st.GetRefreshToken(ctx, newHash)
		if err != nil {
			t.Fatalf("get new: %v", err)
		}
		if newToken.RevokedAt == nil {
			t.Fatalf("reuse should revoke token family")
		}
	})

	t.Run("consent is normalized and revocable", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		consent := Consent{UserID: "u", ClientID: "c", Scope: []string{"email", "openid", "email"}, GrantedAt: now}
		if err := st.PutConsent(ctx, consent); err != nil {
			t.Fatalf("put consent: %v", err)
		}
		got, err := st.GetConsent(ctx, "u", "c", []string{"openid", "email"})
		if err != nil {
			t.Fatalf("get consent: %v", err)
		}
		if len(got.Scope) != 2 || got.Scope[0] != "email" || got.Scope[1] != "openid" {
			t.Fatalf("scope not normalized: %#v", got.Scope)
		}
		if err := st.RevokeConsent(ctx, "u", "c", []string{"email", "openid"}, now); err != nil {
			t.Fatalf("revoke consent: %v", err)
		}
		revoked, err := st.GetConsent(ctx, "u", "c", []string{"openid", "email"})
		if err != nil {
			t.Fatalf("get revoked consent: %v", err)
		}
		if revoked.RevokedAt == nil {
			t.Fatalf("consent was not revoked")
		}
	})

	t.Run("password credentials and account security state", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now().UTC()
		credential := PasswordCredential{UserID: "u1", Login: "alice", PasswordHash: []byte("encoded-hash"), HashAlgorithm: "argon2id-v1", CreatedAt: now, UpdatedAt: now, PasswordChangedAt: now}
		if err := st.PutPasswordCredential(ctx, credential); err != nil {
			t.Fatalf("put credential: %v", err)
		}
		byLogin, err := st.GetPasswordCredentialByLogin(ctx, "alice")
		if err != nil {
			t.Fatalf("get by login: %v", err)
		}
		if byLogin.UserID != "u1" || string(byLogin.PasswordHash) != "encoded-hash" {
			t.Fatalf("bad credential by login: %#v", byLogin)
		}
		byUser, err := st.GetPasswordCredentialByUserID(ctx, "u1")
		if err != nil {
			t.Fatalf("get by user: %v", err)
		}
		if byUser.Login != "alice" {
			t.Fatalf("bad credential by user: %#v", byUser)
		}
		if err := st.PutPasswordCredential(ctx, PasswordCredential{UserID: "u2", Login: "alice", PasswordHash: []byte("other")}); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("duplicate login got %v, want %v", err, ErrDuplicate)
		}
		lockedUntil := now.Add(time.Minute)
		state := AccountSecurityState{UserID: "u1", FailedLoginCount: 2, LockedUntil: &lockedUntil}
		if err := st.PutAccountSecurityState(ctx, state); err != nil {
			t.Fatalf("put security state: %v", err)
		}
		gotState, err := st.GetAccountSecurityState(ctx, "u1")
		if err != nil {
			t.Fatalf("get security state: %v", err)
		}
		if gotState.FailedLoginCount != 2 || gotState.LockedUntil == nil {
			t.Fatalf("bad security state: %#v", gotState)
		}
		if err := st.ResetAccountSecurityState(ctx, "u1", now); err != nil {
			t.Fatalf("reset security state: %v", err)
		}
		reset, err := st.GetAccountSecurityState(ctx, "u1")
		if err != nil {
			t.Fatalf("get reset state: %v", err)
		}
		if reset.FailedLoginCount != 0 || reset.LockedUntil != nil || reset.LastSuccessfulLoginAt == nil {
			t.Fatalf("bad reset state: %#v", reset)
		}
		if err := st.DeletePasswordCredential(ctx, "u1"); err != nil {
			t.Fatalf("delete credential: %v", err)
		}
		if _, err := st.GetPasswordCredentialByUserID(ctx, "u1"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("deleted credential got %v, want not found", err)
		}
	})

	t.Run("active signing key and verification keys", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		if err := st.CreateSigningKey(ctx, SigningKey{ID: "k1", Algorithm: "RS256"}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateSigningKey(ctx, SigningKey{ID: "k2", Algorithm: "RS256"}); err != nil {
			t.Fatal(err)
		}
		if err := st.ActivateSigningKey(ctx, "k2"); err != nil {
			t.Fatal(err)
		}
		active, err := st.ActiveSigningKey(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if active.ID != "k2" {
			t.Fatalf("active = %s", active.ID)
		}
		keys, err := st.VerificationKeys(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 1 || keys[0].ID != "k2" {
			t.Fatalf("verification keys = %#v", keys)
		}
		if err := st.DeleteRetiredSigningKey(ctx, "k2"); !errors.Is(err, ErrActiveSigningKey) {
			t.Fatalf("purge active key error = %v", err)
		}
		if err := st.RetireSigningKey(ctx, "k1"); err != nil {
			t.Fatal(err)
		}
		if err := st.DeleteRetiredSigningKey(ctx, "k1"); err != nil {
			t.Fatal(err)
		}
		if err := st.ActivateSigningKey(ctx, "k1"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("activate purged key error = %v", err)
		}
	})
}
