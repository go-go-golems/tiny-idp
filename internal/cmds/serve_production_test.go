package cmds

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
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

func TestReadProductionSignupProgramRequiresBoundedRegularSource(t *testing.T) {
	if _, err := readProductionSignupProgram(""); err == nil {
		t.Fatal("empty program path accepted")
	}
	directory := t.TempDir()
	if _, err := readProductionSignupProgram(directory); err == nil {
		t.Fatal("directory accepted as program source")
	}
	path := filepath.Join(directory, "signup.js")
	if err := os.WriteFile(path, []byte(idpsignup.DefaultSource), 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := readProductionSignupProgram(path)
	if err != nil {
		t.Fatal(err)
	}
	if source != idpsignup.DefaultSource {
		t.Fatal("program source changed while reading")
	}
	oversized := filepath.Join(directory, "oversized.js")
	if err := os.WriteFile(oversized, make([]byte, maxProductionSignupProgramBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readProductionSignupProgram(oversized); err == nil {
		t.Fatal("oversized program accepted")
	}
}

func TestNewProductionSignupManagerChecksAndActivatesOnlySupportedPrograms(t *testing.T) {
	manager, err := newProductionSignupManager(context.Background(), idpsignup.DefaultSource, idp.NewMemorySink())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(context.Background())
	if err := manager.Ready(); err != nil {
		t.Fatalf("activated program not ready: %v", err)
	}
	if _, err := newProductionSignupManager(context.Background(), "not javascript", idp.NewMemorySink()); err == nil {
		t.Fatal("invalid program accepted")
	}
	if _, err := newProductionSignupManager(context.Background(), idpsignup.EmailVerifiedSource, idp.NewMemorySink()); err == nil || !strings.Contains(err.Error(), "unsupported native services: email_challenge") {
		t.Fatalf("unsupported email challenge error = %v", err)
	}
	capabilityProgram := `const A = require("tinyidp").v1;
module.exports = A.program("unsupported-capability", p => {
  p.capabilities({"clock.now": {version:1}});
  const start = A.lambda("signup.start", { input:"signupStartInput", output:"signupResult", outcomes:["complete"], effects:[], capabilities:["clock.now"], timeoutMs:250, maxCapabilityCalls:1, maxOutputBytes:1024, run: async ctx => { await ctx.cap.clock.now({}); return A.result.complete(); } });
  p.workflow("signup", { version:1, entry:"start", handlers:{start}, edges:[] });
});`
	_, err = newProductionSignupManager(context.Background(), capabilityProgram, idp.NewMemorySink())
	if err == nil || !strings.Contains(err.Error(), "unsupported native capabilities: clock.now") {
		t.Fatalf("unsupported capability error = %v", err)
	}
}

func TestProductionCommandRequiresSignupProgramAndDropsLegacyRegistrationFlag(t *testing.T) {
	command, err := NewServeProductionCommand()
	if err != nil {
		t.Fatal(err)
	}
	section, ok := command.Schema.Get(schema.DefaultSlug)
	if !ok {
		t.Fatal("default command section is unavailable")
	}
	program, ok := section.GetDefinitions().Get("signup-program-file")
	if !ok || !program.Required {
		t.Fatal("signup-program-file is not a required production flag")
	}
	if _, legacy := section.GetDefinitions().Get("registration-enabled"); legacy {
		t.Fatal("legacy registration-enabled production flag is still exposed")
	}
}
