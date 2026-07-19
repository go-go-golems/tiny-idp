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

func TestProductionListenerModesAreExplicitAndMutuallyExclusive(t *testing.T) {
	if _, err := parseProductionListenerMode(""); err == nil {
		t.Fatal("empty listener mode accepted")
	}
	if _, err := parseProductionListenerMode("plaintext"); err == nil {
		t.Fatal("unknown listener mode accepted")
	}
	direct, err := parseProductionListenerMode("direct-tls")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProductionListenerSettings(direct, &serveProductionSettings{TLSCertFile: "cert.pem", TLSKeyFile: "key.pem"}); err != nil {
		t.Fatalf("valid direct TLS settings rejected: %v", err)
	}
	if err := validateProductionListenerSettings(direct, &serveProductionSettings{TLSCertFile: "cert.pem", TLSKeyFile: "key.pem", TrustedProxyCIDRs: []string{"10.42.0.0/24"}}); err == nil {
		t.Fatal("direct TLS accepted proxy CIDRs")
	}
	proxy, err := parseProductionListenerMode("trusted-proxy-http")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProductionListenerSettings(proxy, &serveProductionSettings{Issuer: "https://idp.example.test/idp", TrustedProxyCIDRs: []string{"10.42.0.0/24"}}); err != nil {
		t.Fatalf("valid trusted proxy settings rejected: %v", err)
	}
	if err := validateProductionListenerSettings(proxy, &serveProductionSettings{Issuer: "http://idp.example.test", TrustedProxyCIDRs: []string{"10.42.0.0/24"}}); err == nil {
		t.Fatal("trusted proxy accepted HTTP issuer")
	}
}
