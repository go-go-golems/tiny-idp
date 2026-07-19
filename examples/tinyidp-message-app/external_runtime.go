package main

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func openExternalMessageApplication(ctx context.Context, stateRoot, issuer, backchannelURL string) (*initializedMessageApplication, error) {
	manifest, err := readStateManifest(resolveStatePaths(stateRoot).Manifest)
	if err != nil {
		return nil, errors.Wrap(err, "read application state manifest")
	}
	cookieSecure := strings.HasPrefix(manifest.PublicBaseURL, "https://")
	if err := (externalOIDCConfig{PublicBaseURL: manifest.PublicBaseURL, Issuer: issuer, BackchannelURL: backchannelURL, ClientID: clientID, CookieSecure: cookieSecure}).validate(); err != nil {
		return nil, err
	}
	store, err := openAppStore(ctx, resolveStatePaths(stateRoot).ApplicationDatabase)
	if err != nil {
		return nil, err
	}
	client, err := externalBackchannelClient(issuer, backchannelURL)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	client.Timeout = 10 * time.Second
	oidc, err := newOIDCClient(ctx, issuer, manifest.PublicBaseURL, client)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	handler := newMessageApp(store, oidc, nil, nil, cookieSecure)
	// The external example delegates account provisioning to the separately
	// operated identity provider, so it does not expose the embedded app's
	// self-registration endpoints or registration form.
	handler.registrationEnabled = false
	// Account creation is instead initiated through /auth/register and carried
	// out by Tiny-IDP as part of the authorization transaction.
	handler.providerRegistrationEnabled = true
	return &initializedMessageApplication{manifest: manifest, paths: resolveStatePaths(stateRoot), application: store, handler: handler}, nil
}
