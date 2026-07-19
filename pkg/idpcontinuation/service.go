package idpcontinuation

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

const (
	handleBytes      = 32
	handleHashDomain = "tiny-idp/workflow-continuation/v1\x00"
)

type FailureClass string

const (
	FailureMissing               FailureClass = "missing"
	FailureExpired               FailureClass = "expired"
	FailureReplayed              FailureClass = "replayed"
	FailureRevoked               FailureClass = "revoked"
	FailureBrowserMismatch       FailureClass = "browser_mismatch"
	FailureClientMismatch        FailureClass = "client_mismatch"
	FailureRequestMismatch       FailureClass = "request_mismatch"
	FailureGenerationUnavailable FailureClass = "generation_unavailable"
	FailureGenerationMismatch    FailureClass = "generation_mismatch"
	FailureInvalid               FailureClass = "invalid"
)

type Failure struct {
	Class FailureClass
	Err   error
}

func (e *Failure) Error() string {
	if e == nil || e.Err == nil {
		return "workflow continuation failed safely"
	}
	return "workflow continuation failed safely: " + e.Err.Error()
}

func (e *Failure) Unwrap() error { return e.Err }

// SafeTerminal describes the information split at the browser/audit boundary.
// PublicCode is intentionally uniform so a browser cannot distinguish missing,
// replayed, mismatched, or unavailable records. AuditClass is bounded and may
// be recorded internally without the handle or backend error text.
type SafeTerminal struct {
	PublicCode string
	AuditClass FailureClass
}

func ClassifyTerminal(err error) SafeTerminal {
	terminal := SafeTerminal{PublicCode: "interaction_unavailable", AuditClass: FailureInvalid}
	var failure *Failure
	if errors.As(err, &failure) {
		terminal.AuditClass = failure.Class
	}
	return terminal
}

type Config struct {
	HashKey  []byte
	Clock    func() time.Time
	Random   io.Reader
	Resolver GenerationResolver
	Cleaner  AttachmentCleaner
}

type Service struct {
	store    Store
	hashKey  []byte
	clock    func() time.Time
	random   io.Reader
	resolver GenerationResolver
	cleaner  AttachmentCleaner
	metrics  metrics
}

type metrics struct{ created, loaded, loadFailures, replayed, expired, cleaned atomic.Uint64 }
type Metrics struct{ Created, Loaded, LoadFailures, Replayed, Expired, Cleaned uint64 }

func (s *Service) Metrics() Metrics {
	if s == nil {
		return Metrics{}
	}
	return Metrics{Created: s.metrics.created.Load(), Loaded: s.metrics.loaded.Load(), LoadFailures: s.metrics.loadFailures.Load(), Replayed: s.metrics.replayed.Load(), Expired: s.metrics.expired.Load(), Cleaned: s.metrics.cleaned.Load()}
}

func NewService(store Store, config Config) (*Service, error) {
	if store == nil {
		return nil, errors.New("continuation store is required")
	}
	if len(config.HashKey) < 32 {
		return nil, errors.New("continuation handle hash key must be at least 32 bytes")
	}
	if config.Resolver == nil {
		return nil, errors.New("continuation generation resolver is required")
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	return &Service{
		store:    store,
		hashKey:  append([]byte(nil), config.HashKey...),
		clock:    config.Clock,
		random:   config.Random,
		resolver: config.Resolver,
		cleaner:  config.Cleaner,
	}, nil
}

func (s *Service) Create(ctx context.Context, continuation WorkflowContinuation) (string, WorkflowContinuation, error) {
	now := s.clock().UTC()
	handle, hash, err := s.newHandle()
	if err != nil {
		return "", WorkflowContinuation{}, err
	}
	continuation.Version = RecordVersionV1
	continuation.HandleHash = hash
	continuation.Revision = 1
	continuation.CreatedAt = now
	continuation.Status = StatusActive
	continuation.Terminal = nil
	if err := s.validate(ctx, continuation, now); err != nil {
		return "", WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: err}
	}
	if err := s.store.Create(ctx, continuation); err != nil {
		return "", WorkflowContinuation{}, errors.Wrap(err, "create workflow continuation")
	}
	s.metrics.created.Add(1)
	return handle, clone(continuation), nil
}

