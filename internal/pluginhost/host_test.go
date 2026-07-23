package pluginhost

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

type testPrepared struct {
	descriptor pluginapi.Descriptor
	enabled    bool
	require    pluginapi.Requirements
	build      func() (pluginapi.Runtime, error)
}

func (p testPrepared) Descriptor() pluginapi.Descriptor     { return p.descriptor }
func (p testPrepared) Enabled() bool                        { return p.enabled }
func (p testPrepared) Requirements() pluginapi.Requirements { return p.require }
func (p testPrepared) Build(context.Context, pluginapi.RuntimeServices) (pluginapi.Runtime, error) {
	return p.build()
}

type testRuntime struct {
	descriptor pluginapi.Descriptor
	handler    http.Handler
	ready      bool
	close      func() error
}

func (r *testRuntime) Descriptor() pluginapi.Descriptor { return r.descriptor }
func (r *testRuntime) Handler() http.Handler            { return r.handler }
func (r *testRuntime) Readiness(context.Context) idp.ReadinessCheck {
	return idp.ReadinessCheck{Ready: r.ready, CheckedAt: time.Unix(1, 0)}
}
func (r *testRuntime) Close(context.Context) error { return r.close() }

var _ pluginapi.Prepared = testPrepared{}
var _ pluginapi.Runtime = (*testRuntime)(nil)

func TestBuildFailureClosesConstructedRuntimesInReverse(t *testing.T) {
	var closed []string
	descriptor := func(id string) pluginapi.Descriptor {
		return pluginapi.Descriptor{ID: id, APIVersion: pluginapi.APIVersion, Summary: id}
	}
	prepared := []pluginapi.Prepared{
		testPrepared{descriptor: descriptor("alpha"), enabled: true, build: func() (pluginapi.Runtime, error) {
			return &testRuntime{descriptor: descriptor("alpha"), handler: http.NotFoundHandler(), ready: true, close: func() error { closed = append(closed, "alpha"); return nil }}, nil
		}},
		testPrepared{descriptor: descriptor("beta"), enabled: true, build: func() (pluginapi.Runtime, error) {
			return &testRuntime{descriptor: descriptor("beta"), handler: http.NotFoundHandler(), ready: true, close: func() error { closed = append(closed, "beta"); return nil }}, nil
		}},
		testPrepared{descriptor: descriptor("failure"), enabled: true, build: func() (pluginapi.Runtime, error) {
			return nil, errors.New("deliberate")
		}},
	}
	if _, err := Build(context.Background(), prepared, pluginapi.RuntimeServices{}); err == nil {
		t.Fatal("build failure was ignored")
	}
	if !reflect.DeepEqual(closed, []string{"beta", "alpha"}) {
		t.Fatalf("close order = %#v", closed)
	}
}

func TestMountScopesPathsAndAddsSecurityHeaders(t *testing.T) {
	descriptor := pluginapi.Descriptor{ID: "jitsi", APIVersion: pluginapi.APIVersion, Summary: "jitsi"}
	runtime := &testRuntime{
		descriptor: descriptor,
		handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte(request.URL.Path))
		}),
		ready: true,
		close: func() error { return nil },
	}
	mux := http.NewServeMux()
	if err := Mount(mux, []pluginapi.Runtime{runtime}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/integrations/jitsi/start", nil))
	if response.Code != http.StatusOK || response.Body.String() != "/start" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	for _, header := range []string{"Content-Security-Policy", "Referrer-Policy", "X-Content-Type-Options", "X-Frame-Options"} {
		if response.Header().Get(header) == "" {
			t.Fatalf("missing %s", header)
		}
	}
}

func TestReadinessFailsClosed(t *testing.T) {
	descriptor := pluginapi.Descriptor{ID: "jitsi", APIVersion: pluginapi.APIVersion, Summary: "jitsi"}
	report := Readiness(context.Background(), []pluginapi.Runtime{
		&testRuntime{descriptor: descriptor, handler: http.NotFoundHandler(), ready: false, close: func() error { return nil }},
	})
	if report.Ready || len(report.Checks) != 1 || report.Checks[0].Name != "plugin.jitsi" {
		t.Fatalf("report = %#v", report)
	}
}

func TestValidateClientRequirementsRequiresReviewedExactProfile(t *testing.T) {
	descriptor := pluginapi.Descriptor{ID: "jitsi", APIVersion: pluginapi.APIVersion, Summary: "jitsi"}
	requirement := pluginapi.OIDCClientRequirement{
		ID: "tinyidp-plugin-jitsi", RedirectURI: "https://idp.example/integrations/jitsi/callback",
		Scopes: []string{"openid", "profile"}, Public: true, RequirePKCE: true,
	}
	prepared := []pluginapi.Prepared{testPrepared{descriptor: descriptor, enabled: true, require: pluginapi.Requirements{OIDCClients: []pluginapi.OIDCClientRequirement{requirement}}}}
	valid := idpstore.Client{
		ID: requirement.ID, Public: true, RequirePKCE: true,
		RedirectURIs: []string{requirement.RedirectURI}, AllowedScopes: requirement.Scopes,
		AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode},
	}
	if err := ValidateClientRequirements(prepared, []idpstore.Client{valid}); err != nil {
		t.Fatalf("valid client rejected: %v", err)
	}
	invalid := valid
	invalid.RequirePKCE = false
	if err := ValidateClientRequirements(prepared, []idpstore.Client{invalid}); err == nil {
		t.Fatal("client without PKCE accepted")
	}
	if err := ValidateClientRequirements(prepared, nil); err == nil {
		t.Fatal("missing client accepted")
	}
}
