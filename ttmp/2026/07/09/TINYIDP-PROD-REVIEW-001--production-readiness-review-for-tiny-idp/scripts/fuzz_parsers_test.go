package main

import (
	"testing"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/oidcmeta"
	"github.com/manuel/tinyidp/internal/passwordhash"
)

func FuzzIssuerParsing(f *testing.F) {
	for _, seed := range []string{
		"https://id.example.test",
		"https://id.example.test/realms/demo",
		"http://127.0.0.1:5556",
		"https://id.example.test?query=discarded#fragment",
		"not a URL",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		issuer, err := oidcmeta.ParseIssuer(raw)
		if err != nil {
			return
		}
		if issuer.URL.Scheme == "" || issuer.URL.Host == "" {
			t.Fatalf("successful parse lacks scheme or host: %#v", issuer.URL)
		}
		if issuer.URL.Fragment != "" || issuer.URL.RawQuery != "" {
			t.Fatalf("successful parse retained query or fragment: %q", issuer.String())
		}
	})
}

func FuzzProductionRedirectURI(f *testing.F) {
	for _, seed := range []string{
		"https://client.example.test/callback",
		"http://localhost:3000/callback",
		"http://127.0.0.1:3000/callback",
		"https://client.example.test/callback#fragment",
		"javascript:alert(1)",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		_ = domain.ValidateRedirectURI(raw, domain.ProductionMode)
	})
}

func FuzzArgon2idHashParsing(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$a2V5"),
		[]byte("$argon2id$v=18$m=1,t=1,p=1$c2FsdA$a2V5"),
		{},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, encoded []byte) {
		parsed, err := passwordhash.Parse(encoded)
		if err != nil {
			return
		}
		if parsed.Params.MemoryKiB == 0 || parsed.Params.Iterations == 0 || parsed.Params.Parallelism == 0 {
			t.Fatalf("successful parse returned zero work factor: %#v", parsed.Params)
		}
		if len(parsed.Salt) == 0 || len(parsed.Key) == 0 {
			t.Fatal("successful parse returned empty salt or key")
		}
	})
}
