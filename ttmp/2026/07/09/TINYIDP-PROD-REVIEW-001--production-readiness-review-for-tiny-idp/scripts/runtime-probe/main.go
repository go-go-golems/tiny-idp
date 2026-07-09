package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	runtimemetrics "runtime/metrics"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/manuel/tinyidp/internal/admin"
	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

type config struct {
	requests     int
	concurrency  int
	output       string
	cpuProfile   string
	heapProfile  string
	logLevel     string
	requestLimit time.Duration
}

type event struct {
	Type       string             `json:"type"`
	At         time.Time          `json:"at"`
	Phase      string             `json:"phase,omitempty"`
	Name       string             `json:"name,omitempty"`
	Method     string             `json:"method,omitempty"`
	Path       string             `json:"path,omitempty"`
	Status     int                `json:"status,omitempty"`
	Bytes      int                `json:"bytes,omitempty"`
	DurationUS int64              `json:"duration_us,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
	DB         *dbSnapshot        `json:"db,omitempty"`
	AuditCount int                `json:"audit_count,omitempty"`
	Error      string             `json:"error,omitempty"`
}

type dbSnapshot struct {
	OpenConnections int   `json:"open_connections"`
	InUse           int   `json:"in_use"`
	Idle            int   `json:"idle"`
	WaitCount       int64 `json:"wait_count"`
	WaitDurationUS  int64 `json:"wait_duration_us"`
	MaxIdleClosed   int64 `json:"max_idle_closed"`
	MaxLifetimeDone int64 `json:"max_lifetime_closed"`
}

type emitter struct {
	mu  sync.Mutex
	enc *json.Encoder
}

func (e *emitter) emit(value event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.enc.Encode(value)
}

type response struct {
	status int
	header http.Header
	body   []byte
	cookie []*http.Cookie
}

func main() {
	cfg := config{}
	flag.IntVar(&cfg.requests, "requests", 40, "number of bounded concurrent read requests")
	flag.IntVar(&cfg.concurrency, "concurrency", 4, "number of request workers")
	flag.StringVar(&cfg.output, "output", "-", "NDJSON output path or - for stdout")
	flag.StringVar(&cfg.cpuProfile, "cpu-profile", "", "optional CPU profile path")
	flag.StringVar(&cfg.heapProfile, "heap-profile", "", "optional heap profile path")
	flag.StringVar(&cfg.logLevel, "log-level", "info", "zerolog level")
	flag.DurationVar(&cfg.requestLimit, "request-timeout", 5*time.Second, "per-request client timeout")
	flag.Parse()

	level, err := zerolog.ParseLevel(cfg.logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --log-level: %v\n", err)
		os.Exit(2)
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	if cfg.requests < 0 || cfg.concurrency < 1 {
		log.Fatal().Msg("--requests must be non-negative and --concurrency must be positive")
	}
	if err := run(context.Background(), cfg); err != nil {
		log.Fatal().Err(err).Msg("runtime probe failed")
	}
}

func run(ctx context.Context, cfg config) error {
	out, closeOutput, err := outputWriter(cfg.output)
	if err != nil {
		return err
	}
	defer closeOutput()
	emit := &emitter{enc: json.NewEncoder(out)}

	stopCPU, err := startCPUProfile(cfg.cpuProfile)
	if err != nil {
		return err
	}
	defer stopCPU()

	dir, err := os.MkdirTemp("", "tinyidp-runtime-probe-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Warn().Err(err).Str("path", dir).Msg("remove runtime probe directory")
		}
	}()
	store, err := sqlitestore.Open(filepath.Join(dir, "runtime.db"))
	if err != nil {
		return fmt.Errorf("open SQLite: %w", err)
	}
	defer store.Close()

	auditSink := idp.NewMemorySink()
	adminService, err := admin.NewService(store, admin.Options{Audit: auditSink})
	if err != nil {
		return fmt.Errorf("create admin service: %w", err)
	}
	_, _, err = adminService.CreateClient(ctx, admin.CreateClientRequest{
		ID:              "runtime-spa",
		Public:          true,
		RequirePKCE:     true,
		RedirectURIs:    []string{"http://127.0.0.1/callback"},
		AllowedScopes:   []string{"openid", "profile", "email", "offline_access"},
		AccessTokenTTL:  time.Hour,
		IDTokenTTL:      time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	_, err = adminService.CreateUser(ctx, admin.CreateUserRequest{
		Login:             "alice",
		Password:          []byte("correct horse battery staple"),
		Email:             "alice@example.test",
		EmailVerified:     true,
		Name:              "Alice Runtime",
		PreferredUsername: "alice",
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	if _, err := adminService.GenerateSigningKey(ctx, "runtime-key-1", true); err != nil {
		return fmt.Errorf("create signing key: %w", err)
	}

	provider, err := embeddedidp.New(embeddedidp.Options{
		Issuer:      "https://id.example.test",
		Mode:        embeddedidp.ProductionMode,
		Store:       store,
		Cookie:      embeddedidp.CookieConfig{Secure: true, SameSite: "Lax"},
		Token:       embeddedidp.TokenConfig{SecretKey: []byte("runtime-probe-secret-key-32-bytes-minimum")},
		Audit:       auditSink,
		RateLimiter: fositeadapter.NewFixedWindowRateLimiter(10_000, time.Minute),
	})
	if err != nil {
		return fmt.Errorf("create production provider: %w", err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := &http.Client{Timeout: cfg.requestLimit, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	emitSnapshot(emit, "before", store.SQLDB())
	tokens, err := runAuthorizationCodeFlow(ctx, emit, client, server.URL)
	if err != nil {
		return err
	}
	if err := runLoad(ctx, emit, client, server.URL, tokens.accessToken, cfg.requests, cfg.concurrency); err != nil {
		return err
	}
	client.CloseIdleConnections()
	time.Sleep(25 * time.Millisecond)
	emitSnapshot(emit, "after", store.SQLDB())
	if err := emit.emit(event{Type: "summary", At: time.Now().UTC(), AuditCount: len(auditSink.Events())}); err != nil {
		return err
	}
	if err := writeHeapProfile(cfg.heapProfile); err != nil {
		return err
	}
	log.Info().Int("requests", cfg.requests).Int("concurrency", cfg.concurrency).Int("audit_events", len(auditSink.Events())).Msg("runtime probe complete")
	return nil
}

type flowTokens struct {
	accessToken  string
	refreshToken string
}

func runAuthorizationCodeFlow(ctx context.Context, emit *emitter, client *http.Client, baseURL string) (flowTokens, error) {
	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {"runtime-spa"},
		"redirect_uri":          {"http://127.0.0.1/callback"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {"runtime-state-1234567890"},
		"nonce":                 {"runtime-nonce-1234567890"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}

	interaction, err := doRequest(ctx, emit, client, "authorize_get", http.MethodGet, baseURL+"/authorize?"+form.Encode(), "", nil)
	if err != nil {
		return flowTokens{}, err
	}
	if interaction.status != http.StatusOK {
		return flowTokens{}, fmt.Errorf("authorize GET status %d: %s", interaction.status, interaction.body)
	}
	csrf := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindSubmatch(interaction.body)
	if len(csrf) != 2 {
		return flowTokens{}, fmt.Errorf("csrf token not found")
	}
	var csrfCookie *http.Cookie
	for _, cookie := range interaction.cookie {
		if cookie.Name == "tinyidp_csrf" {
			csrfCookie = cookie
		}
	}
	if csrfCookie == nil {
		return flowTokens{}, fmt.Errorf("csrf cookie not found")
	}
	form.Set("csrf_token", string(csrf[1]))
	form.Set("login", "alice")
	form.Set("password", "correct horse battery staple")
	form.Set("consent_approved", "true")
	authorized, err := doRequest(ctx, emit, client, "authorize_post", http.MethodPost, baseURL+"/authorize", form.Encode(), []*http.Cookie{csrfCookie})
	if err != nil {
		return flowTokens{}, err
	}
	if authorized.status != http.StatusFound && authorized.status != http.StatusSeeOther {
		return flowTokens{}, fmt.Errorf("authorize POST status %d: %s", authorized.status, authorized.body)
	}
	callback, err := url.Parse(authorized.header.Get("Location"))
	if err != nil {
		return flowTokens{}, err
	}
	code := callback.Query().Get("code")
	if code == "" {
		return flowTokens{}, fmt.Errorf("authorization code missing: %s", callback)
	}

	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"runtime-spa"},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1/callback"},
		"code_verifier": {verifier},
	}
	tokenResponse, err := doRequest(ctx, emit, client, "token_exchange", http.MethodPost, baseURL+"/token", tokenForm.Encode(), nil)
	if err != nil {
		return flowTokens{}, err
	}
	if tokenResponse.status != http.StatusOK {
		return flowTokens{}, fmt.Errorf("token status %d: %s", tokenResponse.status, tokenResponse.body)
	}
	var tokenBody map[string]any
	if err := json.Unmarshal(tokenResponse.body, &tokenBody); err != nil {
		return flowTokens{}, err
	}
	accessToken, _ := tokenBody["access_token"].(string)
	refreshToken, _ := tokenBody["refresh_token"].(string)
	if accessToken == "" || refreshToken == "" {
		return flowTokens{}, fmt.Errorf("token response missing access or refresh token")
	}

	userinfo, err := doBearerRequest(ctx, emit, client, "userinfo", baseURL+"/userinfo", accessToken)
	if err != nil {
		return flowTokens{}, err
	}
	if userinfo.status != http.StatusOK {
		return flowTokens{}, fmt.Errorf("userinfo status %d: %s", userinfo.status, userinfo.body)
	}

	refreshForm := url.Values{"grant_type": {"refresh_token"}, "client_id": {"runtime-spa"}, "refresh_token": {refreshToken}}
	refreshed, err := doRequest(ctx, emit, client, "token_refresh", http.MethodPost, baseURL+"/token", refreshForm.Encode(), nil)
	if err != nil {
		return flowTokens{}, err
	}
	if refreshed.status != http.StatusOK {
		return flowTokens{}, fmt.Errorf("refresh status %d: %s", refreshed.status, refreshed.body)
	}
	var refreshedBody map[string]any
	if err := json.Unmarshal(refreshed.body, &refreshedBody); err != nil {
		return flowTokens{}, err
	}
	if next, _ := refreshedBody["access_token"].(string); next != "" {
		accessToken = next
	}
	if next, _ := refreshedBody["refresh_token"].(string); next != "" {
		refreshToken = next
	}
	return flowTokens{accessToken: accessToken, refreshToken: refreshToken}, nil
}

func runLoad(ctx context.Context, emit *emitter, client *http.Client, baseURL, accessToken string, requests, concurrency int) error {
	if requests == 0 {
		return nil
	}
	paths := []string{"/.well-known/openid-configuration", "/jwks", "/readyz", "/userinfo"}
	jobs := make(chan int)
	group, groupCtx := errgroup.WithContext(ctx)
	for worker := 0; worker < concurrency; worker++ {
		group.Go(func() error {
			for index := range jobs {
				path := paths[index%len(paths)]
				var result response
				var err error
				if path == "/userinfo" {
					result, err = doBearerRequest(groupCtx, emit, client, "load_userinfo", baseURL+path, accessToken)
				} else {
					result, err = doRequest(groupCtx, emit, client, "load"+strings.ReplaceAll(path, "/", "_"), http.MethodGet, baseURL+path, "", nil)
				}
				if err != nil {
					return err
				}
				if result.status != http.StatusOK {
					return fmt.Errorf("load request %s returned %d", path, result.status)
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(jobs)
		for i := 0; i < requests; i++ {
			select {
			case jobs <- i:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})
	return group.Wait()
}

func doBearerRequest(ctx context.Context, emit *emitter, client *http.Client, name, rawURL, token string) (response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return response{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return execute(emit, client, name, req)
}

func doRequest(ctx context.Context, emit *emitter, client *http.Client, name, method, rawURL, body string, cookies []*http.Cookie) (response, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, reader)
	if err != nil {
		return response{}, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	return execute(emit, client, name, req)
}

func execute(emit *emitter, client *http.Client, name string, req *http.Request) (response, error) {
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	entry := event{Type: "request", At: time.Now().UTC(), Name: name, Method: req.Method, Path: req.URL.Path, DurationUS: duration.Microseconds()}
	if err != nil {
		entry.Error = err.Error()
		_ = emit.emit(entry)
		return response{}, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	entry.Status = resp.StatusCode
	entry.Bytes = len(body)
	if readErr != nil {
		entry.Error = readErr.Error()
	}
	if err := emit.emit(entry); err != nil {
		return response{}, err
	}
	return response{status: resp.StatusCode, header: resp.Header.Clone(), body: body, cookie: resp.Cookies()}, readErr
}

func emitSnapshot(emit *emitter, phase string, db *sql.DB) {
	_ = emit.emit(event{Type: "runtime", At: time.Now().UTC(), Phase: phase, Metrics: readRuntimeMetrics()})
	stats := db.Stats()
	_ = emit.emit(event{Type: "database", At: time.Now().UTC(), Phase: phase, DB: &dbSnapshot{
		OpenConnections: stats.OpenConnections,
		InUse:           stats.InUse,
		Idle:            stats.Idle,
		WaitCount:       stats.WaitCount,
		WaitDurationUS:  stats.WaitDuration.Microseconds(),
		MaxIdleClosed:   stats.MaxIdleClosed,
		MaxLifetimeDone: stats.MaxLifetimeClosed,
	}})
}

func readRuntimeMetrics() map[string]float64 {
	names := []string{
		"/cpu/classes/gc/total:cpu-seconds",
		"/gc/cycles/total:gc-cycles",
		"/gc/heap/allocs:bytes",
		"/gc/heap/live:bytes",
		"/gc/heap/objects:objects",
		"/memory/classes/heap/objects:bytes",
		"/sched/goroutines:goroutines",
	}
	samples := make([]runtimemetrics.Sample, len(names))
	for index, name := range names {
		samples[index].Name = name
	}
	runtimemetrics.Read(samples)
	out := make(map[string]float64, len(samples))
	for _, sample := range samples {
		switch sample.Value.Kind() {
		case runtimemetrics.KindUint64:
			out[sample.Name] = float64(sample.Value.Uint64())
		case runtimemetrics.KindFloat64:
			out[sample.Name] = sample.Value.Float64()
		case runtimemetrics.KindBad, runtimemetrics.KindFloat64Histogram:
			continue
		}
	}
	return out
}

func outputWriter(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() { _ = file.Close() }, nil
}

func startCPUProfile(path string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		pprof.StopCPUProfile()
		_ = file.Close()
	}, nil
}

func writeHeapProfile(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	runtime.GC()
	return pprof.WriteHeapProfile(file)
}