func (s *Service) Load(ctx context.Context, handle string, bindings Bindings) (WorkflowContinuation, error) {
	if err := validateExpectedBindings(bindings); err != nil {
		s.metrics.loadFailures.Add(1)
		return WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: err}
	}
	hash, err := s.hashHandle(handle)
	if err != nil {
		failure := &Failure{Class: FailureMissing, Err: ErrNotFound}
		s.recordLoadFailure(failure)
		return WorkflowContinuation{}, failure
	}
	now := s.clock().UTC()
	continuation, err := s.store.Load(ctx, hash, now)
	if err != nil {
		failure := classifyStoreFailure(err)
		s.recordLoadFailure(failure)
		return WorkflowContinuation{}, failure
	}
	if err := validateBindings(continuation, bindings); err != nil {
		s.recordLoadFailure(err)
		return WorkflowContinuation{}, err
	}
	if err := s.validate(ctx, continuation, now); err != nil {
		s.recordLoadFailure(err)
		if errors.Is(err, ErrGenerationUnavailable) {
			return WorkflowContinuation{}, &Failure{Class: FailureGenerationUnavailable, Err: err}
		}
		return WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: err}
	}
	s.metrics.loaded.Add(1)
	return clone(continuation), nil
}

func (s *Service) recordLoadFailure(err error) {
	s.metrics.loadFailures.Add(1)
	var failure *Failure
	if errors.As(err, &failure) {
		if failure.Class == FailureExpired {
			s.metrics.expired.Add(1)
		}
		if failure.Class == FailureReplayed {
			s.metrics.replayed.Add(1)
		}
	}
}

// ValidateResumeInput validates the provider-projected browser submission
// against the exact destination lambda schema pinned by the continuation.
// Sensitive values are allowed here because this input is ephemeral; they are
// forbidden only in durable public Carry.
func (s *Service) ValidateResumeInput(ctx context.Context, continuation WorkflowContinuation, input json.RawMessage) error {
	program, err := s.resolver.ResolveProgram(ctx, continuation.ProgramFingerprint)
	if err != nil {
		return &Failure{Class: FailureGenerationUnavailable, Err: errors.Wrap(ErrGenerationUnavailable, err.Error())}
	}
	workflow, ok := program.Workflows[continuation.WorkflowID]
	if !ok || workflow.Version != continuation.WorkflowVersion {
		return &Failure{Class: FailureGenerationMismatch, Err: errors.New("continuation workflow generation does not match")}
	}
	handler, ok := workflow.Handlers[continuation.ResumeHandlerID]
	if !ok {
		return &Failure{Class: FailureInvalid, Err: errors.New("continuation resume handler is unknown")}
	}
	lambda, ok := program.Lambdas[handler.LambdaID]
	if !ok || lambda.InputSchema != continuation.InputSchema {
		return &Failure{Class: FailureInvalid, Err: errors.New("continuation destination input schema is incompatible")}
	}
	if err := idpprogram.ValidateJSON(program.Schemas, continuation.InputSchema, input); err != nil {
		return &Failure{Class: FailureInvalid, Err: errors.Wrap(err, "validate resumed workflow input")}
	}
	return nil
}

func (s *Service) Advance(ctx context.Context, handle string, expectedRevision uint64, bindings Bindings, next WorkflowContinuation) (string, WorkflowContinuation, error) {
	current, err := s.Load(ctx, handle, bindings)
	if err != nil {
		return "", WorkflowContinuation{}, err
	}
	if current.Revision != expectedRevision {
		s.metrics.replayed.Add(1)
		return "", WorkflowContinuation{}, &Failure{Class: FailureReplayed, Err: ErrConflict}
	}
	nextHandle, nextHash, err := s.newHandle()
	if err != nil {
		return "", WorkflowContinuation{}, err
	}
	now := s.clock().UTC()
	next.Version = RecordVersionV1
	next.HandleHash = nextHash
	next.Revision = 1
	next.CreatedAt = now
	next.Status = StatusActive
	next.Terminal = nil
	if err := inheritAndValidateNext(current, &next); err != nil {
		return "", WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: err}
	}
	if err := s.validate(ctx, next, now); err != nil {
		return "", WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: err}
	}
	if err := s.store.Advance(ctx, current.HandleHash, expectedRevision, next, now); err != nil {
		return "", WorkflowContinuation{}, classifyStoreFailure(err)
	}
	return nextHandle, clone(next), nil
}

