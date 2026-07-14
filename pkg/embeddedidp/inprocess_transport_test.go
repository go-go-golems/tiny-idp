package embeddedidp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestInProcessIssuerTransportRoundTripsRequestAndResponse(t *testing.T) {
	var requestBody, requestURI, remoteAddr string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		requestBody = string(body)
		requestURI = r.RequestURI
		remoteAddr = r.RemoteAddr
		if r.URL.IsAbs() {
			t.Errorf("handler received client URL: %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "one=1")
		w.Header().Add("Set-Cookie", "two=2")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	transport, err := NewInProcessIssuerTransport("https://identity.example.test/idp", handler, InProcessTransportOptions{RemoteAddr: "127.0.0.2:4242"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "https://identity.example.test/idp/token?trace=1", strings.NewReader("grant_type=authorization_code"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if requestBody != "grant_type=authorization_code" || requestURI != "/idp/token?trace=1" || remoteAddr != "127.0.0.2:4242" {
		t.Fatalf("handler metadata body=%q URI=%q remote=%q", requestBody, requestURI, remoteAddr)
	}
	if response.StatusCode != http.StatusCreated || response.Header.Get("Content-Type") != "application/json" || len(response.Header.Values("Set-Cookie")) != 2 || string(body) != `{"ok":true}` || response.Request != request {
		t.Fatalf("response = %#v body=%q", response, body)
	}
}

func TestInProcessIssuerTransportFailsClosed(t *testing.T) {
	var calls atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	})
	transport, err := NewInProcessIssuerTransport("https://identity.example.test/idp", handler, InProcessTransportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, rawURL := range []string{
		"https://other.example.test/idp/keys",
		"http://identity.example.test/idp/keys",
		"https://identity.example.test/application",
		"https://identity.example.test/idp-other",
		"https://user@identity.example.test/idp/keys",
		"https://identity.example.test/idp//keys",
		"https://identity.example.test/idp/../application",
		"https://identity.example.test/idp/%2e%2e/application",
		"https://identity.example.test/idp%2fapplication",
		"https://identity.example.test/idp/%5ckeys",
		"/idp/keys",
	} {
		request, requestErr := http.NewRequest(http.MethodGet, rawURL, nil)
		if requestErr != nil {
			continue
		}
		if _, err := transport.RoundTrip(request); err == nil {
			t.Errorf("RoundTrip(%q) succeeded", rawURL)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("rejected requests reached handler %d times", calls.Load())
	}

	request, err := http.NewRequest(http.MethodGet, "https://identity.example.test/idp/keys", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if calls.Load() != 1 {
		t.Fatalf("allowed request calls = %d", calls.Load())
	}
}

func TestNewInProcessIssuerTransportRejectsNoncanonicalIssuer(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, issuer := range []string{
		"", "/idp", "ftp://identity.example.test/idp", "https://user@identity.example.test/idp",
		"https://identity.example.test/idp?q=1", "https://identity.example.test/idp#fragment",
		"https://identity.example.test/idp/", "https://identity.example.test/idp//nested",
		"https://identity.example.test/idp/../other", "https://identity.example.test/idp/%2e%2e/other",
		"https://identity.example.test/idp%2fother", "https://identity.example.test/idp\\other",
	} {
		if _, err := NewInProcessIssuerTransport(issuer, handler, InProcessTransportOptions{}); err == nil {
			t.Errorf("issuer %q was accepted", issuer)
		}
	}
	if _, err := NewInProcessIssuerTransport("https://identity.example.test/idp", nil, InProcessTransportOptions{}); err == nil {
		t.Fatal("nil handler was accepted")
	}
	if _, err := NewInProcessIssuerTransport("https://identity.example.test/idp", handler, InProcessTransportOptions{MaxResponseBytes: -1}); err == nil {
		t.Fatal("negative response limit was accepted")
	}
}

func TestInProcessIssuerTransportEnforcesResponseBoundWhenHandlerIgnoresError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "1234")
		_, _ = io.WriteString(w, "567890")
	})
	transport, err := NewInProcessIssuerTransport("https://identity.example.test/idp", handler, InProcessTransportOptions{MaxResponseBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, "https://identity.example.test/idp/large", nil)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := transport.RoundTrip(request); err == nil || response != nil || !strings.Contains(err.Error(), "exceeded 8 bytes") {
		t.Fatalf("response = %#v, error = %v", response, err)
	}
}

func TestInProcessIssuerTransportUsesDefaultResponseBound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, DefaultInProcessResponseLimit+1))
	})
	transport, err := NewInProcessIssuerTransport("https://identity.example.test", handler, InProcessTransportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodGet, "https://identity.example.test/discovery", nil)
	if _, err := transport.RoundTrip(request); err == nil {
		t.Fatal("oversized default response succeeded")
	}
}

func TestInProcessIssuerTransportPropagatesCancellation(t *testing.T) {
	started := make(chan struct{})
	// Use a context-aware handler so the synchronous RoundTripper can return as
	// soon as the handler observes cancellation.
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	transport, err := NewInProcessIssuerTransport("https://identity.example.test/idp", handler, InProcessTransportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://identity.example.test/idp/slow", nil)
	done := make(chan error, 1)
	go func() {
		_, err := transport.RoundTrip(request)
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RoundTrip did not return after cancellation")
	}
}
