package main

import (
	"context"
	stderrors "errors"
	"net/http"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type ServeInitializedCommand struct{ *cmds.CommandDescription }

type ServeInitializedSettings struct {
	StateRoot         string `glazed:"state-root"`
	Listen            string `glazed:"listen"`
	TLSCertificate    string `glazed:"tls-cert"`
	TLSKey            string `glazed:"tls-key"`
	Maintenance       string `glazed:"maintenance-interval"`
	ShutdownTimeout   string `glazed:"shutdown-timeout"`
	MaxRequestBytes   int    `glazed:"max-request-bytes"`
	ReadHeaderTimeout string `glazed:"read-header-timeout"`
	ReadTimeout       string `glazed:"read-timeout"`
	WriteTimeout      string `glazed:"write-timeout"`
	IdleTimeout       string `glazed:"idle-timeout"`
}

var _ cmds.BareCommand = (*ServeInitializedCommand)(nil)

func NewServeInitializedCommand() (*ServeInitializedCommand, error) {
	return &ServeInitializedCommand{CommandDescription: cmds.NewCommandDescription(
		"serve-initialized",
		cmds.WithShort("Serve the initialized persistent product over TLS"),
		cmds.WithLong(`Validate and open a completed state root, construct all
persistent identity/application/object services, load trusted routes, run
maintenance, and only then bind the TLS listener. This command does not trust
forwarded headers; deploy it directly or add an explicitly reviewed proxy mode.`),
		cmds.WithFlags(
			fields.New("state-root", fields.TypeString, fields.WithRequired(true)),
			fields.New("listen", fields.TypeString, fields.WithDefault(":8443")),
			fields.New("tls-cert", fields.TypeString, fields.WithRequired(true)),
			fields.New("tls-key", fields.TypeString, fields.WithRequired(true)),
			fields.New("maintenance-interval", fields.TypeString, fields.WithDefault("15m")),
			fields.New("shutdown-timeout", fields.TypeString, fields.WithDefault("20s")),
			fields.New("max-request-bytes", fields.TypeInteger, fields.WithDefault(1<<20)),
			fields.New("read-header-timeout", fields.TypeString, fields.WithDefault("5s")),
			fields.New("read-timeout", fields.TypeString, fields.WithDefault("15s")),
			fields.New("write-timeout", fields.TypeString, fields.WithDefault("30s")),
			fields.New("idle-timeout", fields.TypeString, fields.WithDefault("1m")),
		),
	)}, nil
}

func (c *ServeInitializedCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings ServeInitializedSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return errors.Wrap(err, "decode initialized serve settings")
	}
	maintenance, shutdown, readHeader, read, write, idle, err := parseInitializedDurations(settings)
	if err != nil {
		return err
	}
	if settings.MaxRequestBytes <= 0 {
		return errors.New("max-request-bytes must be positive")
	}
	app, err := NewInitializedApplication(ctx, settings.StateRoot)
	if err != nil {
		return err
	}
	defer func() { _ = app.Close(context.Background()) }()
	if err := app.Ready(ctx); err != nil {
		return errors.Wrap(err, "refuse listener while application is not ready")
	}
	server := &http.Server{
		Addr:              settings.Listen,
		Handler:           initializedHandler(app, int64(settings.MaxRequestBytes)),
		ReadHeaderTimeout: readHeader,
		ReadTimeout:       read,
		WriteTimeout:      write,
		IdleTimeout:       idle,
		MaxHeaderBytes:    1 << 20,
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Info().Str("listen", settings.Listen).Msg("tinyidp-xapp initialized TLS server started")
		if err := server.ListenAndServeTLS(settings.TLSCertificate, settings.TLSKey); err != nil && !stderrors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "serve initialized TLS")
		}
		return nil
	})
	group.Go(func() error {
		ticker := time.NewTicker(maintenance)
		defer ticker.Stop()
		for {
			select {
			case <-groupCtx.Done():
				return nil
			case <-ticker.C:
				if _, err := app.idp.RunMaintenance(groupCtx); err != nil && groupCtx.Err() == nil {
					log.Error().Err(err).Msg("initialized product maintenance failed; readiness degraded")
				}
			}
		}
	})
	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdown)
		defer cancel()
		return errors.Wrap(server.Shutdown(shutdownCtx), "shutdown initialized TLS server")
	})
	return group.Wait()
}

func initializedHandler(app *DevelopmentApplication, maxRequestBytes int64) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"status\":\"live\"}\n"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := app.Ready(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("{\"status\":\"not_ready\"}\n"))
			return
		}
		_, _ = w.Write([]byte("{\"status\":\"ready\"}\n"))
	})
	mux.Handle("/", http.MaxBytesHandler(app.Handler(), maxRequestBytes))
	return mux
}

func parseInitializedDurations(settings ServeInitializedSettings) (time.Duration, time.Duration, time.Duration, time.Duration, time.Duration, time.Duration, error) {
	raw := []string{settings.Maintenance, settings.ShutdownTimeout, settings.ReadHeaderTimeout, settings.ReadTimeout, settings.WriteTimeout, settings.IdleTimeout}
	parsed := make([]time.Duration, len(raw))
	for index, value := range raw {
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return 0, 0, 0, 0, 0, 0, errors.Errorf("invalid positive duration %q", value)
		}
		parsed[index] = duration
	}
	return parsed[0], parsed[1], parsed[2], parsed[3], parsed[4], parsed[5], nil
}
