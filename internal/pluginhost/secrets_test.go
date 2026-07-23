package pluginhost

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSecretResolverAcceptsOnlyBoundedOwnerOnlyFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := FileSecretResolver{}
	value, err := resolver.Read(context.Background(), path, 32)
	if err != nil || len(value) != 32 {
		t.Fatalf("secret = %d bytes, %v", len(value), err)
	}
	zeroSecretBytes(value)
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Read(context.Background(), path, 32); err == nil {
		t.Fatal("group-readable secret accepted")
	}
}
