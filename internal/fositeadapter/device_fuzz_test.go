package fositeadapter

import (
	"strings"
	"testing"
)

func FuzzNormalizeUserCode(f *testing.F) {
	for _, seed := range []string{"ABCD-EFGH", "abcd efgh", " A-B C-D E-F G-H ", "", "ABCD-EFG", "OOOO-1111", "éééé-éééé"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		normalized := normalizeUserCode(raw)
		if normalized == "" {
			return
		}
		if len(normalized) != 9 || normalized[4] != '-' {
			t.Fatalf("normalized user code %q has invalid shape", normalized)
		}
		if normalized != strings.ToUpper(normalized) {
			t.Fatalf("normalized user code %q is not uppercase", normalized)
		}
		for i, value := range normalized {
			if i == 4 {
				continue
			}
			if !containsUserCodeByte(byte(value)) {
				t.Fatalf("normalized user code %q contains non-alphabet byte %q", normalized, value)
			}
		}
		if got := normalizeUserCode(normalized); got != normalized {
			t.Fatalf("normalization is not idempotent: first=%q second=%q", normalized, got)
		}
	})
}

func FuzzDeviceAndUserCodeHashesAreDomainSeparated(f *testing.F) {
	f.Add("same-code")
	f.Add("")
	f.Fuzz(func(t *testing.T, raw string) {
		key := []byte("device-code-fuzz-key-32-bytes-long")
		device := deviceCodeHash(key, raw)
		user := userCodeHash(key, raw)
		if string(device) == string(user) {
			t.Fatalf("domain-separated device and user hashes collided for %q", raw)
		}
	})
}
