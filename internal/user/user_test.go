package user

import (
	"testing"
)

func TestFromLoginDerivesStableSub(t *testing.T) {
	a1 := FromLogin("alice")
	a2 := FromLogin("alice")
	if a1.Sub != a2.Sub {
		t.Fatalf("sub not stable: %q vs %q", a1.Sub, a2.Sub)
	}
	if a1.Sub == "alice" {
		t.Fatalf("sub must not equal the raw login")
	}
}

func TestFromLoginDistinctUsers(t *testing.T) {
	if FromLogin("alice").Sub == FromLogin("bob").Sub {
		t.Fatal("alice and bob share a sub")
	}
}

func TestNormalizeLowercasesAndTrims(t *testing.T) {
	if Normalize(" Alice ") != "alice" {
		t.Fatalf("normalize(%%q) = %q", " Alice ")
	}
	if Normalize("ALICE") != "alice" {
		t.Fatal("Normalize did not lowercase")
	}
	if FromLogin("Alice").Sub != FromLogin("alice").Sub {
		t.Fatal("Alice and alice must share a sub")
	}
	if FromLogin(" alice ").Sub != FromLogin("alice").Sub {
		t.Fatal("' alice ' and alice must share a sub")
	}
}

func TestFromLoginEmailDomain(t *testing.T) {
	// Plain login -> synthetic domain.
	if got := FromLogin("alice").Email; got != "alice@example.test" {
		t.Fatalf("plain login email = %q, want alice@example.test", got)
	}
	// Email login -> preserved.
	if got := FromLogin("carol@example.test").Email; got != "carol@example.test" {
		t.Fatalf("email login email = %q, want carol@example.test", got)
	}
}

func TestFromLoginNameStripsDomain(t *testing.T) {
	if got := FromLogin("carol@example.test").Name; got != "carol" {
		t.Fatalf("name = %q, want carol", got)
	}
	if got := FromLogin("alice").Name; got != "alice" {
		t.Fatalf("name = %q, want alice", got)
	}
}
