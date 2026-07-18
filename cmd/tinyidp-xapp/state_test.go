package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestInitializeStateIsIdempotentAndComplete(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := InitializeStateConfig{
		StateRoot:     root,
		PublicBaseURL: "https://app.example.test/",
		Login:         "alice",
		Password:      []byte("a unique production password phrase 2026"),
		Email:         "alice@example.test",
		Name:          "Alice",
	}
	first, err := InitializeState(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	paths := ResolveStatePaths(root)
	tokenBefore, err := os.ReadFile(paths.TokenSecret)
	if err != nil {
		t.Fatal(err)
	}
	bindingBefore, err := os.ReadFile(paths.ObjectBindingKey)
	if err != nil {
		t.Fatal(err)
	}
	resourceBefore, err := os.ReadFile(paths.ResourceClientSecret)
	if err != nil {
		t.Fatal(err)
	}
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(paths.IdentityDatabase))
	if err != nil {
		t.Fatal(err)
	}
	credentialBefore, err := store.GetPasswordCredentialByLogin(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	config.Password = []byte("a different password must not overwrite existing state")
	second, err := InitializeState(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.PublicBaseURL != "https://app.example.test" || first.Issuer != "https://app.example.test/idp" {
		t.Fatalf("manifest changed: first=%#v second=%#v", first, second)
	}
	tokenAfter, _ := os.ReadFile(paths.TokenSecret)
	bindingAfter, _ := os.ReadFile(paths.ObjectBindingKey)
	resourceAfter, _ := os.ReadFile(paths.ResourceClientSecret)
	if !bytes.Equal(tokenBefore, tokenAfter) || !bytes.Equal(bindingBefore, bindingAfter) || !bytes.Equal(resourceBefore, resourceAfter) {
		t.Fatal("idempotent initialization rotated a security root")
	}
	store, err = sqlitestore.Open(ctx, sqlitestore.DefaultConfig(paths.IdentityDatabase))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	credentialAfter, err := store.GetPasswordCredentialByLogin(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(credentialBefore.PasswordHash, credentialAfter.PasswordHash) {
		t.Fatal("idempotent initialization replaced the first credential")
	}
	client, err := store.GetClient(ctx, developmentClientID)
	if err != nil {
		t.Fatal(err)
	}
	if !client.Public || !client.RequirePKCE || len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != "https://app.example.test/auth/callback" {
		t.Fatalf("unexpected relying-party client: %#v", client)
	}
	deviceClient, err := store.GetClient(ctx, deviceClientID)
	if err != nil {
		t.Fatal(err)
	}
	if !deviceClient.Public || !deviceClient.AllowsGrantType("urn:ietf:params:oauth:grant-type:device_code") || !deviceClient.AllowsAudience([]string{first.ResourceAudience}) {
		t.Fatalf("unexpected device client: %#v", deviceClient)
	}
	resourceClient, err := store.GetClient(ctx, resourceClientID)
	if err != nil {
		t.Fatal(err)
	}
	if resourceClient.Public || !resourceClient.CanIntrospect || !resourceClient.AllowsAudience([]string{first.ResourceAudience}) || len(resourceClient.SecretHash) == 0 {
		t.Fatalf("unexpected resource client: %#v", resourceClient)
	}
	resourceInfo, err := os.Stat(paths.ResourceClientSecret)
	if err != nil || resourceInfo.Mode().Perm() != 0o600 {
		t.Fatalf("resource-client secret stat=%v mode=%#o", err, resourceInfo.Mode().Perm())
	}
	if _, err := store.ActiveSigningKey(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateInitializedState(root); err != nil {
		t.Fatal(err)
	}
}

func TestInitializeStateRejectsConflictsAndIncompleteState(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := InitializeStateConfig{
		StateRoot:     root,
		PublicBaseURL: "https://app.example.test",
		Login:         "alice",
		Password:      []byte("a unique production password phrase 2026"),
	}
	if _, err := InitializeState(ctx, config); err != nil {
		t.Fatal(err)
	}
	config.PublicBaseURL = "https://other.example.test"
	if _, err := InitializeState(ctx, config); err == nil {
		t.Fatal("expected conflicting public URL to be rejected")
	}
	paths := ResolveStatePaths(root)
	if err := os.Remove(paths.TokenSecret); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateInitializedState(root); err == nil {
		t.Fatal("expected missing token secret to make state incomplete")
	}
}

func TestInitializeStateRejectsNonHTTPSOriginBeforeWriting(t *testing.T) {
	root := t.TempDir()
	_, err := InitializeState(context.Background(), InitializeStateConfig{
		StateRoot:     root,
		PublicBaseURL: "http://127.0.0.1:8787",
		Login:         "alice",
		Password:      []byte("a unique production password phrase 2026"),
	})
	if err == nil {
		t.Fatal("expected non-HTTPS production origin rejection")
	}
	if _, statErr := os.Stat(ResolveStatePaths(root).Manifest); !os.IsNotExist(statErr) {
		t.Fatalf("invalid initialization wrote a manifest: %v", statErr)
	}
}

func TestReadOwnerOnlyPasswordStripsOneLineEnding(t *testing.T) {
	file := t.TempDir() + "/password"
	if err := os.WriteFile(file, []byte("a unique password phrase\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	password, err := readOwnerOnlyPassword(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(password) != "a unique password phrase" {
		t.Fatalf("password = %q", password)
	}
	if err := os.Chmod(file, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readOwnerOnlyPassword(file); err == nil {
		t.Fatal("expected loose password-file permissions to be rejected")
	}
}
