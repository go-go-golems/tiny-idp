package sqlitestore_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestStoreSuite(t *testing.T) {
	idpstore.RunStoreSuite(t, func(t *testing.T) idpstore.Store {
		st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = st.Close() })
		return st
	})
}

func TestClientGrantCapabilityMigrationBackfillsKnownLegacyProfiles(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "grant-capabilities.db")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	legacyClients := []idpstore.Client{
		{ID: "browser", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}},
		{ID: "device", Public: true, RequirePKCE: true, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantDeviceCode}},
		{ID: "ambiguous", SecretHash: []byte("hash"), AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}},
	}
	for _, client := range legacyClients {
		if err := store.PutClient(ctx, client); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.SQLDB().ExecContext(ctx, `UPDATE clients SET data=json_remove(data, '$.AllowedGrantTypes')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SQLDB().ExecContext(ctx, `DELETE FROM schema_migrations WHERE version=8`); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cases := []struct {
		id   string
		want []string
	}{
		{id: "browser", want: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}},
		{id: "device", want: []string{idpstore.GrantDeviceCode}},
		{id: "ambiguous", want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			client, err := store.GetClient(ctx, tc.id)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(client.AllowedGrantTypes, tc.want) {
				t.Fatalf("AllowedGrantTypes = %#v, want %#v", client.AllowedGrantTypes, tc.want)
			}
		})
	}
	ambiguous, err := store.GetClient(ctx, "ambiguous")
	if err != nil {
		t.Fatal(err)
	}
	if err := ambiguous.Validate(idpstore.ProductionMode); !errors.Is(err, idpstore.ErrClientMissingGrantTypes) {
		t.Fatalf("ambiguous client validation error = %v", err)
	}
}

func TestDeviceGrantSurvivesRestartAndConcurrentConsumptionHasOneWinner(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "device-grant.db")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 15, 15, 0, 0, 0, time.UTC)
	if err := store.PutClient(ctx, idpstore.Client{ID: "device-client"}); err != nil {
		t.Fatal(err)
	}
	grant := idpstore.DeviceGrant{ID: "restartable", DeviceCodeHash: []byte("device-hash"), UserCodeHash: []byte("user-hash"), ClientID: "device-client", Status: idpstore.DeviceGrantPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour), PollInterval: 5 * time.Second, NextPollAt: now}
	if err := store.CreateDeviceGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DecideDeviceGrant(ctx, idpstore.DeviceDecisionRequest{UserCodeHash: grant.UserCodeHash, Decision: idpstore.DeviceGrantApprove, UserID: "u1", Subject: "subject-1", AuthTime: now, AuthenticationMethods: []string{"pwd"}, ApprovedScopes: []string{"openid"}, Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	loaded, err := store.InspectDeviceGrantByDeviceCodeHash(ctx, grant.DeviceCodeHash, grant.ClientID)
	if err != nil || loaded.Status != idpstore.DeviceGrantApproved || loaded.Subject != "subject-1" {
		t.Fatalf("restarted grant = %#v, %v", loaded, err)
	}

	const workers = 16
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.ConsumeDeviceGrant(ctx, idpstore.DeviceConsumeRequest{DeviceCodeHash: grant.DeviceCodeHash, ClientID: grant.ClientID, Now: now.Add(time.Second)})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	winners := 0
	for err := range errs {
		if err == nil {
			winners++
			continue
		}
		if !errors.Is(err, idpstore.ErrAlreadyConsumed) {
			t.Fatalf("concurrent consume error = %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("consume winners = %d, want 1", winners)
	}

	decisionGrant := idpstore.DeviceGrant{ID: "decision-race", DeviceCodeHash: []byte("decision-device-hash"), UserCodeHash: []byte("decision-user-hash"), ClientID: "device-client", Status: idpstore.DeviceGrantPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour), PollInterval: 5 * time.Second, NextPollAt: now}
	if err := store.CreateDeviceGrant(ctx, decisionGrant); err != nil {
		t.Fatal(err)
	}
	errs = make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.DecideDeviceGrant(ctx, idpstore.DeviceDecisionRequest{UserCodeHash: decisionGrant.UserCodeHash, Decision: idpstore.DeviceGrantDeny, Now: now.Add(time.Second)})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	winners = 0
	for err := range errs {
		if err == nil {
			winners++
			continue
		}
		if !errors.Is(err, idpstore.ErrDeviceGrantNotPending) {
			t.Fatalf("concurrent decision error = %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("decision winners = %d, want 1", winners)
	}
}

func TestSigningKeyRotationPersistsRetiredVerificationKey(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "idp.db")
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	old, err := keys.GenerateRSA("old", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, old); err != nil {
		t.Fatal(err)
	}
	if _, _, err := keys.RotateRSA(ctx, st, "new", time.Now()); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st2, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	active, err := st2.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != "new" {
		t.Fatalf("active = %s", active.ID)
	}
	keysForVerify, err := st2.VerificationKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, k := range keysForVerify {
		seen[k.ID] = true
	}
	if !seen["old"] || !seen["new"] {
		t.Fatalf("verification keys = %#v", seen)
	}
}

func TestSigningKeyPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idp.db")
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-restart", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st2, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	active, err := st2.ActiveSigningKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != "kid-restart" {
		t.Fatalf("active key = %s", active.ID)
	}
}
