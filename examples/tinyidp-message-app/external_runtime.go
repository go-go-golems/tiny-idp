package main

import (
	"context"
	"strings"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/pkg/errors"
)

func openExternalMessageApplication(ctx context.Context, stateRoot, issuer, backchannelURL string) (*initializedMessageApplication, error) {
	manifest, paths, err := validateStateRoot(stateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "validate external application state")
	}
	cookieSecure := strings.HasPrefix(manifest.PublicBaseURL, "https://")
	if err := (externalOIDCConfig{PublicBaseURL: manifest.PublicBaseURL, Issuer: issuer, BackchannelURL: backchannelURL, ClientID: clientID, CookieSecure: cookieSecure}).validate(); err != nil {
		return nil, err
	}
	store, err := openAppStore(ctx, paths.ApplicationDatabase)
	if err != nil {
		return nil, err
	}
	client, err := externalBackchannelClient(issuer, backchannelURL)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	audit, err := idp.NewFileAuditSink(paths.AuditLog)
	if err != nil {
		_ = store.Close()
		return nil, errors.Wrap(err, "open application audit log")
	}
	client.Timeout = 10 * time.Second
	oidc, err := newOIDCClient(ctx, issuer, manifest.PublicBaseURL, client)
	if err != nil {
		_ = audit.Close()
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
	handler.audit = audit
	return &initializedMessageApplication{manifest: manifest, paths: paths, application: store, audit: audit, handler: handler}, nil
}
