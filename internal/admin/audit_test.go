package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
)

type failingAuditSink struct{}

func (failingAuditSink) Emit(context.Context, idp.Event) error { return errors.New("disk full") }

func TestCreateUserSurfacesPostCommitAuditFailure(t *testing.T) {
	store := memory.New()
	service, err := NewService(store, Options{Audit: failingAuditSink{}, PasswordPolicy: idp.DevelopmentPasswordAcceptancePolicy(), Clock: func() time.Time { return time.Unix(100, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.CreateUser(context.Background(), CreateUserRequest{Login: "alice", Password: []byte("long-enough-development-password")})
	if !errors.Is(err, idp.ErrAuditDelivery) {
		t.Fatalf("CreateUser error = %v", err)
	}
	if user.ID == "" {
		t.Fatal("committed result must be returned with the audit error")
	}
	if _, getErr := store.GetUserByLogin(context.Background(), "alice"); getErr != nil {
		t.Fatalf("mutation did not commit: %v", getErr)
	}
}
