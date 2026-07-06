package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/user"
)

const (
	deviceGrantType       = "urn:ietf:params:oauth:grant-type:device_code"
	defaultDeviceInterval = 5 * time.Second
	defaultDeviceTTL      = 10 * time.Minute
	userCodeAlphabet      = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type deviceGrantStatus string

const (
	devicePending  deviceGrantStatus = "pending"
	deviceApproved deviceGrantStatus = "approved"
	deviceDenied   deviceGrantStatus = "denied"
)

type deviceGrant struct {
	DeviceCode string
	UserCode   string

	ClientID string
	Scope    string
	Expires  time.Time
	Interval time.Duration

	Status deviceGrantStatus

	User     user.User
	Scenario *scenario.Scenario
	AuthTime time.Time

	LastPoll      time.Time
	SlowDownCount int
}

type devicePageData struct {
	UserCode  string
	LoginHint string
	Message   string
}

func (s *Server) deviceAuthorization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tokenError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}

	clientID, c, ok := s.authenticateOAuthClient(w, r)
	if !ok {
		return
	}

	scope := strings.TrimSpace(r.Form.Get("scope"))
	if scope == "" {
		scope = "openid profile email"
	}
	if !hasScope(scope, "openid") || !c.AllowsScope(scope) {
		tokenError(w, http.StatusBadRequest, "invalid_scope", "scope not allowed")
		return
	}

	grant := s.newDeviceGrant(clientID, scope, time.Now())
	s.mu.Lock()
	for {
		if _, exists := s.deviceGrants[grant.DeviceCode]; !exists && !s.userCodeExistsLocked(grant.UserCode) {
			break
		}
		grant = s.newDeviceGrant(clientID, scope, time.Now())
	}
	s.deviceGrants[grant.DeviceCode] = grant
	s.mu.Unlock()

	verificationURI := s.issuer + "/device"
	complete, _ := url.Parse(verificationURI)
	q := complete.Query()
	q.Set("user_code", grant.UserCode)
	complete.RawQuery = q.Encode()

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{
		"device_code":               grant.DeviceCode,
		"user_code":                 grant.UserCode,
		"verification_uri":          verificationURI,
		"verification_uri_complete": complete.String(),
		"expires_in":                int(defaultDeviceTTL.Seconds()),
		"interval":                  int(defaultDeviceInterval.Seconds()),
	})
}

func (s *Server) newDeviceGrant(clientID, scope string, now time.Time) deviceGrant {
	return deviceGrant{
		DeviceCode: randomB64(32),
		UserCode:   generateUserCode(),
		ClientID:   clientID,
		Scope:      scope,
		Expires:    now.Add(defaultDeviceTTL),
		Interval:   defaultDeviceInterval,
		Status:     devicePending,
	}
}

func generateUserCode() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = userCodeAlphabet[int(randomB64Byte())%len(userCodeAlphabet)]
	}
	return string(b[:4]) + "-" + string(b[4:])
}

func randomB64Byte() byte {
	// randomB64 already panics on CSPRNG failure. Decode a one-byte random string
	// would be wasteful, so keep this tiny helper next to device-code generation.
	return randomBytes(1)[0]
}

func normalizeUserCode(value string) string {
	value = strings.ToUpper(value)
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func displayUserCode(value string) string {
	n := normalizeUserCode(value)
	if len(n) <= 4 {
		return n
	}
	return n[:4] + "-" + n[4:]
}

func (s *Server) userCodeExistsLocked(userCode string) bool {
	normalized := normalizeUserCode(userCode)
	for _, grant := range s.deviceGrants {
		if normalizeUserCode(grant.UserCode) == normalized {
			return true
		}
	}
	return false
}

func (s *Server) findDeviceGrantByUserCodeLocked(userCode string) (string, deviceGrant, bool) {
	normalized := normalizeUserCode(userCode)
	for deviceCode, grant := range s.deviceGrants {
		if normalizeUserCode(grant.UserCode) == normalized {
			return deviceCode, grant, true
		}
	}
	return "", deviceGrant{}, false
}

func (s *Server) device(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.deviceGET(w, r, "")
	case http.MethodPost:
		s.devicePOST(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) deviceGET(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = devicePage.Execute(w, devicePageData{
		UserCode: displayUserCode(r.URL.Query().Get("user_code")),
		Message:  message,
	})
}

func (s *Server) devicePOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderDeviceMessage(w, r, "invalid form")
		return
	}

	userCode := r.PostForm.Get("user_code")
	now := time.Now()

	s.mu.Lock()
	deviceCode, grant, ok := s.findDeviceGrantByUserCodeLocked(userCode)
	s.mu.Unlock()
	if !ok || now.After(grant.Expires) {
		s.renderDeviceMessage(w, r, "unknown or expired user code")
		return
	}

	switch r.PostForm.Get("action") {
	case "deny":
		s.mu.Lock()
		grant.Status = deviceDenied
		s.deviceGrants[deviceCode] = grant
		s.mu.Unlock()
		s.renderDeviceDone(w, "Device request denied.")
		return
	case "approve", "":
		// Continue below. Empty action defaults to approve for simple forms/tests.
	default:
		s.renderDeviceMessage(w, r, "unknown action")
		return
	}

	login := user.Normalize(r.PostForm.Get("login"))
	if login == "" {
		s.renderDeviceMessage(w, r, "login is required")
		return
	}
	sc, _ := s.registry.Lookup(login)
	if !passwordAccepted(sc, r.PostForm.Get("password")) {
		s.renderDeviceMessage(w, r, "invalid login or password")
		return
	}
	if sc.AuthError != "" {
		s.renderDeviceMessage(w, r, fmt.Sprintf("login scenario cannot approve device request: %s", sc.AuthError))
		return
	}

	s.mu.Lock()
	grant.Status = deviceApproved
	grant.User = sc.User
	grant.Scenario = &sc
	grant.AuthTime = now
	s.deviceGrants[deviceCode] = grant
	s.mu.Unlock()

	s.renderDeviceDone(w, "Device request approved. You can return to the device.")
}

func (s *Server) renderDeviceMessage(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = devicePage.Execute(w, devicePageData{
		UserCode:  displayUserCode(r.PostForm.Get("user_code")),
		LoginHint: user.Normalize(r.PostForm.Get("login")),
		Message:   message,
	})
}

func (s *Server) renderDeviceDone(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><html><head><meta charset=\"utf-8\"><title>Tiny IdP Device Approval</title></head><body><h1>" + message + "</h1></body></html>"))
}
