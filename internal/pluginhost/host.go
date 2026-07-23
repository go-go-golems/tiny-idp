// Package pluginhost prepares, builds, routes, observes, and closes compiled-in
// TinyIDP plugins.
package pluginhost

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/values"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func Prepare(ctx context.Context, registry *pluginapi.Registry, vals *values.Values) ([]pluginapi.Prepared, error) {
	if registry == nil || vals == nil {
		return nil, errors.New("plugin registry and parsed values are required")
	}
	prepared := make([]pluginapi.Prepared, 0, len(registry.Definitions()))
	for _, definition := range registry.Definitions() {
		value, err := definition.Prepare(ctx, vals)
		if err != nil {
			return nil, fmt.Errorf("prepare plugin %q: %w", definition.Descriptor().ID, err)
		}
		if value == nil || value.Descriptor() != definition.Descriptor() {
			return nil, fmt.Errorf("prepare plugin %q returned an invalid descriptor", definition.Descriptor().ID)
		}
		prepared = append(prepared, value)
	}
	return prepared, nil
}

func Build(ctx context.Context, prepared []pluginapi.Prepared, services pluginapi.RuntimeServices) ([]pluginapi.Runtime, error) {
	runtimes := make([]pluginapi.Runtime, 0, len(prepared))
	for _, value := range prepared {
		if value == nil || !value.Enabled() {
			continue
		}
		runtime, err := value.Build(ctx, services)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("build plugin %q: %w", value.Descriptor().ID, err), Close(context.Background(), runtimes))
		}
		if runtime == nil || runtime.Handler() == nil || runtime.Descriptor() != value.Descriptor() {
			if runtime != nil {
				_ = runtime.Close(context.Background())
			}
			return nil, errors.Join(fmt.Errorf("build plugin %q returned an invalid runtime", value.Descriptor().ID), Close(context.Background(), runtimes))
		}
		runtimes = append(runtimes, runtime)
	}
	return runtimes, nil
}

func ValidateClientRequirements(prepared []pluginapi.Prepared, clients []idpstore.Client) error {
	byID := make(map[string]idpstore.Client, len(clients))
	for _, client := range clients {
		byID[client.ID] = client
	}
	claimed := map[string]string{}
	for _, value := range prepared {
		if value == nil || !value.Enabled() {
			continue
		}
		for _, requirement := range value.Requirements().OIDCClients {
			if owner, duplicate := claimed[requirement.ID]; duplicate {
				return fmt.Errorf("plugin %q OIDC client %q is already required by plugin %q", value.Descriptor().ID, requirement.ID, owner)
			}
			claimed[requirement.ID] = value.Descriptor().ID
			client, ok := byID[requirement.ID]
			if !ok {
				return fmt.Errorf("plugin %q requires missing OIDC client %q", value.Descriptor().ID, requirement.ID)
			}
			if client.Public != requirement.Public || client.RequirePKCE != requirement.RequirePKCE ||
				!client.AllowsRedirectURI(requirement.RedirectURI) ||
				!slices.Contains(client.AllowedGrantTypes, idpstore.GrantAuthorizationCode) {
				return fmt.Errorf("plugin %q OIDC client %q does not satisfy its security profile", value.Descriptor().ID, requirement.ID)
			}
			for _, scope := range requirement.Scopes {
				if !slices.Contains(client.AllowedScopes, scope) {
					return fmt.Errorf("plugin %q OIDC client %q is missing scope %q", value.Descriptor().ID, requirement.ID, scope)
				}
			}
		}
	}
	return nil
}

func Close(ctx context.Context, runtimes []pluginapi.Runtime) error {
	var closeErr error
	for index := len(runtimes) - 1; index >= 0; index-- {
		closeErr = errors.Join(closeErr, runtimes[index].Close(ctx))
	}
	return closeErr
}

func Mount(mux *http.ServeMux, runtimes []pluginapi.Runtime) error {
	if mux == nil {
		return errors.New("plugin route mux is required")
	}
	mounted := map[string]struct{}{}
	for _, runtime := range runtimes {
		prefix := runtime.Descriptor().RoutePrefix()
		if _, duplicate := mounted[prefix]; duplicate {
			return fmt.Errorf("duplicate plugin route prefix %q", prefix)
		}
		mounted[prefix] = struct{}{}
		mux.Handle(prefix, http.StripPrefix(strings.TrimSuffix(prefix, "/"), SecurityHeaders(runtime.Handler())))
	}
	return nil
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'")
		next.ServeHTTP(writer, request)
	})
}

func Readiness(ctx context.Context, runtimes []pluginapi.Runtime) idp.ReadinessReport {
	report := idp.ReadinessReport{Ready: true, Checks: make([]idp.ReadinessCheck, 0, len(runtimes))}
	for _, runtime := range runtimes {
		check := runtime.Readiness(ctx)
		if check.Name == "" {
			check.Name = "plugin." + runtime.Descriptor().ID
		}
		if check.CheckedAt.IsZero() {
			check.CheckedAt = time.Now().UTC()
		}
		report.Checks = append(report.Checks, check)
		report.Ready = report.Ready && check.Ready
	}
	return report
}
