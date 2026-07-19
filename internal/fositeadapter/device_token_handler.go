package fositeadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/storage"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

// deviceTokenHandler is the Fosite extension for RFC 8628 device-code
// redemption. Fosite owns client authentication and token construction; the
// handler owns the durable device-grant state machine and maps it to RFC 8628
// token errors.
type deviceTokenHandler struct {
	provider *Provider
	core     oauth2.CoreStrategy
	oidc     openid.OpenIDConnectTokenStrategy
	storage  oauth2.CoreStorage
}

var _ fosite.TokenEndpointHandler = (*deviceTokenHandler)(nil)

func newDeviceTokenHandler(provider *Provider, core oauth2.CoreStrategy, oidc openid.OpenIDConnectTokenStrategy, store interface{}) (*deviceTokenHandler, error) {
	if provider == nil || core == nil || oidc == nil {
		return nil, fmt.Errorf("device token handler requires provider and token strategies")
	}
	storage, ok := store.(oauth2.CoreStorage)
	if !ok {
		return nil, fmt.Errorf("fosite store does not provide core token storage")
	}
	return &deviceTokenHandler{provider: provider, core: core, oidc: oidc, storage: storage}, nil
}

func (h *deviceTokenHandler) CanHandleTokenEndpointRequest(_ context.Context, request fosite.AccessRequester) bool {
	return request.GetGrantTypes().ExactOne(idpstore.GrantDeviceCode)
}

func (h *deviceTokenHandler) CanSkipClientAuth(context.Context, fosite.AccessRequester) bool {
	// RFC 8628 does not exempt public clients from identifying themselves at the
	// token endpoint. Fosite accepts their registered client_id without a secret.
	return false
}

func (h *deviceTokenHandler) HandleTokenEndpointRequest(ctx context.Context, request fosite.AccessRequester) error {
	if !h.CanHandleTokenEndpointRequest(ctx, request) {
		return fosite.ErrUnknownRequest
	}
	form := request.GetRequestForm()
	if len(form["device_code"]) != 1 || strings.TrimSpace(form.Get("device_code")) == "" || len(form["scope"]) > 0 || len(form["client_id"]) > 1 {
		return fosite.ErrInvalidRequest
	}
	client := request.GetClient()
	if client == nil || !client.GetGrantTypes().Has(idpstore.GrantDeviceCode) {
		return fosite.ErrUnauthorizedClient
	}
	result, err := h.provider.store.PollDeviceGrant(ctx, idpstore.DevicePollRequest{DeviceCodeHash: deviceCodeHash(h.provider.csrfKey, form.Get("device_code")), ClientID: client.GetID(), Now: h.provider.now()})
	if err != nil {
		if errors.Is(err, idpstore.ErrNotFound) {
			return fosite.ErrInvalidGrant
		}
		return fosite.ErrServerError
	}
	switch result.Outcome {
	case idpstore.DevicePollPending:
		return deviceTokenProtocolError("authorization_pending")
	case idpstore.DevicePollSlowDown:
		return deviceTokenProtocolError("slow_down")
	case idpstore.DevicePollDenied:
		return deviceTokenProtocolError("access_denied")
	case idpstore.DevicePollExpired:
		return deviceTokenProtocolError("expired_token")
	case idpstore.DevicePollConsumed:
		return fosite.ErrInvalidGrant
	case idpstore.DevicePollApproved:
	default:
		return fosite.ErrServerError
	}
	grant := result.Grant
	user, err := h.provider.store.GetUser(ctx, grant.UserID)
	if err != nil || user.Disabled || grant.Subject == "" || user.Sub != grant.Subject || grant.AuthTime.IsZero() {
		return fosite.ErrInvalidGrant
	}
	if len(grant.ApprovedScopes) == 0 || !containsScope(grant.ApprovedScopes, "openid") {
		return fosite.ErrInvalidGrant
	}
	request.SetRequestedScopes(fosite.Arguments(append([]string(nil), grant.ApprovedScopes...)))
	for _, scope := range grant.ApprovedScopes {
		request.GrantScope(scope)
	}
	request.SetRequestedAudience(fosite.Arguments(append([]string(nil), grant.ApprovedAudiences...)))
	for _, audience := range grant.ApprovedAudiences {
		request.GrantAudience(audience)
	}
	request.SetID("device:" + grant.ID)
	session, err := h.provider.newOIDCSession(ctx, user, request, grant.AuthTime)
	if err != nil {
		return fosite.ErrServerError
	}
	request.SetSession(session)
	now := h.provider.now()
	accessTTL := fosite.GetEffectiveLifespan(client, idpstore.GrantDeviceCode, fosite.AccessToken, h.provider.config.GetAccessTokenLifespan(ctx))
	request.GetSession().SetExpiresAt(fosite.AccessToken, now.Add(accessTTL).Round(time.Second))
	if client.GetGrantTypes().Has(idpstore.GrantRefreshToken) && request.GetGrantedScopes().Has("offline_access") {
		refreshTTL := fosite.GetEffectiveLifespan(client, idpstore.GrantDeviceCode, fosite.RefreshToken, h.provider.config.GetRefreshTokenLifespan(ctx))
		if refreshTTL > -1 {
			request.GetSession().SetExpiresAt(fosite.RefreshToken, now.Add(refreshTTL).Round(time.Second))
		}
	}
	return nil
}

