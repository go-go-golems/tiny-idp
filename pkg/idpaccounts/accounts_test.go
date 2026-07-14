package idpaccounts

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestCreateAuthenticateAndSetPassword(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	service := testService(t, store, Options{})
	u, err := service.Create(ctx, CreateRequest{Login: " Alice ", Password: []byte("alice-password-long"), Email: "alice@example.test", Name: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == "" || u.Sub == "" || u.PreferredUsername != "alice" {
		t.Fatalf("created user = %#v", u)
	}
	if _, err := service.Create(ctx, CreateRequest{Login: "alice", Password: []byte("other-password-long")}); !errors.Is(err, idpstore.ErrDuplicate) {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := service.AuthenticatePassword(ctx, "alice", "alice-password-long", idp.LoginMetadata{}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetPassword(ctx, SetPasswordRequest{Login: "alice", Password: []byte("replacement-password-long")}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticatePassword(ctx, "alice", "replacement-password-long", idp.LoginMetadata{}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAndSetPasswordEnforceAcceptancePolicy(t *testing.T) {
	service := testService(t, memory.New(), Options{})
	ctx := context.Background()
	if _, err := service.Create(ctx, CreateRequest{Login: "alice", Password: []byte("too-short")}); !errors.Is(err, idp.ErrPasswordRejected) {
		t.Fatalf("short create password error = %v", err)
	}
	if _, err := service.Create(ctx, CreateRequest{Login: "alice", Password: []byte("a valid password phrase")}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetPassword(ctx, SetPasswordRequest{Login: "alice", Password: []byte("temporarypassword")}); !errors.Is(err, idp.ErrPasswordRejected) {
		t.Fatalf("blocklisted replacement error = %v", err)
	}
}

func TestCreateRejectsDuplicateExplicitIDAtomically(t *testing.T) {
	stores := map[string]func(*testing.T) idpstore.Store{
		"memory": func(*testing.T) idpstore.Store { return memory.New() },
		"sqlite": func(t *testing.T) idpstore.Store {
			store, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			return store
		},
	}
	for name, newStore := range stores {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := newStore(t)
			service, err := newService(store, Options{}, passwordhash.New(passwordhash.TestParams()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Create(ctx, CreateRequest{Login: "alice", ID: "fixed-user-id", Password: []byte("alice-password-long")}); err != nil {
				t.Fatal(err)
			}
			if _, err := service.Create(ctx, CreateRequest{Login: "alias", ID: "fixed-user-id", Password: []byte("alias-password-long")}); !errors.Is(err, idpstore.ErrDuplicate) {
				t.Fatalf("duplicate ID error = %v", err)
			}
			if _, err := store.GetUserByLogin(ctx, "alias"); !errors.Is(err, idpstore.ErrNotFound) {
				t.Fatalf("duplicate left login alias: %v", err)
			}
		})
	}
}

type failingAuditSink struct{}

func (failingAuditSink) Emit(context.Context, idp.Event) error { return errors.New("disk full") }

func TestCreateReturnsCommittedResultWithAuditFailure(t *testing.T) {
	store := memory.New()
	service := testService(t, store, Options{Audit: failingAuditSink{}, Clock: func() time.Time { return time.Unix(100, 0) }})
	u, err := service.Create(context.Background(), CreateRequest{Login: "alice", Password: []byte("long-enough-development-password")})
	if !errors.Is(err, idp.ErrAuditDelivery) || u.ID == "" {
		t.Fatalf("create result = %#v, error = %v", u, err)
	}
	if _, err := store.GetUserByLogin(context.Background(), "alice"); err != nil {
		t.Fatalf("mutation did not commit: %v", err)
	}
}
