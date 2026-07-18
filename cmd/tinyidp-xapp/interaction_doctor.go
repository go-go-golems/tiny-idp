package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const interactionCSPPrefix = "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self' "
const interactionCSPSuffix = "; base-uri 'none'"

type InteractionHealth struct {
	HTMLBytes int
	CSSBytes  int
}

// CheckInteractionUI exercises the browser-facing authorization document and
// its declared stylesheet entirely in process. It never returns, logs, or
// persists cookies, interaction handles, CSRF values, or document contents.
func (a *DevelopmentApplication) CheckInteractionUI(ctx context.Context) (InteractionHealth, error) {
	if a == nil || a.handler == nil || a.publicBaseURL == "" {
		return InteractionHealth{}, fmt.Errorf("application interaction UI is unavailable")
	}
	login := httptest.NewRequest(http.MethodGet, a.publicBaseURL+"/auth/login", nil).WithContext(ctx)
	loginResponse := httptest.NewRecorder()
	a.Handler().ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusFound && loginResponse.Code != http.StatusSeeOther {
		return InteractionHealth{}, fmt.Errorf("login initiation status %d", loginResponse.Code)
	}
	location := loginResponse.Header().Get("Location")
	target, err := url.Parse(location)
	if err != nil || target.Path == "" {
		return InteractionHealth{}, fmt.Errorf("login initiation returned an invalid redirect")
	}
	base, _ := url.Parse(a.publicBaseURL)
	if target.IsAbs() && (target.Scheme != base.Scheme || target.Host != base.Host) {
		return InteractionHealth{}, fmt.Errorf("login initiation left the configured origin")
	}
	interaction := httptest.NewRequest(http.MethodGet, target.String(), nil).WithContext(ctx)
	for _, cookie := range loginResponse.Result().Cookies() {
		interaction.AddCookie(cookie)
	}
	interactionResponse := httptest.NewRecorder()
	a.Handler().ServeHTTP(interactionResponse, interaction)
	if interactionResponse.Code != http.StatusOK {
		return InteractionHealth{}, fmt.Errorf("interaction document status %d", interactionResponse.Code)
	}
	wantCSP, err := interactionCSPForRedirect(target.Query().Get("redirect_uri"))
	if err != nil {
		return InteractionHealth{}, fmt.Errorf("interaction document has an invalid redirect URI: %w", err)
	}
	if interactionResponse.Header().Get("Content-Security-Policy") != wantCSP || interactionResponse.Header().Get("Cache-Control") != "no-store" {
		return InteractionHealth{}, fmt.Errorf("interaction document security headers are invalid")
	}
	document := interactionResponse.Body.Bytes()
	if len(document) == 0 || len(document) > 256<<10 {
		return InteractionHealth{}, fmt.Errorf("interaction document size is invalid")
	}
	stylesheetPath, err := declaredStylesheet(document)
	if err != nil {
		return InteractionHealth{}, err
	}
	stylesheetRequest := httptest.NewRequest(http.MethodGet, a.publicBaseURL+stylesheetPath, nil).WithContext(ctx)
	stylesheetResponse := httptest.NewRecorder()
	a.Handler().ServeHTTP(stylesheetResponse, stylesheetRequest)
	if stylesheetResponse.Code != http.StatusOK || !strings.HasPrefix(stylesheetResponse.Header().Get("Content-Type"), "text/css") {
		return InteractionHealth{}, fmt.Errorf("interaction stylesheet is unhealthy")
	}
	if stylesheetResponse.Body.Len() == 0 || stylesheetResponse.Body.Len() > 256<<10 {
		return InteractionHealth{}, fmt.Errorf("interaction stylesheet size is invalid")
	}
	return InteractionHealth{HTMLBytes: len(document), CSSBytes: stylesheetResponse.Body.Len()}, nil
}

func interactionCSPForRedirect(rawRedirectURI string) (string, error) {
	redirectURI, err := url.Parse(rawRedirectURI)
	if err != nil || redirectURI.Scheme == "" || redirectURI.Host == "" || redirectURI.User != nil {
		return "", fmt.Errorf("invalid redirect URI")
	}
	return interactionCSPPrefix + redirectURI.Scheme + "://" + redirectURI.Host + interactionCSPSuffix, nil
}

func declaredStylesheet(document []byte) (string, error) {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return "", fmt.Errorf("parse interaction document: %w", err)
	}
	var stylesheet string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "link" {
			attributes := make(map[string]string, len(node.Attr))
			for _, attribute := range node.Attr {
				attributes[strings.ToLower(attribute.Key)] = attribute.Val
			}
			if attributes["rel"] == "stylesheet" {
				stylesheet = attributes["href"]
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	parsed, err := url.Parse(stylesheet)
	if err != nil || !strings.HasPrefix(stylesheet, "/static/") || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("interaction document has no safe declared stylesheet")
	}
	return stylesheet, nil
}