func (h *deviceTokenHandler) PopulateTokenEndpointResponse(ctx context.Context, requester fosite.AccessRequester, responder fosite.AccessResponder) error {
	if !h.CanHandleTokenEndpointRequest(ctx, requester) {
		return fosite.ErrUnknownRequest
	}
	deviceCode := requester.GetRequestForm().Get("device_code")
	if deviceCode == "" || requester.GetClient() == nil {
		return fosite.ErrInvalidRequest
	}
	access, accessSignature, err := h.core.GenerateAccessToken(ctx, requester)
	if err != nil {
		return fosite.ErrServerError
	}
	var refresh, refreshSignature string
	if requester.GetClient().GetGrantTypes().Has(idpstore.GrantRefreshToken) && requester.GetGrantedScopes().Has("offline_access") && !requester.GetSession().GetExpiresAt(fosite.RefreshToken).IsZero() {
		refresh, refreshSignature, err = h.core.GenerateRefreshToken(ctx, requester)
		if err != nil {
			return fosite.ErrServerError
		}
	}
	// The SQLite production store deliberately permits one open connection. ID
	// token signing loads the active signing key through the project store, so it
	// must happen before the Fosite token transaction reserves that connection.
	// No token response is emitted when a later persistence step fails.
	responder.SetAccessToken(access)
	responder.SetTokenType("bearer")
	responder.SetExpiresIn(requester.GetSession().GetExpiresAt(fosite.AccessToken).Sub(h.provider.now()))
	responder.SetScopes(requester.GetGrantedScopes())
	if refresh != "" {
		responder.SetExtra("refresh_token", refresh)
	}
	session, ok := requester.GetSession().(openid.Session)
	if !ok {
		return fosite.ErrServerError
	}
	session.IDTokenClaims().AccessTokenHash = (&openid.IDTokenHandleHelper{IDTokenStrategy: h.oidc}).GetAccessTokenHash(ctx, requester, responder)
	if err = (&openid.IDTokenHandleHelper{IDTokenStrategy: h.oidc}).IssueExplicitIDToken(ctx, fosite.GetEffectiveLifespan(requester.GetClient(), idpstore.GrantDeviceCode, fosite.IDToken, h.provider.config.GetIDTokenLifespan(ctx)), requester, responder); err != nil {
		return fosite.ErrServerError
	}
	ctx, err = storage.MaybeBeginTx(ctx, h.storage)
	if err != nil {
		return fosite.ErrServerError
	}
	rollback := func() { _ = storage.MaybeRollbackTx(ctx, h.storage) }
	consume := idpstore.DeviceConsumeRequest{DeviceCodeHash: deviceCodeHash(h.provider.csrfKey, deviceCode), ClientID: requester.GetClient().GetID(), Now: h.provider.now()}
	if h.provider.sqlStore != nil {
		_, err = h.provider.sqlStore.consumeDeviceGrantInTokenTransaction(ctx, consume)
	} else {
		// The in-memory adapter is development-only. SQLite production flows use
		// the branch above, where consumption and token persistence share one SQL
		// transaction.
		_, err = h.provider.store.ConsumeDeviceGrant(ctx, consume)
	}
	if err != nil {
		rollback()
		return deviceConsumeError(err)
	}
	// Device code material is accepted only at this boundary. Never let it enter
	// Fosite's persisted sanitized requester representation.
	requester.GetRequestForm().Del("device_code")
	if err = h.storage.CreateAccessTokenSession(ctx, accessSignature, requester.Sanitize([]string{})); err != nil {
		rollback()
		return fosite.ErrServerError
	}
	if refreshSignature != "" {
		if err = h.storage.CreateRefreshTokenSession(ctx, refreshSignature, accessSignature, requester.Sanitize([]string{})); err != nil {
			rollback()
			return fosite.ErrServerError
		}
	}
	if err = storage.MaybeCommitTx(ctx, h.storage); err != nil {
		rollback()
		return fosite.ErrServerError
	}
	return nil
}

func deviceTokenProtocolError(code string) *fosite.RFC6749Error {
	return &fosite.RFC6749Error{ErrorField: code, DescriptionField: "device authorization is not ready", CodeField: 400}
}

func deviceConsumeError(err error) error {
	switch {
	case errors.Is(err, idpstore.ErrExpired):
		return deviceTokenProtocolError("expired_token")
	case errors.Is(err, idpstore.ErrNotFound), errors.Is(err, idpstore.ErrAlreadyConsumed), errors.Is(err, idpstore.ErrDeviceGrantNotApproved):
		return fosite.ErrInvalidGrant
	default:
		return fosite.ErrServerError
	}
}