func (s *Service) Consume(ctx context.Context, handle string, expectedRevision uint64, bindings Bindings, outcome TerminalOutcome) (WorkflowContinuation, error) {
	current, err := s.Load(ctx, handle, bindings)
	if err != nil {
		return WorkflowContinuation{}, err
	}
	if current.Revision != expectedRevision {
		s.metrics.replayed.Add(1)
		return WorkflowContinuation{}, &Failure{Class: FailureReplayed, Err: ErrConflict}
	}
	if outcome.Kind != TerminalComplete && outcome.Kind != TerminalDeny && outcome.Kind != TerminalError {
		return WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: errors.Errorf("invalid terminal outcome %q", outcome.Kind)}
	}
	return s.consumeLoaded(ctx, current, bindings, outcome, s.store)
}

// ConsumeLoaded records a terminal transition for a continuation already
// loaded and binding-checked by the caller. The supplied store may be a
// transaction-scoped implementation of Store, which lets a native effect
// committer consume a continuation in the same transaction as its own state.
// It never accepts a raw browser handle, so it cannot widen the public API.
func (s *Service) ConsumeLoaded(ctx context.Context, current WorkflowContinuation, bindings Bindings, outcome TerminalOutcome, store Store) (WorkflowContinuation, error) {
	if store == nil {
		return WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: errors.New("continuation transaction store is required")}
	}
	if err := validateExpectedBindings(bindings); err != nil {
		return WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: err}
	}
	if err := validateBindings(current, bindings); err != nil {
		return WorkflowContinuation{}, err
	}
	if err := s.validate(ctx, current, s.clock().UTC()); err != nil {
		return WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: err}
	}
	if outcome.Kind != TerminalComplete && outcome.Kind != TerminalDeny && outcome.Kind != TerminalError {
		return WorkflowContinuation{}, &Failure{Class: FailureInvalid, Err: errors.Errorf("invalid terminal outcome %q", outcome.Kind)}
	}
	return s.consumeLoaded(ctx, current, bindings, outcome, store)
}

func (s *Service) consumeLoaded(ctx context.Context, current WorkflowContinuation, _ Bindings, outcome TerminalOutcome, store Store) (WorkflowContinuation, error) {
	now := s.clock().UTC()
	outcome.At = now
	consumed, err := store.Consume(ctx, current.HandleHash, current.Revision, outcome, now)
	if err != nil {
		return WorkflowContinuation{}, classifyStoreFailure(err)
	}
	return clone(consumed), nil
}

func (s *Service) Revoke(ctx context.Context, handle string, expectedRevision uint64) error {
	hash, err := s.hashHandle(handle)
	if err != nil {
		return &Failure{Class: FailureMissing, Err: ErrNotFound}
	}
	if err := s.store.Revoke(ctx, hash, expectedRevision, s.clock().UTC()); err != nil {
		return classifyStoreFailure(err)
	}
	return nil
}

func (s *Service) Cleanup(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		return 0, errors.New("cleanup limit must be greater than zero")
	}
	now := s.clock().UTC()
	expired, err := s.store.ListExpired(ctx, now, limit)
	if err != nil {
		return 0, errors.Wrap(err, "list expired workflow continuations")
	}
	removed := 0
	for _, continuation := range expired {
		if s.cleaner != nil {
			if err := s.cleaner.DeleteContinuationAttachments(ctx, continuation.SecretReferences, continuation.EvidenceReferences); err != nil {
				return removed, errors.Wrap(err, "cleanup continuation attachments")
			}
		}
		if err := s.store.DeleteExpired(ctx, continuation.HandleHash, now); err != nil {
			return removed, errors.Wrap(err, "delete expired workflow continuation")
		}
		removed++
	}
	s.metrics.cleaned.Add(uint64(removed))
	return removed, nil
}

