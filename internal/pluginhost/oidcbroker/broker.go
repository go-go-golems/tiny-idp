package oidcbroker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
)

const maxProtocolResponseBytes int64 = 1 << 20

type ErrorCode string

const (
	CodeInvalidRequest  ErrorCode = "invalid_request"
	CodeTransaction     ErrorCode = "transaction_rejected"
	CodeTokenExchange   ErrorCode = "token_exchange_failed"
	CodeTokenValidation ErrorCode = "id_token_rejected"
	CodeUserInfo        ErrorCode = "userinfo_failed"
)

type BrokerError struct {
	Code ErrorCode
	Err  error
}

func (e *BrokerError) Error() string { return string(e.Code) }
func (e *BrokerError) Unwrap() error { return e.Err }

type Broker struct {
	issuer       string
	endpoint     oauth2.Endpoint
	verifier     func(string) *oidc.IDTokenVerifier
	client       *http.Client
	transactions *TransactionManager
}

var _ pluginapi.RelyingPartyBroker = (*Broker)(nil)

func New(ctx context.Context, issuer string, providerHandler http.Handler, transactions *TransactionManager) (*Broker, error) {
	if ctx == nil || transactions == nil {
		return nil, errors.New("broker context and transaction manager are required")
	}
	transport, err := embeddedidp.NewInProcessIssuerTransport(issuer, providerHandler, embeddedidp.InProcessTransportOptions{})
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("OIDC broker redirects are forbidden")
		},
		Timeout: 10 * time.Second,
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, client), issuer)
	if err != nil {
		return nil, fmt.Errorf("discover in-process OIDC provider: %w", err)
	}
	return &Broker{
		issuer: strings.TrimSuffix(issuer, "/"), endpoint: provider.Endpoint(),
		verifier: func(clientID string) *oidc.IDTokenVerifier {
			return provider.Verifier(&oidc.Config{ClientID: clientID, Now: transactions.now})
		},
		client: client, transactions: transactions,
	}, nil
}

func (b *Broker) Start(ctx context.Context, request pluginapi.StartRequest) (pluginapi.StartResult, error) {
	if b == nil || request.PluginID == "" || request.ClientID == "" || request.CallbackPath == "" ||
		!strings.HasPrefix(request.CallbackPath, "/integrations/"+request.PluginID+"/") {
		return pluginapi.StartResult{}, &BrokerError{Code: CodeInvalidRequest, Err: errors.New("invalid broker start request")}
	}
	created, err := b.transactions.Create(ctx, NewTransaction{
		PluginID: request.PluginID, ClientID: request.ClientID, CallbackPath: request.CallbackPath,
		PluginState: request.PluginState, BrowserBinding: request.BrowserBinding, TTL: request.TTL,
	})
	if err != nil {
		return pluginapi.StartResult{}, &BrokerError{Code: CodeTransaction, Err: err}
	}
	callbackURI := b.issuer + request.CallbackPath
	parameters := url.Values{
		"client_id":             {request.ClientID},
		"redirect_uri":          {callbackURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(request.Scopes, " ")},
		"state":                 {created.State},
		"nonce":                 {created.Nonce},
		"code_challenge":        {created.PKCEChallenge},
		"code_challenge_method": {"S256"},
	}
	authorizationURL := b.endpoint.AuthURL
	separator := "?"
	if strings.Contains(authorizationURL, "?") {
		separator = "&"
	}
	return pluginapi.StartResult{AuthorizationURL: authorizationURL + separator + parameters.Encode(), State: created.State}, nil
}

