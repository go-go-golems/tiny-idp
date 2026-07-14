package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/manuel/tinyidp/internal/admin"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
	"github.com/pkg/errors"
)

const stateManifestVersion = 1

type StatePaths struct {
	Root             string
	Manifest         string
	IdentityDatabase string
	AuditLog         string
	TokenSecret      string
	ObjectBindingKey string
	ObjectRoot       string
	AppAuthDatabase  string
}

func ResolveStatePaths(root string) StatePaths {
	return StatePaths{
		Root:             root,
		Manifest:         filepath.Join(root, "state.json"),
		IdentityDatabase: filepath.Join(root, "identity", "tinyidp.sqlite"),
		AuditLog:         filepath.Join(root, "audit", "tinyidp.jsonl"),
		TokenSecret:      filepath.Join(root, "secrets", "token.key"),
		ObjectBindingKey: filepath.Join(root, "secrets", "object-binding.key"),
		ObjectRoot:       filepath.Join(root, "objects"),
		AppAuthDatabase:  filepath.Join(root, "application", "auth.sqlite"),
	}
}

type StateManifest struct {
	Version       int       `json:"version"`
	PublicBaseURL string    `json:"publicBaseUrl"`
	Issuer        string    `json:"issuer"`
	ClientID      string    `json:"clientId"`
	CreatedAt     time.Time `json:"createdAt"`
}

type InitializeStateConfig struct {
	StateRoot     string
	PublicBaseURL string
	Login         string
	Password      []byte
	Email         string
	Name          string
}

func InitializeState(ctx context.Context, cfg InitializeStateConfig) (_ StateManifest, retErr error) {
	if ctx == nil {
		return StateManifest{}, errors.New("initialization context is required")
	}
	if cfg.StateRoot == "" || cfg.PublicBaseURL == "" || cfg.Login == "" || len(cfg.Password) == 0 {
		return StateManifest{}, errors.New("state root, public base URL, login, and password are required")
	}
	publicBaseURL, err := normalizeProductionBaseURL(cfg.PublicBaseURL)
	if err != nil {
		return StateManifest{}, err
	}
	cfg.PublicBaseURL = publicBaseURL
	cfg.Login = strings.TrimSpace(cfg.Login)
	if cfg.Login == "" {
		return StateManifest{}, errors.New("login is required")
	}
	paths := ResolveStatePaths(filepath.Clean(cfg.StateRoot))
	if err := os.MkdirAll(paths.Root, 0o700); err != nil {
		return StateManifest{}, errors.Wrap(err, "create state root")
	}
	if err := ensureOwnerOnlyDirectory(paths.Root); err != nil {
		return StateManifest{}, err
	}
	desired := StateManifest{
		Version:       stateManifestVersion,
		PublicBaseURL: cfg.PublicBaseURL,
		Issuer:        cfg.PublicBaseURL + "/idp",
		ClientID:      developmentClientID,
	}
	if existing, err := ReadStateManifest(paths.Manifest); err == nil {
		if existing.Version != desired.Version || existing.PublicBaseURL != desired.PublicBaseURL || existing.Issuer != desired.Issuer || existing.ClientID != desired.ClientID {
			return StateManifest{}, errors.New("initialized state manifest conflicts with requested public URL or client identity")
		}
		desired.CreatedAt = existing.CreatedAt
	} else if !os.IsNotExist(errors.Cause(err)) {
		return StateManifest{}, err
	}

	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(paths.IdentityDatabase))
	if err != nil {
		return StateManifest{}, errors.Wrap(err, "open identity database")
	}
	defer func() {
		if err := store.Close(); err != nil && retErr == nil {
			retErr = errors.Wrap(err, "close identity database")
		}
	}()
	if err := os.MkdirAll(filepath.Dir(paths.AuditLog), 0o700); err != nil {
		return StateManifest{}, errors.Wrap(err, "create audit directory")
	}
	audit, err := idp.NewFileAuditSink(paths.AuditLog)
	if err != nil {
		return StateManifest{}, errors.Wrap(err, "open initialization audit")
	}
	defer func() {
		if err := audit.Close(); err != nil && retErr == nil {
			retErr = errors.Wrap(err, "close initialization audit")
		}
	}()
	service, err := admin.NewService(store, admin.Options{Audit: audit})
	if err != nil {
		return StateManifest{}, errors.Wrap(err, "create initialization service")
	}
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		return StateManifest{}, errors.Wrap(err, "create account lifecycle service")
	}

	if _, err := loadOrCreateKey(paths.TokenSecret); err != nil {
		return StateManifest{}, errors.Wrap(err, "initialize token secret")
	}
	if _, err := loadOrCreateKey(paths.ObjectBindingKey); err != nil {
		return StateManifest{}, errors.Wrap(err, "initialize object binding key")
	}
	if err := reconcileRPClient(ctx, service, desired); err != nil {
		return StateManifest{}, err
	}
	if err := reconcileFirstUser(ctx, store, accounts, cfg); err != nil {
		return StateManifest{}, err
	}
	if err := reconcileSigningKey(ctx, store); err != nil {
		return StateManifest{}, err
	}
	if desired.CreatedAt.IsZero() {
		desired.CreatedAt = time.Now().UTC()
	}
	if err := writeManifestAtomic(paths.Manifest, desired); err != nil {
		return StateManifest{}, err
	}
	return desired, nil
}

func normalizeProductionBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.Wrap(err, "parse public base URL")
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("public base URL must be an absolute HTTPS origin without userinfo, query, or fragment")
	}
	if parsed.EscapedPath() != "" && parsed.EscapedPath() != "/" {
		return "", errors.New("public base URL must not contain a path")
	}
	return "https://" + parsed.Host, nil
}

func reconcileRPClient(ctx context.Context, service *admin.Service, manifest StateManifest) error {
	desired := idpstore.Client{
		ID:                     manifest.ClientID,
		Public:                 true,
		RedirectURIs:           []string{manifest.PublicBaseURL + "/auth/callback"},
		PostLogoutRedirectURIs: []string{manifest.PublicBaseURL + "/"},
		AllowedScopes:          []string{"openid", "profile", "email"},
		RequirePKCE:            true,
		AccessTokenTTL:         time.Hour,
		IDTokenTTL:             time.Hour,
		RefreshTokenTTL:        24 * time.Hour,
	}
	existing, err := service.GetClient(ctx, manifest.ClientID)
	if err == nil {
		if existing.Public != desired.Public || existing.RequirePKCE != desired.RequirePKCE || existing.Disabled ||
			!reflect.DeepEqual(existing.RedirectURIs, desired.RedirectURIs) ||
			!reflect.DeepEqual(existing.PostLogoutRedirectURIs, desired.PostLogoutRedirectURIs) ||
			!reflect.DeepEqual(existing.AllowedScopes, desired.AllowedScopes) {
			return errors.New("existing relying-party client conflicts with initialized product contract")
		}
		return nil
	}
	if !errors.Is(err, idpstore.ErrNotFound) {
		return errors.Wrap(err, "read relying-party client")
	}
	_, _, err = service.CreateClient(ctx, admin.CreateClientRequest{
		ID:                     desired.ID,
		Public:                 true,
		RedirectURIs:           desired.RedirectURIs,
		PostLogoutRedirectURIs: desired.PostLogoutRedirectURIs,
		AllowedScopes:          desired.AllowedScopes,
		RequirePKCE:            true,
		AccessTokenTTL:         desired.AccessTokenTTL,
		IDTokenTTL:             desired.IDTokenTTL,
		RefreshTokenTTL:        desired.RefreshTokenTTL,
	})
	return errors.Wrap(err, "create relying-party client")
}

