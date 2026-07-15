package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	external "github.com/manuel/tinyidp/examples/tinyidp-external-message-desk"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func main() {
	var stateRoot, issuer, listen, seedFile, level string
	flag.StringVar(&stateRoot, "state-root", "", "owner-only standalone IdP state directory")
	flag.StringVar(&issuer, "issuer", "", "canonical browser-visible issuer URL")
	flag.StringVar(&listen, "listen", ":8081", "HTTP listen address")
	flag.StringVar(&seedFile, "seed-file", "", "operator-mounted JSON seed manifest")
	flag.StringVar(&level, "log-level", "info", "zerolog level")
	flag.Parse()
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid log level")
	}
	zerolog.SetGlobalLevel(parsedLevel)
	if stateRoot == "" || issuer == "" || seedFile == "" {
		log.Fatal().Msg("--state-root, --issuer, and --seed-file are required")
	}
	ctx := context.Background()
	seed, err := readSeed(seedFile)
	if err != nil {
		log.Fatal().Err(err).Msg("read seed manifest")
	}
	if err := os.MkdirAll(stateRoot, 0o700); err != nil {
		log.Fatal().Err(err).Msg("create state root")
	}
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(stateRoot, "tinyidp.sqlite")))
	if err != nil {
		log.Fatal().Err(err).Msg("open identity store")
	}
	defer store.Close()
	secret, err := loadOrCreateSecret(filepath.Join(stateRoot, "token.key"))
	if err != nil {
		log.Fatal().Err(err).Msg("load token secret")
	}
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		log.Fatal().Err(err).Msg("create account service")
	}
	provider, err := external.NewStandaloneIDP(ctx, external.StandaloneIDPConfig{Issuer: issuer, Mode: embeddedidp.DevMode, Store: store, Accounts: accounts, Seed: seed, TokenSecret: secret})
	if err != nil {
		log.Fatal().Err(err).Msg("initialize standalone IdP")
	}
	defer provider.Close(context.Background())
	mux := http.NewServeMux()
	mux.Handle("/", provider.Handler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !provider.Readiness(r.Context()).Ready {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{Addr: listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: time.Minute}
	log.Info().Str("issuer", issuer).Str("listen", listen).Msg("standalone tiny-idp listening")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("serve standalone IdP")
	}
}

func readSeed(file string) (external.SeedManifest, error) {
	var seed external.SeedManifest
	body, err := os.ReadFile(file)
	if err != nil {
		return seed, err
	}
	if err := json.Unmarshal(body, &seed); err != nil {
		return seed, errors.Wrap(err, "decode JSON seed manifest")
	}
	return seed, nil
}

func loadOrCreateSecret(file string) ([]byte, error) {
	if b, err := os.ReadFile(file); err == nil {
		if len(b) != 32 {
			return nil, errors.New("token secret must be 32 bytes")
		}
		return b, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	if err := os.WriteFile(file, b, 0o600); err != nil {
		return nil, err
	}
	return b, nil
}
