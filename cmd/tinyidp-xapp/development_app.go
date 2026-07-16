package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	xgojaapp "github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	httpprovider "github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http"
	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
	durableobjectsprovider "github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects"
	"github.com/manuel/tinyidp/cmd/tinyidp-xapp/internal/loginui"
	"github.com/manuel/tinyidp/cmd/tinyidp-xapp/internal/resourceauth"
	"github.com/manuel/tinyidp/cmd/tinyidp-xapp/internal/xgojaruntime"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	developmentClientID   = "tinyidp-xapp"
	deviceClientID        = "tinyidp-xapp-cli"
	resourceClientID      = "tinyidp-xapp-api"
	userStateNamespace    = "USER_STATE"
	deviceAPIAudiencePath = "/api"
)

type DevelopmentApplicationConfig struct {
	PublicBaseURL  string
	StateRoot      string
	Login          string
	Password       string
	SecondLogin    string
	SecondPassword string
}

type DevelopmentApplication struct {
	handler       http.Handler
	publicBaseURL string
	runtime       *engine.Runtime
	idp           *embeddedidp.Provider
	identityStore *sqlitestore.Store
	objects       *durableobjects.Server
	auth          *hostauth.Services
	apiAuth       *resourceauth.Authenticator
	apiAudit      idp.Sink
	oidc          *observedRoundTripper
	loginUI       *loginui.Renderer
	extras        []func(context.Context) error
}

