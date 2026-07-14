package fositeadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ory/fosite"

	"github.com/manuel/tinyidp/internal/securitytrace"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

const interactionFieldName = "interaction"

var transientAuthorizeFields = map[string]struct{}{
	interactionFieldName: {},
	"action":             {},
	"consent_approved":   {},
	"csrf_token":         {},
	"login":              {},
	"password":           {},
	"account":            {},
}

func canonicalAuthorizeForm(ar fosite.AuthorizeRequester) url.Values {
	canonical := make(url.Values, len(ar.GetRequestForm()))
	for key, values := range ar.GetRequestForm() {
		if _, transient := transientAuthorizeFields[key]; transient {
			continue
		}
		canonical[key] = append([]string(nil), values...)
	}
	return canonical
}

func authorizeRequestDigest(values url.Values) []byte {
	sum := sha256.Sum256([]byte(values.Encode()))
	return sum[:]
}

func clientGenerationHash(client idpstore.Client) []byte {
	policy := struct {
		ID            string
		RedirectURIs  []string
		AllowedScopes []string
		RequirePKCE   bool
		Disabled      bool
		UpdatedAt     string
	}{
		ID:            client.ID,
		RedirectURIs:  append([]string(nil), client.RedirectURIs...),
		AllowedScopes: append([]string(nil), client.AllowedScopes...),
		RequirePKCE:   client.RequirePKCE,
		Disabled:      client.Disabled,
		UpdatedAt:     client.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	b, _ := json.Marshal(policy)
	sum := sha256.Sum256(b)
	return sum[:]
}

func (p *Provider) createInteraction(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, actions idpstore.InteractionRequiredAction) (string, string, error) {
	handle, err := randomB64(32)
	if err != nil {
		return "", "", err
	}
	csrfToken, bindingHash, err := p.issueCSRF(w, r, handle)
	if err != nil {
		return "", "", err
	}
	client, err := p.store.GetClient(r.Context(), ar.GetClient().GetID())
	if err != nil {
		return "", "", fmt.Errorf("load current client: %w", err)
	}
	if client.Disabled {
		return "", "", fmt.Errorf("current client is disabled")
	}
	canonical := canonicalAuthorizeForm(ar)
	now := p.now()
	var sessionIDHash []byte
	var browserContextHash []byte
	if !actions.Has(idpstore.InteractionRequireLogin) {
		sessionIDHash = p.browserSessionHash(r)
	}
	if actions.Has(idpstore.InteractionRequireAccountSelection) {
		browserContextHash = p.browserContextHash(r)
	}
	record := idpstore.InteractionRecord{
		IDHash:             idpstore.HashSecret(p.csrfKey, handle),
		CanonicalRequest:   canonical,
		RequestDigest:      authorizeRequestDigest(canonical),
		ClientID:           client.ID,
		RedirectURI:        ar.GetRedirectURI().String(),
		RequiredActions:    actions,
		BrowserBindingHash: bindingHash,
		SessionIDHash:      sessionIDHash,
		BrowserContextHash: browserContextHash,
		GenerationHash:     clientGenerationHash(client),
		CreatedAt:          now,
		ExpiresAt:          now.Add(p.interactionTTL),
	}
	if err := p.store.CreateInteraction(r.Context(), record); err != nil {
		return "", "", fmt.Errorf("create interaction: %w", err)
	}
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionCreated, InteractionID: interactionTraceID(record), ClientID: record.ClientID, RequiredActions: uint32(record.RequiredActions)})
	return handle, csrfToken, nil
}

func interactionTraceID(record idpstore.InteractionRecord) string {
	return hex.EncodeToString(record.IDHash)
}

func (p *Provider) browserSessionHash(r *http.Request) []byte {
	cookie, err := r.Cookie(p.sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}
	return idpstore.HashSecret(p.csrfKey, cookie.Value)
}

func (p *Provider) browserContextHash(r *http.Request) []byte {
	if !p.chooser.Enabled {
		return nil
	}
	cookie, err := r.Cookie(p.chooser.ContextCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}
	return idpstore.HashSecret(p.csrfKey, cookie.Value)
}

func (p *Provider) browserBindingHash(r *http.Request) []byte {
	cookie, err := r.Cookie(p.csrfCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}
	return idpstore.HashSecret(p.csrfKey, cookie.Value)
}

func (p *Provider) reconstructAuthorizeRequest(r *http.Request, record idpstore.InteractionRecord) (fosite.AuthorizeRequester, error) {
	values := make(url.Values, len(record.CanonicalRequest))
	for key, entries := range record.CanonicalRequest {
		values[key] = append([]string(nil), entries...)
	}
	reconstructed, err := http.NewRequestWithContext(r.Context(), http.MethodPost, p.issuer.Endpoint("/authorize"), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("reconstruct authorize request: %w", err)
	}
	reconstructed.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reconstructed.Header.Set("User-Agent", r.UserAgent())
	reconstructed.RemoteAddr = r.RemoteAddr
	ar, err := p.oauth2.NewAuthorizeRequest(fosite.NewContext(), reconstructed)
	if err != nil {
		return ar, err
	}
	if !equalBytes(authorizeRequestDigest(canonicalAuthorizeForm(ar)), record.RequestDigest) {
		return ar, fmt.Errorf("authorization request digest mismatch")
	}
	return ar, nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var different byte
	for i := range a {
		different |= a[i] ^ b[i]
	}
	return different == 0
}
