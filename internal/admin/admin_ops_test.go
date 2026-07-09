package admin_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/admin"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/store/memory"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestServiceClientLifecycle(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	svc, err := admin.NewService(st, admin.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	client, secret, err := svc.CreateClient(ctx, admin.CreateClientRequest{ID: "web-app", GenerateSecret: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid", "email"}, RequirePKCE: true})
	if err != nil {
		t.Fatal(err)
	}
	if client.Public || len(client.SecretHash) == 0 || secret.Secret == "" {
		t.Fatalf("bad client/secret: %#v %#v", client, secret)
	}
	if _, _, err := svc.CreateClient(ctx, admin.CreateClientRequest{ID: "web-app", Secret: "secret", RedirectURIs: []string{"https://app.example.test/callback"}, RequirePKCE: true}); !errors.Is(err, idpstore.ErrDuplicate) {
		t.Fatalf("duplicate err=%v", err)
	}
	disabled, err := svc.SetClientDisabled(ctx, "web-app", true)
	if err != nil {
		t.Fatal(err)
	}
	if !disabled.Disabled {
		t.Fatal("client should be disabled")
	}
	rotated, nextSecret, err := svc.RotateClientSecret(ctx, "web-app")
	if err != nil {
		t.Fatal(err)
	}
	if len(rotated.SecretHash) == 0 || nextSecret.Secret == "" || nextSecret.Secret == secret.Secret {
		t.Fatalf("bad rotated secret: %#v %#v", rotated, nextSecret)
	}
}

func TestServiceKeysDoctorAndBackup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "idp.db")
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := admin.NewService(st, admin.Options{Hasher: passwordhash.New(passwordhash.TestParams()), Clock: func() time.Time { return time.Date(2026, 7, 8, 3, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	if report := svc.Doctor(ctx); report.OK {
		t.Fatalf("doctor should fail before active key exists: %#v", report)
	}
	key, err := svc.GenerateSigningKey(ctx, "kid-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if key.PrivateKeyPEM == nil || !key.Active {
		t.Fatalf("bad generated key: %#v", key)
	}
	client, _, err := svc.CreateClient(ctx, admin.CreateClientRequest{ID: "spa", Public: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, RequirePKCE: true})
	if err != nil {
		t.Fatal(err)
	}
	if client.ID != "spa" {
		t.Fatalf("client=%#v", client)
	}
	if report := svc.Doctor(ctx); !report.OK {
		t.Fatalf("doctor should pass: %#v", report)
	}
	rotation, err := svc.RotateSigningKey(ctx, "kid-2")
	if err != nil {
		t.Fatal(err)
	}
	if rotation.Active.ID != "kid-2" || rotation.Retired == nil || rotation.Retired.ID != "kid-1" {
		t.Fatalf("bad rotation: %#v", rotation)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.db")
	backup, err := admin.CreateSQLiteBackup(ctx, path, backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if backup.Bytes == 0 {
		t.Fatalf("empty backup: %#v", backup)
	}
	if err := admin.VerifySQLiteBackup(ctx, backupPath); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.CreateSQLiteBackup(ctx, path, path); err == nil {
		t.Fatal("expected same-file backup to be rejected")
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Size() != before.Size() || after.Size() == 0 {
		t.Fatalf("source database was modified by rejected backup: before=%d after=%d", before.Size(), after.Size())
	}
}
