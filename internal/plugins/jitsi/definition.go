package jitsi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	productionsection "github.com/go-go-golems/tiny-idp/internal/sections/production"
)

const (
	SectionSlug    = "plugin-jitsi"
	sectionPrefix  = "jitsi-"
	maxPolicyBytes = 64 << 10
	maxTokenTTL    = 10 * time.Minute
)

var domainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?$`)

type Definition struct{}

var _ pluginapi.Definition = Definition{}

func (Definition) Descriptor() pluginapi.Descriptor {
	return pluginapi.Descriptor{ID: "jitsi", APIVersion: pluginapi.APIVersion, Summary: "Jitsi Meet token bridge"}
}

type Settings struct {
	Enabled           bool   `glazed:"enabled"`
	PublicOrigin      string `glazed:"public-origin"`
	XMPPDomain        string `glazed:"xmpp-domain"`
	AppID             string `glazed:"app-id"`
	OIDCClientID      string `glazed:"oidc-client-id"`
	TokenTTL          string `glazed:"token-ttl"`
	SharedSecretFile  string `glazed:"shared-secret-file"`
	PolicyProgramFile string `glazed:"policy-program-file"`
	PolicyPoolSize    int    `glazed:"policy-pool-size"`
}

func (Definition) Section() (schema.Section, error) {
	return schema.NewSection(
		SectionSlug, "Jitsi integration", schema.WithPrefix(sectionPrefix),
		schema.WithFields(
			fields.New("enabled", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Enable the Jitsi token bridge")),
			fields.New("public-origin", fields.TypeString, fields.WithHelp("Canonical public HTTPS Jitsi origin")),
			fields.New("xmpp-domain", fields.TypeString, fields.WithHelp("Prosody/Jitsi XMPP domain placed in the JWT subject")),
			fields.New("app-id", fields.TypeString, fields.WithHelp("Jitsi application ID shared with Prosody")),
			fields.New("oidc-client-id", fields.TypeString, fields.WithHelp("Reviewed public PKCE client used by the internal broker")),
			fields.New("token-ttl", fields.TypeString, fields.WithDefault("5m"), fields.WithHelp("Short Jitsi JWT lifetime (maximum 10m)")),
			fields.New("shared-secret-file", fields.TypeString, fields.WithHelp("Owner-only Jitsi HS256 secret file")),
			fields.New("policy-program-file", fields.TypeString, fields.WithHelp("Optional reviewed non-secret Jitsi Goja policy")),
			fields.New("policy-pool-size", fields.TypeInteger, fields.WithDefault(2), fields.WithHelp("Number of warmed exclusive Jitsi policy runtimes")),
		),
	)
}

type prepared struct {
	descriptor   pluginapi.Descriptor
	settings     Settings
	issuer       string
	tokenTTL     time.Duration
	policySource string
}

var _ pluginapi.Prepared = (*prepared)(nil)

func (Definition) Prepare(ctx context.Context, vals *values.Values) (pluginapi.Prepared, error) {
	if ctx == nil || vals == nil {
		return nil, errors.New("jitsi preparation context and values are required")
	}
	settings := Settings{}
	if err := vals.DecodeSectionInto(SectionSlug, &settings); err != nil {
		return nil, fmt.Errorf("decode Jitsi settings: %w", err)
	}
	production, err := productionsection.GetSettings(vals)
	if err != nil {
		return nil, err
	}
	value := &prepared{descriptor: (Definition{}).Descriptor(), settings: settings, issuer: strings.TrimSuffix(production.Issuer, "/")}
	if !settings.Enabled {
		return value, nil
	}
	if err := value.validate(); err != nil {
		return nil, err
	}
	value.tokenTTL, _ = time.ParseDuration(settings.TokenTTL)
	if settings.PolicyProgramFile != "" {
		source, err := readPolicy(settings.PolicyProgramFile)
		if err != nil {
			return nil, err
		}
		value.policySource = source
	}
	return value, nil
}

func (p *prepared) validate() error {
	origin, err := url.Parse(strings.TrimSpace(p.settings.PublicOrigin))
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil ||
		origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return errors.New("jitsi public origin must be an absolute HTTPS origin without path, query, fragment, or userinfo")
	}
	if !domainPattern.MatchString(p.settings.XMPPDomain) || strings.Contains(p.settings.XMPPDomain, "..") {
		return errors.New("jitsi XMPP domain is invalid")
	}
	if strings.TrimSpace(p.settings.AppID) == "" || len(p.settings.AppID) > 128 ||
		strings.TrimSpace(p.settings.OIDCClientID) == "" || len(p.settings.OIDCClientID) > 128 {
		return errors.New("jitsi app ID and OIDC client ID are required and bounded")
	}
	ttl, err := time.ParseDuration(p.settings.TokenTTL)
	if err != nil || ttl <= 0 || ttl > maxTokenTTL {
		return errors.New("jitsi token TTL must be positive and no greater than 10m")
	}
	if strings.TrimSpace(p.settings.SharedSecretFile) == "" || p.settings.PolicyPoolSize <= 0 || p.settings.PolicyPoolSize > 32 {
		return errors.New("jitsi shared secret file and policy pool size between 1 and 32 are required")
	}
	return nil
}

func (p *prepared) Descriptor() pluginapi.Descriptor { return p.descriptor }
func (p *prepared) Enabled() bool                    { return p.settings.Enabled }
func (p *prepared) Requirements() pluginapi.Requirements {
	if !p.Enabled() {
		return pluginapi.Requirements{}
	}
	return pluginapi.Requirements{OIDCClients: []pluginapi.OIDCClientRequirement{{
		ID: p.settings.OIDCClientID, RedirectURI: p.issuer + p.descriptor.RoutePrefix() + "callback",
		Scopes: []string{"openid", "profile", "email"}, Public: true, RequirePKCE: true,
	}}}
}

func (p *prepared) Build(ctx context.Context, services pluginapi.RuntimeServices) (pluginapi.Runtime, error) {
	if services.OIDC == nil || services.Secrets == nil || services.Audit == nil || services.Clock == nil || services.Random == nil {
		return nil, errors.New("jitsi runtime requires OIDC, secrets, audit, clock, and random host services")
	}
	secret, err := services.Secrets.Read(ctx, p.settings.SharedSecretFile, 32)
	if err != nil {
		return nil, fmt.Errorf("read Jitsi signing secret: %w", err)
	}
	defer zeroSecret(secret)
	signer, err := NewSigner(secret, p.settings.AppID, p.settings.XMPPDomain, p.tokenTTL, services.Clock)
	if err != nil {
		return nil, err
	}
	var policy *PolicyExecutor
	if p.policySource != "" {
		policy, err = NewPolicyExecutor(ctx, p.policySource, p.settings.PolicyPoolSize)
		if err != nil {
			_ = signer.Close()
			return nil, err
		}
	}
	runtime, err := newRuntime(p.settings, services, signer, policy)
	if err != nil {
		if policy != nil {
			_ = policy.Close(context.Background())
		}
		_ = signer.Close()
		return nil, err
	}
	return runtime, nil
}

func readPolicy(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open Jitsi policy: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxPolicyBytes {
		return "", errors.New("jitsi policy must be a non-empty bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPolicyBytes+1))
	if err != nil || len(data) > maxPolicyBytes {
		return "", errors.New("read bounded Jitsi policy")
	}
	return string(data), nil
}

func zeroSecret(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