func (b *Broker) Complete(ctx context.Context, request pluginapi.CompleteRequest) (pluginapi.Completion, error) {
	if b == nil || request.Code == "" {
		return pluginapi.Completion{}, &BrokerError{Code: CodeInvalidRequest, Err: errors.New("authorization code is required")}
	}
	transaction, err := b.transactions.Consume(ctx, request.PluginID, request.BrowserBinding, request.State)
	if err != nil {
		return pluginapi.Completion{}, &BrokerError{Code: CodeTransaction, Err: err}
	}
	callbackURI := b.issuer + transaction.CallbackPath
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {transaction.ClientID},
		"redirect_uri":  {callbackURI},
		"code":          {request.Code},
		"code_verifier": {transaction.PKCEVerifier},
	}
	tokenRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return pluginapi.Completion{}, &BrokerError{Code: CodeTokenExchange, Err: err}
	}
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResponse, err := b.client.Do(tokenRequest)
	if err != nil {
		return pluginapi.Completion{}, &BrokerError{Code: CodeTokenExchange, Err: err}
	}
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode != http.StatusOK {
		return pluginapi.Completion{}, &BrokerError{Code: CodeTokenExchange, Err: fmt.Errorf("token endpoint status %d", tokenResponse.StatusCode)}
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := decodeBoundedJSON(tokenResponse.Body, &tokens); err != nil || tokens.AccessToken == "" || tokens.IDToken == "" {
		return pluginapi.Completion{}, &BrokerError{Code: CodeTokenExchange, Err: errors.New("token response was not accepted")}
	}
	idToken, err := b.verifier(transaction.ClientID).Verify(ctx, tokens.IDToken)
	if err != nil {
		return pluginapi.Completion{}, &BrokerError{Code: CodeTokenValidation, Err: err}
	}
	var tokenClaims struct {
		Nonce    string `json:"nonce"`
		AuthTime int64  `json:"auth_time"`
	}
	if err := idToken.Claims(&tokenClaims); err != nil || !b.transactions.nonceMatches(transaction.NonceHash, tokenClaims.Nonce) {
		return pluginapi.Completion{}, &BrokerError{Code: CodeTokenValidation, Err: errors.New("ID token nonce was not accepted")}
	}
	userRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, b.issuer+"/userinfo", nil)
	if err != nil {
		return pluginapi.Completion{}, &BrokerError{Code: CodeUserInfo, Err: err}
	}
	userRequest.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	userResponse, err := b.client.Do(userRequest)
	if err != nil {
		return pluginapi.Completion{}, &BrokerError{Code: CodeUserInfo, Err: err}
	}
	defer userResponse.Body.Close()
	if userResponse.StatusCode != http.StatusOK {
		return pluginapi.Completion{}, &BrokerError{Code: CodeUserInfo, Err: fmt.Errorf("userinfo endpoint status %d", userResponse.StatusCode)}
	}
	var identity pluginapi.Identity
	var userInfo struct {
		Subject           string   `json:"sub"`
		Email             string   `json:"email"`
		EmailVerified     bool     `json:"email_verified"`
		Name              string   `json:"name"`
		PreferredUsername string   `json:"preferred_username"`
		Groups            []string `json:"groups"`
		Roles             []string `json:"roles"`
	}
	if err := decodeBoundedJSON(userResponse.Body, &userInfo); err != nil || userInfo.Subject == "" {
		return pluginapi.Completion{}, &BrokerError{Code: CodeUserInfo, Err: errors.New("userinfo response was not accepted")}
	}
	identity = pluginapi.Identity{
		Subject: userInfo.Subject, Email: userInfo.Email, EmailVerified: userInfo.EmailVerified,
		Name: userInfo.Name, PreferredUsername: userInfo.PreferredUsername,
		Groups: append([]string(nil), userInfo.Groups...), Roles: append([]string(nil), userInfo.Roles...),
	}
	if tokenClaims.AuthTime > 0 {
		identity.AuthTime = time.Unix(tokenClaims.AuthTime, 0).UTC()
	}
	return pluginapi.Completion{Identity: identity, PluginState: transaction.PluginState}, nil
}

func decodeBoundedJSON(reader io.Reader, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(reader, maxProtocolResponseBytes+1))
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}