var ErrGenerationUnavailable = errors.New("workflow program generation unavailable")

func (s *Service) validate(ctx context.Context, continuation WorkflowContinuation, now time.Time) error {
	if continuation.Version != RecordVersionV1 {
		return errors.Errorf("unsupported continuation version %d", continuation.Version)
	}
	if len(continuation.HandleHash) != sha256.Size {
		return errors.New("continuation handle hash must be 32 bytes")
	}
	if continuation.WorkflowID == "" || continuation.ResumeHandlerID == "" || continuation.ProgramFingerprint == "" || continuation.InputSchema == "" {
		return errors.New("continuation workflow, handler, generation, and input schema are required")
	}
	if continuation.ClientID == "" || continuation.RedirectURI == "" || continuation.ClientGeneration == "" {
		return errors.New("continuation client binding is incomplete")
	}
	if len(continuation.RequestDigest) == 0 || len(continuation.BrowserBindingHash) == 0 {
		return errors.New("continuation request and browser bindings are required")
	}
	if !continuation.ExpiresAt.After(now) || continuation.ExpiresAt.Before(continuation.CreatedAt) {
		return ErrExpired
	}
	if continuation.Status != StatusActive {
		return errors.Errorf("continuation is not active: %s", continuation.Status)
	}
	program, err := s.resolver.ResolveProgram(ctx, continuation.ProgramFingerprint)
	if err != nil {
		return errors.Wrap(ErrGenerationUnavailable, err.Error())
	}
	workflow, ok := program.Workflows[continuation.WorkflowID]
	if !ok || workflow.Version != continuation.WorkflowVersion {
		return errors.New("continuation workflow generation does not match")
	}
	handler, ok := workflow.Handlers[continuation.ResumeHandlerID]
	if !ok {
		return errors.New("continuation resume handler is unknown")
	}
	lambda, ok := program.Lambdas[handler.LambdaID]
	if !ok || lambda.InputSchema != continuation.InputSchema {
		return errors.New("continuation destination input schema is incompatible")
	}
	if err := idpprogram.ValidatePublicJSON(program.Schemas, continuation.InputSchema, continuation.Carry); err != nil {
		return errors.Wrap(err, "validate continuation carry")
	}
	return nil
}

func inheritAndValidateNext(current WorkflowContinuation, next *WorkflowContinuation) error {
	if next.WorkflowID == "" {
		next.WorkflowID = current.WorkflowID
	}
	if next.ProgramFingerprint == "" {
		next.ProgramFingerprint = current.ProgramFingerprint
	}
	if next.SchemaVersion == "" {
		next.SchemaVersion = current.SchemaVersion
	}
	if next.WorkflowVersion == 0 {
		next.WorkflowVersion = current.WorkflowVersion
	}
	if next.ClientID == "" {
		next.ClientID = current.ClientID
	}
	if next.RedirectURI == "" {
		next.RedirectURI = current.RedirectURI
	}
	if next.ClientGeneration == "" {
		next.ClientGeneration = current.ClientGeneration
	}
	if len(next.RequestDigest) == 0 {
		next.RequestDigest = append([]byte(nil), current.RequestDigest...)
	}
	if len(next.BrowserBindingHash) == 0 {
		next.BrowserBindingHash = append([]byte(nil), current.BrowserBindingHash...)
	}
	if len(next.SessionIDHash) == 0 {
		next.SessionIDHash = append([]byte(nil), current.SessionIDHash...)
	}
	if len(next.BrowserContextHash) == 0 {
		next.BrowserContextHash = append([]byte(nil), current.BrowserContextHash...)
	}
	if next.WorkflowID != current.WorkflowID || next.ProgramFingerprint != current.ProgramFingerprint ||
		next.WorkflowVersion != current.WorkflowVersion || next.ClientID != current.ClientID ||
		next.RedirectURI != current.RedirectURI || next.ClientGeneration != current.ClientGeneration ||
		!equal(next.RequestDigest, current.RequestDigest) || !equal(next.BrowserBindingHash, current.BrowserBindingHash) ||
		!equal(next.SessionIDHash, current.SessionIDHash) || !equal(next.BrowserContextHash, current.BrowserContextHash) {
		return errors.New("advanced continuation changed an immutable binding")
	}
	return nil
}

