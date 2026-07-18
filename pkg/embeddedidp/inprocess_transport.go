package embeddedidp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

const DefaultInProcessResponseLimit int64 = 1 << 20

type InProcessTransportOptions struct {
	MaxResponseBytes int64
	RemoteAddr       string
}

// InProcessIssuerTransport dispatches exact-issuer back-channel HTTP requests
// to an in-process provider handler. It has no network fallback.
type InProcessIssuerTransport struct {
	scheme           string
	host             string
	pathPrefix       string
	handler          http.Handler
	maxResponseBytes int64
	remoteAddr       string
}

var _ http.RoundTripper = (*InProcessIssuerTransport)(nil)

func NewInProcessIssuerTransport(issuer string, handler http.Handler, opts InProcessTransportOptions) (*InProcessIssuerTransport, error) {
	if handler == nil {
		return nil, fmt.Errorf("in-process issuer handler is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil {
		return nil, fmt.Errorf("parse in-process issuer: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("in-process issuer scheme must be http or https")
	}
	if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return nil, fmt.Errorf("in-process issuer must be an absolute origin and canonical path without userinfo, query, or fragment")
	}
	issuerPath, err := canonicalInProcessPath(parsed)
	if err != nil {
		return nil, fmt.Errorf("invalid in-process issuer path: %w", err)
	}
	if issuerPath != "/" && strings.HasSuffix(issuerPath, "/") {
		return nil, fmt.Errorf("in-process issuer path must not have a trailing slash")
	}
	prefix := issuerPath
	if prefix == "/" {
		prefix = ""
	}
	limit := opts.MaxResponseBytes
	if limit == 0 {
		limit = DefaultInProcessResponseLimit
	}
	if limit < 1 {
		return nil, fmt.Errorf("in-process response limit must be positive")
	}
	remoteAddr := strings.TrimSpace(opts.RemoteAddr)
	if remoteAddr == "" {
		remoteAddr = "127.0.0.1:0"
	}
	if strings.ContainsAny(remoteAddr, "\x00\r\n") {
		return nil, fmt.Errorf("in-process remote address contains control characters")
	}
	return &InProcessIssuerTransport{
		scheme: strings.ToLower(parsed.Scheme), host: strings.ToLower(parsed.Host), pathPrefix: prefix,
		handler: handler, maxResponseBytes: limit, remoteAddr: remoteAddr,
	}, nil
}

func (t *InProcessIssuerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t == nil || t.handler == nil || t.maxResponseBytes < 1 {
		return nil, fmt.Errorf("in-process issuer transport is not initialized")
	}
	if request == nil || request.URL == nil || !request.URL.IsAbs() {
		return nil, fmt.Errorf("in-process issuer request must use an absolute URL")
	}
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	if request.URL.User != nil || request.URL.Opaque != "" || request.URL.Fragment != "" ||
		strings.ToLower(request.URL.Scheme) != t.scheme || strings.ToLower(request.URL.Host) != t.host {
		return nil, fmt.Errorf("in-process issuer request origin is not allowed")
	}
	requestPath, err := canonicalInProcessPath(request.URL)
	if err != nil {
		return nil, fmt.Errorf("in-process issuer request path is not canonical")
	}
	if t.pathPrefix != "" && requestPath != t.pathPrefix && !strings.HasPrefix(requestPath, t.pathPrefix+"/") {
		return nil, fmt.Errorf("in-process issuer request path is outside the issuer")
	}

	serverRequest := request.Clone(request.Context())
	serverRequest.RequestURI = request.URL.RequestURI()
	serverRequest.RemoteAddr = t.remoteAddr
	serverRequest.URL.Scheme = ""
	serverRequest.URL.Host = ""
	serverRequest.URL.Path = requestPath
	if request.Body != nil {
		defer request.Body.Close()
	}
	writer := newBoundedResponseWriter(t.maxResponseBytes)
	t.handler.ServeHTTP(writer, serverRequest)
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	if writer.overflow {
		return nil, fmt.Errorf("in-process issuer response exceeded %d bytes", t.maxResponseBytes)
	}
	return writer.response(request), nil
}

func canonicalInProcessPath(value *url.URL) (string, error) {
	escaped := value.EscapedPath()
	decoded := value.Path
	if escaped == "" {
		escaped = "/"
		decoded = "/"
	}
	lowerEscaped := strings.ToLower(escaped)
	if strings.Contains(decoded, "\\") || strings.Contains(lowerEscaped, "%2f") || strings.Contains(lowerEscaped, "%5c") || strings.Contains(lowerEscaped, "%2e") {
		return "", errors.New("encoded separator, backslash, or dot segment is not allowed")
	}
	if !strings.HasPrefix(decoded, "/") {
		return "", errors.New("path must be absolute")
	}
	if clean := path.Clean(decoded); clean != decoded {
		return "", errors.New("path is not canonical")
	}
	return decoded, nil
}

type boundedResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	statusCode  int
	limit       int64
	overflow    bool
	wroteHeader bool
}

var _ http.ResponseWriter = (*boundedResponseWriter)(nil)

func newBoundedResponseWriter(limit int64) *boundedResponseWriter {
	return &boundedResponseWriter{header: make(http.Header), limit: limit}
}

func (w *boundedResponseWriter) Header() http.Header { return w.header }

func (w *boundedResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = statusCode
}

func (w *boundedResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	remaining := w.limit - int64(w.body.Len())
	if remaining <= 0 {
		w.overflow = true
		return 0, fmt.Errorf("response limit exceeded")
	}
	if int64(len(p)) <= remaining {
		return w.body.Write(p)
	}
	written, _ := w.body.Write(p[:remaining])
	w.overflow = true
	return written, fmt.Errorf("response limit exceeded")
}

func (w *boundedResponseWriter) response(request *http.Request) *http.Response {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	body := append([]byte(nil), w.body.Bytes()...)
	header := w.header.Clone()
	return &http.Response{
		Status: strconv.Itoa(w.statusCode) + " " + http.StatusText(w.statusCode), StatusCode: w.statusCode,
		Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, Header: header,
		Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: request,
	}
}