func reconcileFirstUser(ctx context.Context, store idpstore.Store, accounts *idpaccounts.Service, cfg InitializeStateConfig) error {
	existing, err := store.GetUserByLogin(ctx, cfg.Login)
	if err == nil {
		if existing.Disabled || (cfg.Email != "" && existing.Email != cfg.Email) {
			return errors.New("existing first user conflicts with initialization request")
		}
		return nil
	}
	if !errors.Is(err, idpstore.ErrNotFound) {
		return errors.Wrap(err, "read first user")
	}
	_, err = accounts.Create(ctx, idpaccounts.CreateRequest{
		Login:             cfg.Login,
		Password:          cfg.Password,
		Email:             cfg.Email,
		EmailVerified:     cfg.Email != "",
		Name:              cfg.Name,
		PreferredUsername: cfg.Login,
	})
	return errors.Wrap(err, "create first user")
}

func reconcileSigningKey(ctx context.Context, store idpstore.Store) error {
	if _, err := store.ActiveSigningKey(ctx); err == nil {
		return nil
	} else if !errors.Is(err, idpstore.ErrNotFound) {
		return errors.Wrap(err, "read active signing key")
	}
	key, err := keys.GenerateRSA("xapp-initial-signing-key", time.Now().UTC())
	if err != nil {
		return errors.Wrap(err, "generate signing key")
	}
	return errors.Wrap(store.CreateSigningKey(ctx, key), "create signing key")
}

func ReadStateManifest(file string) (StateManifest, error) {
	contents, err := os.ReadFile(file)
	if err != nil {
		return StateManifest{}, err
	}
	var manifest StateManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return StateManifest{}, errors.Wrap(err, "decode state manifest")
	}
	if manifest.Version != stateManifestVersion || manifest.PublicBaseURL == "" || manifest.Issuer == "" || manifest.ClientID == "" || manifest.CreatedAt.IsZero() {
		return StateManifest{}, errors.New("state manifest is incomplete or unsupported")
	}
	return manifest, nil
}

func ValidateInitializedState(root string) (StateManifest, error) {
	paths := ResolveStatePaths(filepath.Clean(root))
	manifest, err := ReadStateManifest(paths.Manifest)
	if err != nil {
		return StateManifest{}, errors.Wrap(err, "read initialized state")
	}
	for label, file := range map[string]string{
		"identity database": paths.IdentityDatabase,
		"token secret":      paths.TokenSecret,
		"binding key":       paths.ObjectBindingKey,
	} {
		info, err := os.Stat(file)
		if err != nil {
			return StateManifest{}, errors.Wrapf(err, "%s is unavailable", label)
		}
		if info.Mode().Perm() != 0o600 {
			return StateManifest{}, fmt.Errorf("%s permissions are %#o, want 0600", label, info.Mode().Perm())
		}
	}
	return manifest, nil
}

func ensureOwnerOnlyDirectory(directory string) error {
	info, err := os.Stat(directory)
	if err != nil {
		return errors.Wrap(err, "stat state root")
	}
	if !info.IsDir() {
		return errors.New("state root is not a directory")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("state root permissions are %#o, must not grant group or other access", info.Mode().Perm())
	}
	return nil
}

func writeManifestAtomic(file string, manifest StateManifest) error {
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return errors.Wrap(err, "encode state manifest")
	}
	contents = append(contents, '\n')
	temporary := file + ".tmp"
	handle, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return errors.Wrap(err, "create state manifest temporary file")
	}
	if err := handle.Chmod(0o600); err != nil {
		_ = handle.Close()
		return errors.Wrap(err, "protect state manifest temporary file")
	}
	written, err := handle.Write(contents)
	if err != nil || written != len(contents) {
		_ = handle.Close()
		_ = os.Remove(temporary)
		if err != nil {
			return errors.Wrap(err, "write state manifest temporary file")
		}
		return errors.New("write state manifest temporary file: short write")
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		_ = os.Remove(temporary)
		return errors.Wrap(err, "sync state manifest temporary file")
	}
	if err := handle.Close(); err != nil {
		_ = os.Remove(temporary)
		return errors.Wrap(err, "close state manifest temporary file")
	}
	if err := os.Rename(temporary, file); err != nil {
		_ = os.Remove(temporary)
		return errors.Wrap(err, "commit state manifest")
	}
	return nil
}
