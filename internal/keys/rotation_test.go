package keys_test

import (
	"context"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
)

func TestRotateRSAActivatesNewKeyAndKeepsOldVerifiable(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	old, err := keys.GenerateRSA("old", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, old); err != nil {
		t.Fatal(err)
	}

	active, retired, err := keys.RotateRSA(ctx, st, "new", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if retired == nil || retired.ID != "old" {
		t.Fatalf("retired = %#v", retired)
	}
	if active.ID != "new" || !active.Active {
		t.Fatalf("active = %#v", active)
	}
	current, err := st.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current.ID != "new" {
		t.Fatalf("current active = %s", current.ID)
	}
	verification, err := st.VerificationKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, k := range verification {
		seen[k.ID] = true
	}
	if !seen["old"] || !seen["new"] {
		t.Fatalf("verification keys = %#v", seen)
	}
}

func TestRotateRSACreatesInitialKey(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	active, retired, err := keys.RotateRSA(ctx, st, "initial", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if retired != nil {
		t.Fatalf("unexpected retired key: %#v", retired)
	}
	if active.ID != "initial" || !active.Active {
		t.Fatalf("active = %#v", active)
	}
}
