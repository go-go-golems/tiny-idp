package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func openExternalMessageApplication(ctx context.Context, stateRoot, issuer string) (*initializedMessageApplication, error) {
	manifest, err := readStateManifest(resolveStatePaths(stateRoot).Manifest)
	if err != nil {
		return nil, errors.Wrap(err, "read application state manifest")
	}
	cookieSecure := strings.HasPrefix(manifest.PublicBaseURL, "https://")
	if err := (externalOIDCConfig{PublicBaseURL: manifest.PublicBaseURL, Issuer: issuer, ClientID: clientID, CookieSecure: cookieSecure}).validate(); err != nil {
		return nil, err
	}
	store, err := openAppStore(ctx, resolveStatePaths(stateRoot).ApplicationDatabase)
	if err != nil {
		return nil, err
	}
	oidc, err := newOIDCClient(ctx, issuer, manifest.PublicBaseURL, &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return &initializedMessageApplication{manifest: manifest, paths: resolveStatePaths(stateRoot), application: store, handler: newMessageApp(store, oidc, nil, nil, cookieSecure)}, nil
}
