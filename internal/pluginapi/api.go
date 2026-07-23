// Package pluginapi defines the narrow, first-party extension contract used by
// the TinyIDP production host.
package pluginapi

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

const APIVersion = 1

var descriptorIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Descriptor struct {
	ID         string
	APIVersion uint32
	Summary    string
}

func (d Descriptor) Validate() error {
	if !descriptorIDPattern.MatchString(d.ID) {
		return &ValidationError{Field: "id", Reason: "must contain canonical lowercase ASCII letters, digits, and hyphens"}
	}
	if d.APIVersion != APIVersion {
		return &ValidationError{Field: "api_version", Reason: "unsupported"}
	}
	if d.Summary == "" {
		return &ValidationError{Field: "summary", Reason: "is required"}
	}
	return nil
}

func (d Descriptor) RoutePrefix() string {
	return "/integrations/" + d.ID + "/"
}

type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return "plugin " + e.Field + " " + e.Reason
}

type OIDCClientRequirement struct {
	ID          string
	RedirectURI string
	Scopes      []string
	Public      bool
	RequirePKCE bool
}

type Requirements struct {
	OIDCClients []OIDCClientRequirement
}

type Definition interface {
	Descriptor() Descriptor
	Section() (schema.Section, error)
	Prepare(context.Context, *values.Values) (Prepared, error)
}

type Prepared interface {
	Descriptor() Descriptor
	Enabled() bool
	Requirements() Requirements
	Build(context.Context, RuntimeServices) (Runtime, error)
}

type Runtime interface {
	Descriptor() Descriptor
	Handler() http.Handler
	Readiness(context.Context) idp.ReadinessCheck
	Close(context.Context) error
}

type SecretResolver interface {
	Read(context.Context, string, int) ([]byte, error)
}

type RelyingPartyBroker interface {
	Start(context.Context, StartRequest) (StartResult, error)
	Complete(context.Context, CompleteRequest) (Completion, error)
}

type StartRequest struct {
	ClientID    string
	RedirectURI string
	Scopes      []string
	ReturnTo    string
}

type StartResult struct {
	AuthorizationURL string
}

type CompleteRequest struct {
	State string
	Code  string
}

type Identity struct {
	Subject           string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	Groups            []string
	Roles             []string
}

type Completion struct {
	Identity Identity
	ReturnTo string
}

type Clock interface {
	Now() time.Time
}

type RuntimeServices struct {
	OIDC    RelyingPartyBroker
	Secrets SecretResolver
	Audit   idp.Sink
	Logger  zerolog.Logger
	Meter   metric.Meter
	Tracer  trace.Tracer
	Clock   Clock
	Random  io.Reader
}
