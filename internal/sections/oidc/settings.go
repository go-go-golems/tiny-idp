package oidc

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// Settings is the type-safe decode target for the OIDC section. Struct tags
// map field names (not env vars) to Go fields; Glazed's DecodeSectionInto
// handles flag/env/config/defaults precedence and fills these in.
type Settings struct {
	Issuer       string   `glazed:"issuer"`
	Addr         string   `glazed:"addr"`
	ClientID     string   `glazed:"client-id"`
	ClientSecret string   `glazed:"client-secret"`
	RedirectURIs []string `glazed:"redirect-uris"`
	UsersFile    string   `glazed:"users-file"`
	Engine       string   `glazed:"engine"`
}

// GetSettings decodes the OIDC section from parsed Glaze values into a
// Settings struct. It normalizes the issuer by trimming a trailing slash so
// discovery/JWKS URLs are consistent regardless of how the user typed it.
func GetSettings(vals *values.Values) (*Settings, error) {
	s := &Settings{}
	if err := vals.DecodeSectionInto(Slug, s); err != nil {
		return nil, fmt.Errorf("decode oidc section: %w", err)
	}
	s.Issuer = strings.TrimRight(s.Issuer, "/")
	return s, nil
}