//nolint:nonamedreturns // retErr lets deferred construction cleanup observe any initialization failure.
func NewDevelopmentApplication(ctx context.Context, cfg DevelopmentApplicationConfig) (_ *DevelopmentApplication, retErr error) {
	if ctx == nil {
		return nil, errors.New("development application context is required")
	}
	if cfg.PublicBaseURL == "" || cfg.StateRoot == "" || cfg.Login == "" || cfg.Password == "" {
		return nil, errors.New("public base URL, state root, login, and password are required")
	}
	if (cfg.SecondLogin == "") != (cfg.SecondPassword == "") {
		return nil, errors.New("second development login and password must be provided together")
	}
	if err := os.MkdirAll(cfg.StateRoot, 0o700); err != nil {
		return nil, errors.Wrap(err, "create development state root")
	}

	issuer := cfg.PublicBaseURL + "/idp"
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(cfg.StateRoot, "identity", "development.sqlite")))
	if err != nil {
		return nil, errors.Wrap(err, "open development identity store")
	}
	storeOwnedByApp := false
	defer func() {
		if retErr != nil && !storeOwnedByApp {
			_ = store.Close()
		}
	}()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "create development password service")
	}
	if err := seedDevelopmentUser(ctx, store, accounts, developmentUser{ID: "dev-alice", Subject: "dev-alice-subject", Login: cfg.Login, Password: cfg.Password, Name: "Alice", Email: "alice@example.test"}); err != nil {
		return nil, err
	}
	if cfg.SecondLogin != "" {
		if err := seedDevelopmentUser(ctx, store, accounts, developmentUser{ID: "dev-bob", Subject: "dev-bob-subject", Login: cfg.SecondLogin, Password: cfg.SecondPassword, Name: "Bob", Email: "bob@example.test"}); err != nil {
			return nil, err
		}
	}
	resourceSecretKey, err := loadOrCreateKey(filepath.Join(cfg.StateRoot, "secrets", "resource-client.key"))
	if err != nil {
		return nil, errors.Wrap(err, "load development resource-client secret")
	}
	defer zeroBytes(resourceSecretKey)
	audience := apiAudience(cfg.PublicBaseURL)
	deviceClient := embeddedidp.DeviceClient(deviceClientID, []string{"openid", "bbs.read", "bbs.post.create"})
	deviceClient.Client.AllowedAudiences = []string{audience}
	if _, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{
		Mode: embeddedidp.DevMode,
		Clients: []embeddedidp.ClientSpec{
			embeddedidp.BrowserClient(developmentClientID, []string{cfg.PublicBaseURL + "/auth/callback"}, []string{cfg.PublicBaseURL + "/"}, []string{"openid", "profile", "email"}),
			deviceClient,
		},
		SigningKeyID: "xapp-dev-signing-key",
	}); err != nil {
		return nil, errors.Wrap(err, "bootstrap development identity provider")
	}
	if err := reconcileResourceClient(ctx, store, embeddedidp.DevMode, resourceSecretKey, audience); err != nil {
		return nil, errors.Wrap(err, "bootstrap development resource client")
	}
	tokenSecret, err := randomKey(32)
	if err != nil {
		return nil, err
	}
	interactionUI, err := loginui.New(loginui.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "create development interaction renderer")
	}
	idpProvider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer:        issuer,
		Mode:          embeddedidp.DevMode,
		Store:         store,
		Authenticator: accounts,
		Cookie: embeddedidp.CookieConfig{
			SessionName: "xapp_idp_session",
			CSRFName:    "xapp_idp_csrf",
		},
		Token: embeddedidp.TokenConfig{SecretKey: tokenSecret},
		UI:    embeddedidp.UIConfig{Renderer: interactionUI},
	})
	if err != nil {
		return nil, errors.Wrap(err, "create embedded development IdP")
	}
	app := &DevelopmentApplication{
		idp: idpProvider, identityStore: store, loginUI: interactionUI, publicBaseURL: cfg.PublicBaseURL,
		apiAudit: idp.NoopSink{},
		extras:   []func(context.Context) error{func(context.Context) error { return store.Close() }},
	}
	storeOwnedByApp = true
	defer func() {
		if retErr != nil {
			_ = app.Close(context.Background())
		}
	}()

	transport, err := embeddedidp.NewInProcessIssuerTransport(issuer, idpProvider.Handler(), embeddedidp.InProcessTransportOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "create in-process issuer transport")
	}
	observedTransport := &observedRoundTripper{base: transport}
	app.oidc = observedTransport
	resourceSecret := []byte(resourceClientSecret(resourceSecretKey))
	defer zeroBytes(resourceSecret)
	apiAuth, err := resourceauth.New(ctx, resourceauth.Config{
		IssuerURL: issuer, ClientID: resourceClientID, ClientSecret: resourceSecret, Audience: audience,
		HTTPClient: &http.Client{Transport: observedTransport, Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, errors.Wrap(err, "build development API resource authentication")
	}
	app.apiAuth = apiAuth
	authFactory := hostauth.NewServiceFactory(hostauth.BuilderOptions{
		Config: hostauth.Config{
			Mode: hostauth.ModeOIDC,
			Session: hostauth.SessionConfig{Cookie: hostauth.CookieConfig{
				AllowInsecureHTTP: true,
				Name:              "xapp_session",
				Path:              "/",
				SameSite:          "lax",
			}},
			OIDC: hostauth.OIDCConfig{
				IssuerURL:      issuer,
				ClientID:       developmentClientID,
				PublicBaseURL:  cfg.PublicBaseURL,
				Scopes:         []string{"profile", "email"},
				AfterLoginURL:  "/",
				AfterLogoutURL: "/",
			},
		},
		OIDCHTTPClient: &http.Client{Transport: observedTransport, Timeout: 10 * time.Second},
	})
	authServices, err := authFactory.BuildHostAuthServices(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build application authentication services")
	}
	app.auth = authServices
	if err := composeApplication(ctx, app, authFactory, cfg.StateRoot); err != nil {
		return nil, err
	}
	return app, nil
}

type developmentUser struct {
	ID       string
	Subject  string
	Login    string
	Password string
	Name     string
	Email    string
}

func seedDevelopmentUser(ctx context.Context, store *sqlitestore.Store, accounts *idpaccounts.Service, seed developmentUser) error {
	_, err := accounts.Create(ctx, idpaccounts.CreateRequest{ID: seed.ID, Subject: seed.Subject, Login: seed.Login, Password: []byte(seed.Password), Email: seed.Email, EmailVerified: true, Name: seed.Name, PreferredUsername: seed.Login})
	if err == nil {
		return nil
	}
	if !errors.Is(err, idpstore.ErrDuplicate) {
		return errors.Wrap(err, "seed development user")
	}
	existing, getErr := store.GetUserByLogin(ctx, seed.Login)
	if getErr != nil {
		return errors.Wrap(getErr, "reconcile development user")
	}
	if existing.ID != seed.ID || existing.Sub != seed.Subject || existing.Email != seed.Email || existing.Name != seed.Name {
		return errors.Errorf("development user %q conflicts with persisted identity state", seed.Login)
	}
	if _, authErr := accounts.AuthenticatePassword(ctx, seed.Login, seed.Password, idp.LoginMetadata{ClientID: developmentClientID}); authErr != nil {
		return errors.Errorf("development user %q password conflicts with persisted identity state", seed.Login)
	}
	return nil
}

func composeApplication(ctx context.Context, app *DevelopmentApplication, authFactory hostauth.ServiceFactory, stateRoot string) error {
	if app == nil || app.idp == nil || app.auth == nil || app.apiAuth == nil || app.loginUI == nil {
		return errors.New("identity and application auth services are required")
	}
	httpHost := gojahttp.NewHost(gojahttp.HostOptions{
		Auth:            app.auth.AuthOptions,
		Sessions:        gojahttp.SessionOptions{Disabled: true},
		RejectRawRoutes: true,
	})
	var configureErr error
	bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{ConfigureServices: func(services *xgojaapp.HostServices) {
		if configureErr != nil {
			return
		}
		configureErr = services.SetHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{Host: httpHost, OwnsListen: false})
		if configureErr != nil {
			return
		}
		configureErr = services.SetHostService(hostauth.ServiceFactoryKey, authFactory)
		if configureErr != nil {
			return
		}
		app.objects, configureErr = newDevelopmentObjectServer(ctx, services.Assets, stateRoot)
		if configureErr != nil {
			return
		}
		bindingKey, keyErr := loadOrCreateKey(filepath.Join(stateRoot, "secrets", "object-binding.key"))
		if keyErr != nil {
			configureErr = keyErr
			return
		}
		bound, boundErr := durableobjects.NewBoundDispatcher(app.objects.Manager, bindingKey, []string{userStateNamespace})
		if boundErr != nil {
			configureErr = boundErr
			return
		}
		configureErr = services.SetHostService(durableobjectsprovider.HostServiceKey, durableobjectsprovider.GatewayService{Manager: app.objects.Manager, Handler: app.objects.Handler, EnableRawGateway: false})
		if configureErr != nil {
			return
		}
		configureErr = services.SetHostService(durableobjectsprovider.BoundDispatcherHostServiceKey, durableobjectsprovider.BoundDispatcherService{
			Dispatcher: bound,
			ActorID: func(actorCtx context.Context) (string, error) {
				actor, ok := gojahttp.ActorFromContext(actorCtx)
				if !ok || actor.ID == "" {
					return "", errors.New("authenticated actor is unavailable")
				}
				return actor.ID, nil
			},
		})
	}})
	if err != nil {
		return errors.Wrap(err, "create generated xgoja bundle")
	}
	if configureErr != nil {
		return errors.Wrap(configureErr, "configure generated host services")
	}
	runtime, err := bundle.NewRuntime(ctx)
	if err != nil {
		return errors.Wrap(err, "create generated xgoja runtime")
	}
	app.runtime = runtime
	if err := loadApplicationRoutes(runtime, bundle.Host.EmbeddedJSVerbs); err != nil {
		return errors.Wrap(err, "load trusted application routes")
	}

	mux := http.NewServeMux()
	mux.Handle("/idp/", app.idp.Handler())
	mux.Handle("GET /static/tinyidp/", app.loginUI.AssetsHandler())
	mux.Handle("/api/device/", newDeviceAPIHandler(app.apiAuth, app.objects.Manager, app.apiAudit))
	for _, native := range app.auth.NativeHandlers {
		mux.Handle(native.Method+" "+native.Path, native.Handler)
	}
	mux.Handle("/", httpHost)
	app.handler = mux
	return nil
}

