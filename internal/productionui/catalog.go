package productionui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-go-golems/tiny-idp/internal/productionconfig"
)

const (
	MaxThemeCatalogBytes = 256 << 10
	MaxThemeCSSBytes     = 256 << 10
	assetRoutePrefix     = "/static/themes/"
)

var themeNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type CatalogDocument struct {
	Version      int                    `json:"version"`
	DefaultTheme string                 `json:"defaultTheme"`
	Themes       map[string]ThemeConfig `json:"themes"`
	ClientThemes map[string]string      `json:"clientThemes"`
}

type ThemeConfig struct {
	ProductName string `json:"productName"`
	Stylesheet  string `json:"stylesheet"`
}

type Theme struct {
	Name            string
	ProductName     string
	StylesheetRoute string
	css             []byte
}

type Catalog struct {
	defaultTheme string
	themes       map[string]Theme
	clientThemes map[string]string
	assets       map[string][]byte
}

func LoadCatalog(themeDir, catalogFile string, clients *productionconfig.ClientCatalog) (*Catalog, error) {
	if strings.TrimSpace(themeDir) == "" {
		return nil, fmt.Errorf("theme directory is required")
	}
	root, err := filepath.Abs(themeDir)
	if err != nil {
		return nil, fmt.Errorf("resolve theme directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat theme directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("theme directory must be a directory")
	}
	if strings.TrimSpace(catalogFile) == "" {
		return nil, fmt.Errorf("theme catalog file is required")
	}
	catalogPath, err := filepath.Abs(catalogFile)
	if err != nil {
		return nil, fmt.Errorf("resolve theme catalog file: %w", err)
	}
	if !pathWithinRoot(root, catalogPath) {
		return nil, fmt.Errorf("theme catalog file must be inside theme directory")
	}
	data, err := readRegularFile(catalogPath, MaxThemeCatalogBytes, "theme catalog")
	if err != nil {
		return nil, err
	}
	if err := rejectDuplicateObjectKeys(data); err != nil {
		return nil, fmt.Errorf("decode theme catalog: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	document := CatalogDocument{}
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode theme catalog: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values are not allowed")
		}
		return nil, fmt.Errorf("decode theme catalog: %w", err)
	}
	return NewCatalog(root, document, clients)
}

// rejectDuplicateObjectKeys closes a subtle gap in encoding/json's default map
// behavior: repeated object keys otherwise overwrite earlier values. A theme
// name and every other catalog field must have one unambiguous declaration.
func rejectDuplicateObjectKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var visitValue func() error
	visitValue = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, isDelimiter := token.(json.Delim)
		if !isDelimiter {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("object key is not a string")
				}
				if _, duplicate := seen[key]; duplicate {
					return fmt.Errorf("duplicate object key %q", key)
				}
				seen[key] = struct{}{}
				if err := visitValue(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := visitValue(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
		}
	}
	return visitValue()
}

func NewCatalog(themeDir string, document CatalogDocument, clients *productionconfig.ClientCatalog) (*Catalog, error) {
	if clients == nil {
		return nil, fmt.Errorf("client catalog is required")
	}
	if document.Version != 1 {
		return nil, fmt.Errorf("theme catalog version must be 1")
	}
	if len(document.Themes) == 0 {
		return nil, fmt.Errorf("theme catalog requires at least one theme")
	}
	if !themeNamePattern.MatchString(document.DefaultTheme) {
		return nil, fmt.Errorf("default theme name is invalid")
	}
	if _, ok := document.Themes[document.DefaultTheme]; !ok {
		return nil, fmt.Errorf("default theme %q is not declared", document.DefaultTheme)
	}
	catalog := &Catalog{
		defaultTheme: document.DefaultTheme,
		themes:       make(map[string]Theme, len(document.Themes)),
		clientThemes: make(map[string]string, len(document.ClientThemes)),
		assets:       make(map[string][]byte, len(document.Themes)),
	}
	names := make([]string, 0, len(document.Themes))
	for name := range document.Themes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		config := document.Themes[name]
		if !themeNamePattern.MatchString(name) {
			return nil, fmt.Errorf("theme name %q is invalid", name)
		}
		productName := strings.TrimSpace(config.ProductName)
		if productName == "" || productName != config.ProductName || len(productName) > 100 {
			return nil, fmt.Errorf("theme %q product name must be canonical and at most 100 bytes", name)
		}
		if !validStylesheetName(config.Stylesheet) {
			return nil, fmt.Errorf("theme %q stylesheet must be a CSS basename", name)
		}
		css, err := readRegularFile(filepath.Join(themeDir, config.Stylesheet), MaxThemeCSSBytes, "theme stylesheet")
		if err != nil {
			return nil, fmt.Errorf("theme %q: %w", name, err)
		}
		route := assetRoutePrefix + name + ".css"
		catalog.themes[name] = Theme{Name: name, ProductName: productName, StylesheetRoute: route, css: append([]byte(nil), css...)}
		catalog.assets[route] = append([]byte(nil), css...)
	}
	for clientID, themeName := range document.ClientThemes {
		if !clients.Has(clientID) {
			return nil, fmt.Errorf("theme mapping references undeclared client %q", clientID)
		}
		if _, ok := catalog.themes[themeName]; !ok {
			return nil, fmt.Errorf("client %q references undeclared theme %q", clientID, themeName)
		}
		catalog.clientThemes[clientID] = themeName
	}
	return catalog, nil
}

func (c *Catalog) Resolve(clientID string) (Theme, error) {
	if c == nil {
		return Theme{}, fmt.Errorf("theme catalog is unavailable")
	}
	name := c.clientThemes[clientID]
	if name == "" {
		name = c.defaultTheme
	}
	theme, ok := c.themes[name]
	if !ok {
		return Theme{}, fmt.Errorf("resolved theme %q is unavailable", name)
	}
	theme.css = nil
	return theme, nil
}

func (c *Catalog) AssetsHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if c == nil || (request.Method != http.MethodGet && request.Method != http.MethodHead) {
			http.NotFound(writer, request)
			return
		}
		css, ok := c.assets[request.URL.Path]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/css; charset=utf-8")
		writer.Header().Set("Cache-Control", "public, max-age=300")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.WriteHeader(http.StatusOK)
		if request.Method == http.MethodGet {
			_, _ = writer.Write(css)
		}
	})
}

func validStylesheetName(name string) bool {
	return name != "" && name == filepath.Base(name) && !strings.ContainsAny(name, `/\\?#`) && strings.HasSuffix(name, ".css") && name != ".css"
}

func pathWithinRoot(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func readRegularFile(path string, limit int64, label string) ([]byte, error) {
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
