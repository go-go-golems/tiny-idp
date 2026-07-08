package admin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/admin"
	"github.com/manuel/tinyidp/internal/authn"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/storage"
	"github.com/manuel/tinyidp/internal/store/memory"
)

func TestServiceCreateUserAndAuthenticate(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	now := time.Date(2026, 7, 8, 2, 0, 0, 0, time.UTC)
	svc, err := admin.NewService(st, admin.Options{Hasher: passwordhash.New(passwordhash.TestParams()), Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	u, err := svc.CreateUser(ctx, admin.CreateUserRequest{Login: "Alice", Password: []byte("alice-password"), Email: "alice@example.test", Name: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Sub == "" || u.ID == "" || u.PreferredUsername != "alice" {
		t.Fatalf("bad user: %#v", u)
	}
	if _, err := svc.CreateUser(ctx, admin.CreateUserRequest{Login: "alice", Password: []byte("other-password")}); !errors.Is(err, storage.ErrDuplicate) {
		t.Fatalf("duplicate err=%v", err)
	}
	authSvc, err := authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	result, err := authSvc.AuthenticatePassword(ctx, "alice", "alice-password", authn.LoginMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Sub != u.Sub {
		t.Fatalf("auth user = %#v, want sub %s", result.User, u.Sub)
	}
}

func TestServiceSetPasswordAndDisableUser(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	svc, err := admin.NewService(st, admin.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateUser(ctx, admin.CreateUserRequest{Login: "bob", Password: []byte("old-password")}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetPassword(ctx, admin.SetPasswordRequest{Login: "bob", Password: []byte("new-password"), MustChangeAtLogin: true}); err != nil {
		t.Fatal(err)
	}
	cred, err := st.GetPasswordCredentialByLogin(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}
	if !cred.MustChangeAtLogin {
		t.Fatal("password credential should require change at login")
	}
	if _, err := svc.SetUserDisabled(ctx, "bob", true); err != nil {
		t.Fatal(err)
	}
	authSvc, err := authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authSvc.AuthenticatePassword(ctx, "bob", "new-password", authn.LoginMetadata{}); !errors.Is(err, authn.ErrAccountDisabled) {
		t.Fatalf("disabled login err=%v", err)
	}
}
