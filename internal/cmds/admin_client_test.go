package cmds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveClientSecretFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("operator-managed-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err := resolveClientSecret("", path, false)
	if err != nil || secret != "operator-managed-secret" {
		t.Fatalf("secret length=%d err=%v", len(secret), err)
	}
	if _, err := resolveClientSecret("inline", path, false); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("mutual exclusion error = %v", err)
	}
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveClientSecret("", empty, false); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty error = %v", err)
	}
	symlink := filepath.Join(t.TempDir(), "symlink")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveClientSecret("", symlink, false); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink error = %v", err)
	}
	large := filepath.Join(t.TempDir(), "large")
	if err := os.WriteFile(large, make([]byte, 4097), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveClientSecret("", large, false); err == nil || !strings.Contains(err.Error(), "large") {
		t.Fatalf("large file error = %v", err)
	}
}
