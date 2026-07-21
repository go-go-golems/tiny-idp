package productionconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewClientCatalogAcceptsTwoBrowserClients(t *testing.T) {
	catalog, err := NewClientCatalog(ClientCatalogDocument{Version: 1, Clients: []BrowserClientConfig{
		{ID: "message-desk", Profile: "browser", RedirectURIs: []string{"https://message.example/auth/callback"}, PostLogoutRedirectURIs: []string{"https://message.example/"}, AllowedScopes: []string{"profile", "openid"}},
		{ID: "goja-auth", Profile: "browser", RedirectURIs: []string{"https://goja.example/auth/callback"}, PostLogoutRedirectURIs: []string{"https://goja.example/"}, AllowedScopes: []string{"openid", "email", "profile"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Has("message-desk") || !catalog.Has("goja-auth") || len(catalog.Specs()) != 2 {
		t.Fatalf("catalog = %#v", catalog)
	}
	if got := catalog.Specs()[0].Client.ID; got != "goja-auth" {
		t.Fatalf("first sorted client = %q", got)
	}
}

func TestNewClientCatalogRejectsUnsafeDeclarations(t *testing.T) {
	valid := BrowserClientConfig{ID: "app", Profile: "browser", RedirectURIs: []string{"https://app.example/auth/callback"}, PostLogoutRedirectURIs: []string{"https://app.example/"}, AllowedScopes: []string{"openid"}}
	tests := []struct {
		name     string
		mutate   func(*ClientCatalogDocument)
		contains string
	}{
		{name: "version", mutate: func(d *ClientCatalogDocument) { d.Version = 2 }, contains: "version must be 1"},
		{name: "duplicate id", mutate: func(d *ClientCatalogDocument) { d.Clients = append(d.Clients, d.Clients[0]) }, contains: "duplicate client id"},
		{name: "unknown profile", mutate: func(d *ClientCatalogDocument) { d.Clients[0].Profile = "device" }, contains: "profile must be browser"},
		{name: "HTTP redirect", mutate: func(d *ClientCatalogDocument) { d.Clients[0].RedirectURIs = []string{"http://app.example/callback"} }, contains: "absolute HTTPS"},
		{name: "missing openid", mutate: func(d *ClientCatalogDocument) { d.Clients[0].AllowedScopes = []string{"profile"} }, contains: "include openid"},
		{name: "duplicate scope", mutate: func(d *ClientCatalogDocument) { d.Clients[0].AllowedScopes = []string{"openid", "openid"} }, contains: "duplicate scope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := ClientCatalogDocument{Version: 1, Clients: []BrowserClientConfig{valid}}
			tt.mutate(&document)
			_, err := NewClientCatalog(document)
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("error = %v, want %q", err, tt.contains)
			}
		})
	}
}

func TestLoadClientCatalogIsStrictAndBounded(t *testing.T) {
	directory := t.TempDir()
	unknown := filepath.Join(directory, "unknown.json")
	if err := os.WriteFile(unknown, []byte(`{"version":1,"clients":[],"surprise":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadClientCatalog(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
	oversized := filepath.Join(directory, "oversized.json")
	if err := os.WriteFile(oversized, make([]byte, MaxClientCatalogBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadClientCatalog(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized error = %v", err)
	}
}