func (a *DevelopmentApplication) Handler() http.Handler {
	if a == nil || a.handler == nil {
		return http.NotFoundHandler()
	}
	return a.handler
}

func (a *DevelopmentApplication) Close(ctx context.Context) error {
	if a == nil {
		return nil
	}
	var first error
	for _, closeResource := range []func(context.Context) error{
		func(closeCtx context.Context) error {
			if a.runtime == nil {
				return nil
			}
			return a.runtime.Close(closeCtx)
		},
		func(closeCtx context.Context) error {
			if a.objects == nil {
				return nil
			}
			return a.objects.Close(closeCtx)
		},
		func(closeCtx context.Context) error {
			if a.auth == nil {
				return nil
			}
			return a.auth.Close(closeCtx)
		},
		func(closeCtx context.Context) error {
			if a.idp == nil {
				return nil
			}
			return a.idp.Close(closeCtx)
		},
	} {
		if err := closeResource(ctx); err != nil && first == nil {
			first = err
		}
	}
	for _, closeResource := range a.extras {
		if err := closeResource(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func newDevelopmentObjectServer(ctx context.Context, assets *xgojaapp.AssetStore, stateRoot string) (*durableobjects.Server, error) {
	if assets == nil {
		return nil, errors.New("generated asset store is required")
	}
	assetFS, root, ok := assets.ResolveAsset("object-bundle")
	if !ok {
		return nil, errors.New("embedded object-bundle asset is unavailable")
	}
	source, err := fs.ReadFile(assetFS, path.Join(root, "objects.js"))
	if err != nil {
		return nil, errors.Wrap(err, "read embedded object bundle")
	}
	objectRoot := filepath.Join(stateRoot, "objects")
	if err := os.MkdirAll(objectRoot, 0o700); err != nil {
		return nil, errors.Wrap(err, "create object storage root")
	}
	return durableobjects.NewServer(ctx, durableobjects.ServerOptions{
		BundleSource:    string(source),
		StorageRoot:     objectRoot,
		CPUTimeout:      2 * time.Second,
		IdleTimeout:     time.Minute,
		AlarmInterval:   time.Second,
		IdleInterval:    time.Minute,
		MaxRequestBytes: 64 * 1024,
	})
}

func loadApplicationRoutes(runtime *engine.Runtime, sourceFS fs.FS) error {
	if runtime == nil || runtime.VM == nil {
		return errors.New("xgoja runtime is required")
	}
	source, err := fs.ReadFile(sourceFS, "xgoja_embed/jsverbs/application_routes/site.js")
	if err != nil {
		return errors.Wrap(err, "read embedded route source")
	}
	vm := runtime.VM
	module := vm.NewObject()
	exportsObject := vm.NewObject()
	if err := module.Set("exports", exportsObject); err != nil {
		return err
	}
	if err := vm.Set("module", module); err != nil {
		return err
	}
	if err := vm.Set("exports", exportsObject); err != nil {
		return err
	}
	metadataNoop := func(goja.FunctionCall) goja.Value { return goja.Undefined() }
	if err := vm.Set("__package__", metadataNoop); err != nil {
		return err
	}
	if err := vm.Set("__verb__", metadataNoop); err != nil {
		return err
	}
	if _, err := vm.RunString(string(source)); err != nil {
		return errors.Wrap(err, "evaluate route source")
	}
	loadedExports := module.Get("exports").ToObject(vm)
	site, ok := goja.AssertFunction(loadedExports.Get("site"))
	if !ok {
		return errors.New("route source does not export site()")
	}
	if _, err := site(goja.Undefined()); err != nil {
		return errors.Wrap(err, "invoke site route registration")
	}
	return nil
}

func randomKey(size int) ([]byte, error) {
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, errors.Wrap(err, "read cryptographic randomness")
	}
	return key, nil
}

func apiAudience(publicBaseURL string) string {
	return strings.TrimRight(publicBaseURL, "/") + deviceAPIAudiencePath
}

func resourceClientSecret(secretKey []byte) string {
	return base64.RawURLEncoding.EncodeToString(secretKey)
}

func reconcileResourceClient(ctx context.Context, store idpstore.Store, mode idpstore.Mode, secretKey []byte, audience string) error {
	if ctx == nil || store == nil || len(secretKey) == 0 || audience == "" {
		return errors.New("resource-client context, store, secret, and audience are required")
	}
	secret := resourceClientSecret(secretKey)
	desired := idpstore.Client{
		ID: resourceClientID, Public: false, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode},
		AllowedAudiences: []string{audience}, CanIntrospect: true,
		AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour,
	}
	existing, err := store.GetClient(ctx, resourceClientID)
	if errors.Is(err, idpstore.ErrNotFound) {
		desired.SecretHash, err = bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			return errors.Wrap(err, "hash resource-client secret")
		}
		now := time.Now().UTC()
		desired.CreatedAt, desired.UpdatedAt = now, now
		if err := desired.Validate(mode); err != nil {
			return errors.Wrap(err, "validate resource client")
		}
		return errors.Wrap(store.PutClient(ctx, desired), "create resource client")
	}
	if err != nil {
		return errors.Wrap(err, "load resource client")
	}
	if existing.ID != desired.ID || existing.Public || !existing.CanIntrospect || !equalStrings(existing.AllowedAudiences, desired.AllowedAudiences) || !equalStrings(existing.AllowedGrantTypes, desired.AllowedGrantTypes) || existing.Disabled {
		return errors.New("existing resource client conflicts with xapp API security configuration")
	}
	if bcrypt.CompareHashAndPassword(existing.SecretHash, []byte(secret)) != nil {
		return errors.New("existing resource client secret conflicts with owner-only state")
	}
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func zeroBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

func loadOrCreateKey(file string) ([]byte, error) {
	if key, err := os.ReadFile(file); err == nil {
		if len(key) != 32 {
			return nil, fmt.Errorf("binding key %s has %d bytes, want 32", file, len(key))
		}
		info, statErr := os.Stat(file)
		if statErr != nil {
			return nil, errors.Wrap(statErr, "stat binding key")
		}
		if info.Mode().Perm() != 0o600 {
			return nil, fmt.Errorf("binding key %s permissions are %#o, want 0600", file, info.Mode().Perm())
		}
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "read binding key")
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return nil, errors.Wrap(err, "create binding key directory")
	}
	key, err := randomKey(32)
	if err != nil {
		return nil, err
	}
	handle, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return loadOrCreateKey(file)
	}
	if err != nil {
		return nil, errors.Wrap(err, "create binding key")
	}
	written, err := handle.Write(key)
	if err != nil || written != len(key) {
		_ = handle.Close()
		_ = os.Remove(file)
		if err != nil {
			return nil, errors.Wrap(err, "write binding key")
		}
		return nil, errors.New("write binding key: short write")
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		_ = os.Remove(file)
		return nil, errors.Wrap(err, "sync binding key")
	}
	if err := handle.Close(); err != nil {
		return nil, errors.Wrap(err, "close binding key")
	}
	return key, nil
}

type observedHTTPFailure struct {
	Method string
	Path   string
	Status int
	Body   string
}

type observedRoundTripper struct {
	base http.RoundTripper
	mu   sync.Mutex
	last observedHTTPFailure
}

func (t *observedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if err != nil || response == nil || response.StatusCode < http.StatusBadRequest {
		return response, err
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if readErr != nil {
		return nil, errors.Wrap(readErr, "observe failed OIDC response")
	}
	_ = response.Body.Close()
	response.Body = io.NopCloser(strings.NewReader(string(body)))
	failure := observedHTTPFailure{Method: request.Method, Path: request.URL.EscapedPath(), Status: response.StatusCode, Body: string(body)}
	t.mu.Lock()
	t.last = failure
	t.mu.Unlock()
	return response, nil
}

func (t *observedRoundTripper) LastFailure() observedHTTPFailure {
	if t == nil {
		return observedHTTPFailure{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last
}