func validateBindings(record WorkflowContinuation, bindings Bindings) error {
	if bindings.WorkflowID != "" && bindings.WorkflowID != record.WorkflowID ||
		bindings.ClientID != "" && bindings.ClientID != record.ClientID ||
		bindings.RedirectURI != "" && bindings.RedirectURI != record.RedirectURI ||
		bindings.ClientGeneration != "" && bindings.ClientGeneration != record.ClientGeneration {
		return &Failure{Class: FailureClientMismatch, Err: errors.New("client binding mismatch")}
	}
	if bindings.ProgramFingerprint != "" && bindings.ProgramFingerprint != record.ProgramFingerprint {
		return &Failure{Class: FailureGenerationMismatch, Err: errors.New("program generation mismatch")}
	}
	if len(bindings.RequestDigest) != 0 && !equal(bindings.RequestDigest, record.RequestDigest) {
		return &Failure{Class: FailureRequestMismatch, Err: errors.New("authorization request mismatch")}
	}
	if len(bindings.BrowserBindingHash) != 0 && !equal(bindings.BrowserBindingHash, record.BrowserBindingHash) ||
		len(bindings.SessionIDHash) != 0 && !equal(bindings.SessionIDHash, record.SessionIDHash) ||
		len(bindings.BrowserContextHash) != 0 && !equal(bindings.BrowserContextHash, record.BrowserContextHash) {
		return &Failure{Class: FailureBrowserMismatch, Err: errors.New("browser binding mismatch")}
	}
	return nil
}

func validateExpectedBindings(bindings Bindings) error {
	if bindings.WorkflowID == "" || bindings.ClientID == "" || bindings.RedirectURI == "" ||
		bindings.ClientGeneration == "" ||
		len(bindings.RequestDigest) == 0 || len(bindings.BrowserBindingHash) == 0 {
		return errors.New("complete workflow, client, request, and browser bindings are required")
	}
	return nil
}

func classifyStoreFailure(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return &Failure{Class: FailureMissing, Err: ErrNotFound}
	case errors.Is(err, ErrExpired):
		return &Failure{Class: FailureExpired, Err: ErrExpired}
	case errors.Is(err, ErrRevoked):
		return &Failure{Class: FailureRevoked, Err: ErrRevoked}
	case errors.Is(err, ErrAlreadyTerminal), errors.Is(err, ErrConflict):
		return &Failure{Class: FailureReplayed, Err: err}
	default:
		return err
	}
}

func (s *Service) newHandle() (string, []byte, error) {
	raw := make([]byte, handleBytes)
	if _, err := io.ReadFull(s.random, raw); err != nil {
		return "", nil, errors.Wrap(err, "generate continuation handle")
	}
	handle := base64.RawURLEncoding.EncodeToString(raw)
	hash, err := s.hashHandle(handle)
	return handle, hash, err
}

func (s *Service) hashHandle(handle string) ([]byte, error) {
	if strings.ContainsAny(handle, "+/=") {
		return nil, ErrNotFound
	}
	raw, err := base64.RawURLEncoding.DecodeString(handle)
	if err != nil || len(raw) != handleBytes {
		return nil, ErrNotFound
	}
	mac := hmac.New(sha256.New, s.hashKey)
	_, _ = mac.Write([]byte(handleHashDomain))
	_, _ = mac.Write(raw)
	return mac.Sum(nil), nil
}

func equal(left, right []byte) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare(left, right) == 1
}

func clone(record WorkflowContinuation) WorkflowContinuation {
	encoded, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	var copied WorkflowContinuation
	if err := json.Unmarshal(encoded, &copied); err != nil {
		panic(err)
	}
	return copied
}
