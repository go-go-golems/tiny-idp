package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
	"github.com/manuel/tinyidp/cmd/tinyidp-xapp/internal/resourceauth"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/pkg/errors"
)

const deviceAPIMaxBodyBytes = 8 << 10

// deviceAPIHandler is a host-owned bearer API. JavaScript never receives a
// bearer token or the client secret used to introspect it.
type deviceAPIHandler struct {
	auth    *resourceauth.Authenticator
	objects durableobjects.Dispatcher
	audit   idp.Sink
}

func newDeviceAPIHandler(auth *resourceauth.Authenticator, objects durableobjects.Dispatcher, audit idp.Sink) http.Handler {
	if audit == nil {
		audit = idp.NoopSink{}
	}
	return &deviceAPIHandler{auth: auth, objects: objects, audit: audit}
}

func (h *deviceAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.auth == nil || h.objects == nil {
		writeDeviceAPIError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/device/bbs":
		h.handleBoard(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/device/bbs/posts":
		h.handleCreatePost(w, r)
	default:
		writeDeviceAPIError(w, http.StatusNotFound, "not_found")
	}
}

func (h *deviceAPIHandler) handleBoard(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.authenticate(w, r, []string{"bbs.read"})
	if !ok {
		return
	}
	response, err := h.dispatch(r.Context(), http.MethodGet, "/board", map[string]any{"actorId": principal.Subject, "actorName": principal.Subject})
	if err != nil {
		h.record(r.Context(), "xapp.api.bbs.read_failed", principal, "unavailable")
		writeDeviceAPIError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}
	h.record(r.Context(), "xapp.api.bbs.read", principal, "accepted")
	writeDeviceResponse(w, response)
}

func (h *deviceAPIHandler) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.authenticate(w, r, []string{"bbs.post.create"})
	if !ok {
		return
	}
	var input struct {
		Title    string `json:"title"`
		Body     string `json:"body"`
		Category string `json:"category"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, deviceAPIMaxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeDeviceAPIError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeDeviceAPIError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	response, err := h.dispatch(r.Context(), http.MethodPost, "/posts", map[string]any{
		"title": input.Title, "body": input.Body, "category": input.Category,
		"actorId": principal.Subject, "actorName": principal.Subject,
	})
	if err != nil {
		h.record(r.Context(), "xapp.api.bbs.post_failed", principal, "unavailable")
		writeDeviceAPIError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}
	h.record(r.Context(), "xapp.api.bbs.posted", principal, "accepted")
	writeDeviceResponse(w, response)
}

func (h *deviceAPIHandler) authenticate(w http.ResponseWriter, r *http.Request, scopes []string) (resourceauth.Principal, bool) {
	result := h.auth.Authenticate(r.Context(), r.Header.Values("Authorization"), scopes)
	switch result.Outcome {
	case resourceauth.OutcomeAuthenticated:
		h.record(r.Context(), "xapp.api.auth.accepted", result.Principal, "accepted")
		return result.Principal, true
	case resourceauth.OutcomeForbidden:
		h.record(r.Context(), "xapp.api.auth.rejected", resourceauth.Principal{}, "missing_scope")
		writeDeviceAPIError(w, http.StatusForbidden, "forbidden")
	case resourceauth.OutcomeUnavailable:
		h.record(r.Context(), "xapp.api.auth.unavailable", resourceauth.Principal{}, "provider_unavailable")
		writeDeviceAPIError(w, http.StatusServiceUnavailable, "service_unavailable")
	default:
		h.record(r.Context(), "xapp.api.auth.rejected", resourceauth.Principal{}, "unauthorized")
		w.Header().Set("WWW-Authenticate", `Bearer realm="tinyidp-xapp"`)
		writeDeviceAPIError(w, http.StatusUnauthorized, "unauthorized")
	}
	return resourceauth.Principal{}, false
}

func (h *deviceAPIHandler) dispatch(ctx context.Context, method, path string, body map[string]any) (*durableobjects.FetchResponse, error) {
	id, err := durableobjects.NewObjectID("BBS", "community")
	if err != nil {
		return nil, errors.Wrap(err, "construct BBS object ID")
	}
	result, err := h.objects.Dispatch(ctx, durableobjects.Envelope{
		Kind: durableobjects.KindFetch,
		ID:   id,
		Request: &durableobjects.FetchRequest{
			Method: method, URL: "device-api:" + path, Path: path, Body: body,
			Headers: map[string]string{"X-Actor-Kind": "oidc_bearer"},
		},
		Deadline: time.Now().UTC().Add(10 * time.Second),
	})
	if err != nil {
		return nil, errors.Wrap(err, "dispatch BBS object")
	}
	if result.Response == nil {
		return nil, errors.New("BBS object returned no fetch response")
	}
	return result.Response, nil
}

func (h *deviceAPIHandler) record(ctx context.Context, name string, principal resourceauth.Principal, reason string) {
	if h.audit == nil {
		return
	}
	result := "accepted"
	if reason != "accepted" {
		result = "rejected"
	}
	_ = h.audit.Emit(ctx, idp.Event{Time: time.Now().UTC(), Name: name, ClientID: principal.ClientID, Subject: principal.Subject, Result: result, Reason: reason, Fields: map[string]string{"credential_kind": "oidc_bearer"}})
}

func writeDeviceResponse(w http.ResponseWriter, response *durableobjects.FetchResponse) {
	for key, value := range response.Headers {
		if strings.EqualFold(key, "content-type") || strings.EqualFold(key, "cache-control") {
			w.Header().Set(key, value)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Status)
	_ = json.NewEncoder(w).Encode(response.Body)
}

func writeDeviceAPIError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
