package production

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func TestDefaultsAndEnvironmentPrecedenceRecordProvenance(t *testing.T) {
	t.Setenv("TINYIDP_ADDR", "127.0.0.1:9443")

	section, err := NewSection()
	if err != nil {
		t.Fatal(err)
	}
	commandSchema := schema.NewSchema(schema.WithSections(section))
	parsed := values.New()
	if err := sources.Execute(
		commandSchema,
		parsed,
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
		sources.FromEnv("tinyidp", fields.WithSource("env")),
	); err != nil {
		t.Fatal(err)
	}

	settings, err := GetSettings(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Addr != "127.0.0.1:9443" {
		t.Fatalf("addr = %q", settings.Addr)
	}
	if settings.RateLimit != 30 || settings.TokenSecretFile != "" {
		t.Fatalf("unexpected defaults: %#v", settings)
	}
	addr, ok := parsed.GetField(Slug, "addr")
	if !ok {
		t.Fatal("addr field is unavailable")
	}
	if len(addr.Log) == 0 || addr.Log[len(addr.Log)-1].Source != "env" {
		t.Fatalf("addr provenance = %#v", addr.Log)
	}
	rateLimit, ok := parsed.GetField(Slug, "rate-limit")
	if !ok {
		t.Fatal("rate-limit field is unavailable")
	}
	if len(rateLimit.Log) == 0 || rateLimit.Log[len(rateLimit.Log)-1].Source != fields.SourceDefaults {
		t.Fatalf("rate-limit provenance = %#v", rateLimit.Log)
	}
}

func TestSectionContainsOnlySecretFileReferences(t *testing.T) {
	section, err := NewSection()
	if err != nil {
		t.Fatal(err)
	}
	definitions := section.GetDefinitions()
	for _, name := range []string{
		"token-secret-file",
		"invitation-lookup-key-file",
		"email-challenge-key-file",
		"email-smtp-password-file",
	} {
		if _, ok := definitions.Get(name); !ok {
			t.Fatalf("missing secret file reference %q", name)
		}
	}
	for _, forbidden := range []string{
		"token-secret",
		"invitation-lookup-key",
		"email-challenge-key",
		"email-smtp-password",
	} {
		if _, ok := definitions.Get(forbidden); ok {
			t.Fatalf("raw secret field %q must not be inspectable", forbidden)
		}
	}
}
