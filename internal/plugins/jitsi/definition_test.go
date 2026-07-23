package jitsi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreparedValidationIsStrictAndConditional(t *testing.T) {
	valid := &prepared{settings: Settings{
		Enabled: true, PublicOrigin: "https://meet.example.test", XMPPDomain: "meet.example.test",
		AppID: "app", OIDCClientID: "jitsi-client", TokenTTL: "5m",
		SharedSecretFile: "/run/secrets/jitsi", PolicyPoolSize: 2,
	}}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(*prepared)
	}{
		{name: "http origin", mutate: func(value *prepared) { value.settings.PublicOrigin = "http://meet.example.test" }},
		{name: "origin path", mutate: func(value *prepared) { value.settings.PublicOrigin = "https://meet.example.test/path" }},
		{name: "bad domain", mutate: func(value *prepared) { value.settings.XMPPDomain = "meet..example.test" }},
		{name: "long ttl", mutate: func(value *prepared) { value.settings.TokenTTL = "11m" }},
		{name: "missing secret", mutate: func(value *prepared) { value.settings.SharedSecretFile = "" }},
		{name: "large pool", mutate: func(value *prepared) { value.settings.PolicyPoolSize = 33 }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := *valid
			test.mutate(&candidate)
			if err := candidate.validate(); err == nil {
				t.Fatal("invalid settings accepted")
			}
		})
	}
}

func TestReadPolicyRequiresBoundedRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.js")
	if err := os.WriteFile(path, []byte("module.exports = {};"), 0o644); err != nil {
		t.Fatal(err)
	}
	if source, err := readPolicy(path); err != nil || source == "" {
		t.Fatalf("source = %q, %v", source, err)
	}
	if _, err := readPolicy(filepath.Dir(path)); err == nil {
		t.Fatal("directory policy accepted")
	}
}
