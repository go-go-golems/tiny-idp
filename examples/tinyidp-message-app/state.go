package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const stateManifestVersion = 1

type statePaths struct {
	Root                string
	Manifest            string
	IdentityDatabase    string
	ApplicationDatabase string
	TokenSecret         string
	SessionSecret       string
	AuditLog            string
}

type stateManifest struct {
	Version       int       `json:"version"`
	PublicBaseURL string    `json:"publicBaseUrl"`
	Issuer        string    `json:"issuer"`
	ClientID      string    `json:"clientId"`
	CreatedAt     time.Time `json:"createdAt"`
}

func resolveStatePaths(root string) statePaths {
	root = filepath.Clean(root)
	return statePaths{
		Root: root, Manifest: filepath.Join(root, "state.json"),
		IdentityDatabase:    filepath.Join(root, "identity", "tinyidp.sqlite"),
		ApplicationDatabase: filepath.Join(root, "application", "messages.sqlite"),
		TokenSecret:         filepath.Join(root, "secrets", "token.key"),
		SessionSecret:       filepath.Join(root, "secrets", "app-session.key"),
		AuditLog:            filepath.Join(root, "audit", "events.jsonl"),
	}
}

func initializeStateRoot(ctx context.Context, root, rawPublicBaseURL string, now time.Time) (stateManifest, error) {
	if ctx == nil {
		return stateManifest{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return stateManifest{}, err
	}
	publicBaseURL, err := normalizePublicBaseURL(rawPublicBaseURL)
	if err != nil {
		return stateManifest{}, err
	}
	paths := resolveStatePaths(root)
	if strings.TrimSpace(root) == "" || paths.Root == "." {
		return stateManifest{}, errors.New("state root is required")
	}
	for _, directory := range []string{
		paths.Root, filepath.Dir(paths.IdentityDatabase), filepath.Dir(paths.ApplicationDatabase),
		filepath.Dir(paths.TokenSecret), filepath.Dir(paths.AuditLog),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return stateManifest{}, errors.Wrapf(err, "create state directory %s", directory)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return stateManifest{}, errors.Wrapf(err, "protect state directory %s", directory)
		}
	}
	desired := stateManifest{
		Version: stateManifestVersion, PublicBaseURL: publicBaseURL,
		Issuer: publicBaseURL + issuerPath, ClientID: clientID,
	}
	if existing, err := readStateManifest(paths.Manifest); err == nil {
		if existing.Version != desired.Version || existing.PublicBaseURL != desired.PublicBaseURL ||
			existing.Issuer != desired.Issuer || existing.ClientID != desired.ClientID {
			return stateManifest{}, errors.New("initialized state conflicts with the requested application origin or client")
		}
		desired.CreatedAt = existing.CreatedAt
	} else if !os.IsNotExist(errors.Cause(err)) {
		return stateManifest{}, err
	}
	if desired.CreatedAt.IsZero() {
		desired.CreatedAt = now.UTC()
	}
	if _, err := loadOrCreateSecret(paths.TokenSecret, 32); err != nil {
		return stateManifest{}, errors.Wrap(err, "initialize token secret")
	}
	if _, err := loadOrCreateSecret(paths.SessionSecret, 32); err != nil {
		return stateManifest{}, errors.Wrap(err, "initialize app session secret")
	}
	if _, err := os.Stat(paths.Manifest); os.IsNotExist(err) {
		if err := writeJSONAtomic(paths.Manifest, desired); err != nil {
			return stateManifest{}, err
		}
	} else if err != nil {
		return stateManifest{}, errors.Wrap(err, "stat state manifest")
	}
	return desired, nil
}

func validateStateRoot(root string) (stateManifest, statePaths, error) {
	paths := resolveStatePaths(root)
	manifest, err := readStateManifest(paths.Manifest)
	if err != nil {
		return stateManifest{}, paths, err
	}
	if manifest.Version != stateManifestVersion || manifest.ClientID != clientID || manifest.Issuer != manifest.PublicBaseURL+issuerPath {
		return stateManifest{}, paths, errors.New("state manifest is incomplete or unsupported")
	}
	for label, file := range map[string]string{"token secret": paths.TokenSecret, "session secret": paths.SessionSecret} {
		info, err := os.Stat(file)
		if err != nil {
			return stateManifest{}, paths, errors.Wrapf(err, "%s is unavailable", label)
		}
		if info.Mode().Perm() != 0o600 || info.Size() != 32 {
			return stateManifest{}, paths, errors.Errorf("%s must be a 32-byte owner-only file", label)
		}
	}
	return manifest, paths, nil
}

func normalizePublicBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.Wrap(err, "parse public base URL")
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("public base URL must be an absolute HTTP(S) origin")
	}
	if parsed.Scheme == "http" {
		host := parsed.Hostname()
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", errors.New("plain HTTP is allowed only for loopback development origins")
		}
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func readStateManifest(file string) (stateManifest, error) {
	contents, err := os.ReadFile(file)
	if err != nil {
		return stateManifest{}, err
	}
	var manifest stateManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return stateManifest{}, errors.Wrap(err, "decode state manifest")
	}
	return manifest, nil
}

func loadOrCreateSecret(file string, size int) ([]byte, error) {
	if contents, err := os.ReadFile(file); err == nil {
		if len(contents) != size {
			return nil, errors.Errorf("secret %s has %d bytes, want %d", file, len(contents), size)
		}
		return contents, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	contents := make([]byte, size)
	if _, err := rand.Read(contents); err != nil {
		return nil, errors.Wrap(err, "generate secret")
	}
	if err := writeBytesAtomic(file, contents); err != nil {
		return nil, err
	}
	return contents, nil
}

func writeJSONAtomic(file string, value any) error {
	contents, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return errors.Wrap(err, "encode JSON")
	}
	return writeBytesAtomic(file, append(contents, '\n'))
}

func writeBytesAtomic(file string, contents []byte) error {
	temporary := file + ".tmp"
	handle, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.Wrap(err, "create temporary state file")
	}
	closed := false
	defer func() {
		if !closed {
			_ = handle.Close()
		}
		_ = os.Remove(temporary)
	}()
	if _, err := handle.Write(contents); err != nil {
		return errors.Wrap(err, "write temporary state file")
	}
	if err := handle.Sync(); err != nil {
		return errors.Wrap(err, "sync temporary state file")
	}
	if err := handle.Close(); err != nil {
		return errors.Wrap(err, "close temporary state file")
	}
	closed = true
	if err := os.Rename(temporary, file); err != nil {
		return errors.Wrap(err, "publish state file")
	}
	return nil
}
