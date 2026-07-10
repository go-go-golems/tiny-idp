package cmds

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadOwnerOnlySecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token-secret")
	if err := os.WriteFile(path, []byte("0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err := readOwnerOnlySecret(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) != 32 {
		t.Fatalf("secret length = %d", len(secret))
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readOwnerOnlySecret(path); err == nil {
		t.Fatal("expected permissive secret file rejection")
	}
}

func TestParseProductionDurationsRejectsNonPositive(t *testing.T) {
	settings := &serveProductionSettings{RateWindow: "1m", MaintenanceInterval: "15m", ReadHeaderTimeout: "5s", ReadTimeout: "15s", WriteTimeout: "30s", IdleTimeout: "1m", ShutdownTimeout: "0s"}
	if _, _, _, _, _, _, _, err := parseProductionDurations(settings); err == nil {
		t.Fatal("expected zero shutdown timeout rejection")
	}
}
