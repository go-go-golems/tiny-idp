package productionconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

const MaxClientCatalogBytes = 256 << 10

type ClientCatalogDocument struct {
	Version int                   `json:"version"`
	Clients []BrowserClientConfig `json:"clients"`
}

type BrowserClientConfig struct {
	ID                     string   `json:"id"`
	Profile                string   `json:"profile"`
	RedirectURIs           []string `json:"redirectURIs"`
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	AllowedScopes          []string `json:"allowedScopes"`
}

type ClientCatalog struct {
	specs []embeddedidp.ClientSpec
	ids   map[string]struct{}
}

func LoadClientCatalog(path string) (*ClientCatalog, error) {
	data, err := readBoundedRegularFile(path, MaxClientCatalogBytes, "client catalog")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	document := ClientCatalogDocument{}
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode client catalog: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode client catalog: %w", err)
	}
	return NewClientCatalog(document)
}

func NewClientCatalog(document ClientCatalogDocument) (*ClientCatalog, error) {
	if document.Version != 1 {
		return nil, fmt.Errorf("client catalog version must be 1")
	}
	if len(document.Clients) == 0 {
		return nil, fmt.Errorf("client catalog requires at least one client")
	}
	catalog := &ClientCatalog{specs: make([]embeddedidp.ClientSpec, 0, len(document.Clients)), ids: make(map[string]struct{}, len(document.Clients))}
	for index, declared := range document.Clients {
		id := strings.TrimSpace(declared.ID)
		if id == "" || id != declared.ID {
			return nil, fmt.Errorf("client %d id must be non-empty and canonical", index)
		}
		if _, duplicate := catalog.ids[id]; duplicate {
			return nil, fmt.Errorf("duplicate client id %q", id)
		}
		if declared.Profile != "browser" {
			return nil, fmt.Errorf("client %q profile must be browser", id)
		}
		if err := validateHTTPSURLs(id, "redirect URI", declared.RedirectURIs, true); err != nil {
			return nil, err
		}
		if err := validateHTTPSURLs(id, "post-logout redirect URI", declared.PostLogoutRedirectURIs, true); err != nil {
			return nil, err
		}
		scopes, err := canonicalStrings(id, "scope", declared.AllowedScopes, true)
		if err != nil {
			return nil, err
		}
		if !contains(scopes, "openid") {
			return nil, fmt.Errorf("client %q scopes must include openid", id)
		}
		redirects := append([]string(nil), declared.RedirectURIs...)
		logoutRedirects := append([]string(nil), declared.PostLogoutRedirectURIs...)
		spec := embeddedidp.BrowserClient(id, redirects, logoutRedirects, scopes)
		if err := spec.Client.Validate(idpstore.ProductionMode); err != nil {
			return nil, fmt.Errorf("client %q: %w", id, err)
		}
		catalog.ids[id] = struct{}{}
		catalog.specs = append(catalog.specs, spec)
	}
	sort.Slice(catalog.specs, func(i, j int) bool { return catalog.specs[i].Client.ID < catalog.specs[j].Client.ID })
	return catalog, nil
}

func (c *ClientCatalog) Specs() []embeddedidp.ClientSpec {
	if c == nil {
		return nil
	}
	out := make([]embeddedidp.ClientSpec, len(c.specs))
	for index, spec := range c.specs {
		out[index] = spec
		out[index].Client.RedirectURIs = append([]string(nil), spec.Client.RedirectURIs...)
		out[index].Client.PostLogoutRedirectURIs = append([]string(nil), spec.Client.PostLogoutRedirectURIs...)
		out[index].Client.AllowedScopes = append([]string(nil), spec.Client.AllowedScopes...)
		out[index].Client.AllowedGrantTypes = append([]string(nil), spec.Client.AllowedGrantTypes...)
	}
	return out
}

func (c *ClientCatalog) Has(id string) bool {
	if c == nil {
		return false
	}
	_, ok := c.ids[id]
	return ok
}

func readBoundedRegularFile(path string, limit int64, label string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%s path is required", label)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", label, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file", label)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("%s must not be empty", label)
	}
	return data, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func validateHTTPSURLs(clientID, label string, values []string, required bool) error {
	canonical, err := canonicalStrings(clientID, label, values, required)
	if err != nil {
		return err
	}
	for _, raw := range canonical {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
			return fmt.Errorf("client %q %s %q must be an absolute HTTPS URL without user info or fragment", clientID, label, raw)
		}
	}
	return nil
}

func canonicalStrings(clientID, label string, values []string, required bool) ([]string, error) {
	if required && len(values) == 0 {
		return nil, fmt.Errorf("client %q requires at least one %s", clientID, label)
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed != value {
			return nil, fmt.Errorf("client %q %s values must be non-empty and canonical", clientID, label)
		}
		if _, duplicate := seen[trimmed]; duplicate {
			return nil, fmt.Errorf("client %q has duplicate %s %q", clientID, label, trimmed)
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
