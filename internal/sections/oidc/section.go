// Package oidc defines a reusable Glazed field section for mock OIDC
// Identity Provider configuration.
//
// A "section" in Glazed is a named bundle of field definitions (flags) that
// can be composed into any command. Defining the OIDC config as a section
// means the flags (--issuer, --addr, --client-id, ...), their env-var
// equivalents (TINYIDP_ISSUER, ...), and their config-file schema
// (under the `oidc:` key) are declared in exactly one place.
//
// Any command that needs OIDC provider config composes oidc.NewSection():
//
//	cmds.WithSections(oidc.NewSection(), ...)
//
// The section is intentionally decoupled from internal/server so it can be
// reused without dragging in the HTTP layer.
package oidc

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

// Slug is the section identifier. It becomes the config-file key under which
// these fields are organized:
//
//	oidc:
//	  issuer: http://localhost:5556
//	  client-id: dev-client
const Slug = "oidc"

// NewSection creates the reusable OIDC configuration section.
//
// Fields map 1:1 onto the former OIDC_* env vars, but are now available as
// flags (--issuer), env vars (TINYIDP_ISSUER via AppName "tinyidp"), and
// config-file keys (oidc.issuer), with Glazed's full precedence chain.
func NewSection() (schema.Section, error) {
	return schema.NewSection(
		Slug,
		"OIDC Provider Configuration",
		schema.WithFields(
			fields.New("issuer", fields.TypeString,
				fields.WithDefault("http://localhost:5556"),
				fields.WithHelp("Issuer URL advertised in discovery; all endpoints are derived from it"),
			),
			fields.New("addr", fields.TypeString,
				fields.WithDefault("127.0.0.1:5556"),
				fields.WithHelp("Listen address (binds to loopback by default; set 0.0.0.0:5556 for LAN)"),
			),
			fields.New("client-id", fields.TypeString,
				fields.WithDefault("dev-client"),
				fields.WithHelp("Client ID accepted by the mock IdP (single client until Phase 5)"),
			),
			fields.New("client-secret", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Client secret; if empty the client is treated as public (no secret check at /token)"),
			),
			fields.New("redirect-uris", fields.TypeStringList,
				fields.WithDefault([]string{
					"http://localhost:3000/callback",
					"http://127.0.0.1:3000/callback",
				}),
				fields.WithHelp("Allowlist of redirect URIs (repeat --redirect-uris or pass a list in config)"),
			),
			fields.New("users-file", fields.TypeString,
				fields.WithDefault(""),
				fields.WithHelp("Optional YAML/JSON file with seeded users and claims"),
			),
		),
	)
}
