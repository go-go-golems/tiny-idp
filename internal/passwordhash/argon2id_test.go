package passwordhash_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/manuel/tinyidp/internal/passwordhash"
)

func TestArgon2idHashVerifyAndParse(t *testing.T) {
	h := passwordhash.New(passwordhash.TestParams())
	encoded, err := h.HashPassword([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(encoded), "$argon2id$v=19$") {
		t.Fatalf("encoded hash = %q", encoded)
	}
	needsRehash, err := h.VerifyPassword([]byte("correct horse battery staple"), encoded)
	if err != nil {
		t.Fatal(err)
	}
	if needsRehash {
		t.Fatal("fresh hash should not need rehash")
	}
	parsed, err := passwordhash.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Params.MemoryKiB != passwordhash.TestParams().MemoryKiB || parsed.Params.KeyLength == 0 || parsed.Params.SaltLength == 0 {
		t.Fatalf("bad parsed params: %#v", parsed.Params)
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	h := passwordhash.New(passwordhash.TestParams())
	encoded, err := h.HashPassword([]byte("right"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.VerifyPassword([]byte("wrong"), encoded); !errors.Is(err, passwordhash.ErrPasswordMismatch) {
		t.Fatalf("err=%v, want mismatch", err)
	}
}

func TestVerifyReportsNeedsRehash(t *testing.T) {
	oldHasher := passwordhash.New(passwordhash.Params{MemoryKiB: 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16})
	encoded, err := oldHasher.HashPassword([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	newHasher := passwordhash.New(passwordhash.Params{MemoryKiB: 1024, Iterations: 2, Parallelism: 1, SaltLength: 8, KeyLength: 16})
	needsRehash, err := newHasher.VerifyPassword([]byte("secret"), encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !needsRehash {
		t.Fatal("expected rehash after parameter change")
	}
}

func TestParseRejectsMalformedHash(t *testing.T) {
	if _, err := passwordhash.Parse([]byte("not-a-hash")); !errors.Is(err, passwordhash.ErrInvalidHash) {
		t.Fatalf("err=%v, want invalid hash", err)
	}
}
