// Package twoprocessharness proves the deployable Tiny-IDP and Message Desk
// binaries cooperate without sharing an in-process provider or durable state.
package twoprocessharness

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
)

const (
	idpPublicOrigin     = "https://idp.example.test"
	idpIssuer           = idpPublicOrigin + "/idp"
	messagePublicOrigin = "https://message.example.test"
)

// TestTwoProcessLifecycle starts the actual command binaries. It deliberately
// keeps the browser-visible origins HTTPS while the test is the trusted local
// proxy that terminates TLS and forwards to the private HTTP listeners.
func TestTwoProcessLifecycle(t *testing.T) {
	t.Parallel()
	harness := newHarness(t)
	harness.build()
	harness.initializeMessageDesk()
	harness.startTinyIDP()
	harness.waitReady(harness.idpAddress, idpPublicOrigin, "/idp/readyz")
	harness.startIDPProxy()
	harness.startMessageDesk()
	harness.waitReady(harness.messageAddress, messagePublicOrigin, "/readyz")

	harness.requireAudit("tinyidp", "identity.bootstrap.client_created")
	harness.requireLog("tinyidp", "tinyidp production host listening")
	harness.requireLog("message-desk", "message application listening")
}

type harness struct {
	t              *testing.T
	root           string
	repo           string
	binRoot        string
	idpAddress     string
	messageAddress string
	idpLog         string
	messageLog     string
	idpAudit       string
	messageState   string
	idpProxy       *httptest.Server
	processes      []*startedProcess
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	repo := repositoryRoot(t)
	root := t.TempDir()
	return &harness{
		t: t, root: root, repo: repo, binRoot: filepath.Join(root, "bin"),
		idpAddress: unusedLoopbackAddress(t), messageAddress: unusedLoopbackAddress(t),
		idpLog:       filepath.Join(root, "logs", "tinyidp.log"),
		messageLog:   filepath.Join(root, "logs", "message-desk.log"),
		idpAudit:     filepath.Join(root, "tinyidp", "audit", "events.jsonl"),
		messageState: filepath.Join(root, "message-desk"),
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate harness source")
	}
	directory := filepath.Dir(file)
	for {
		if info, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil && !info.IsDir() {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("could not locate repository go.mod")
		}
		directory = parent
	}
}

func unusedLoopbackAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate loopback port: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release loopback port: %v", err)
	}
	return address
}

func (h *harness) build() {
	h.t.Helper()
	if err := os.MkdirAll(h.binRoot, 0o700); err != nil {
		h.t.Fatalf("create binary directory: %v", err)
	}
	h.runForeground("build tinyidp", "go", "build", "-o", filepath.Join(h.binRoot, "tinyidp"), "./cmd/tinyidp")
	h.runForeground("build message desk", "go", "build", "-o", filepath.Join(h.binRoot, "message-desk"), "./examples/tinyidp-message-app")
}

func (h *harness) initializeMessageDesk() {
	h.t.Helper()
	h.runForeground("initialize message desk", filepath.Join(h.binRoot, "message-desk"), "init",
		"--state-root", h.messageState, "--public-base-url", messagePublicOrigin)
}

func (h *harness) startTinyIDP() {
	h.t.Helper()
	program := filepath.Join(h.root, "tinyidp", "signup.js")
	secret := filepath.Join(h.root, "tinyidp", "secrets", "token.key")
	if err := os.MkdirAll(filepath.Dir(program), 0o700); err != nil {
		h.t.Fatalf("create Tiny-IDP state directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(secret), 0o700); err != nil {
		h.t.Fatalf("create Tiny-IDP secret directory: %v", err)
	}
	if err := os.WriteFile(program, []byte(idpsignup.DefaultSource), 0o600); err != nil {
		h.t.Fatalf("write checked signup program: %v", err)
	}
	if err := os.WriteFile(secret, bytes.Repeat([]byte{0x42}, 32), 0o600); err != nil {
		h.t.Fatalf("write Tiny-IDP test secret: %v", err)
	}
	h.start("tinyidp", h.idpLog, filepath.Join(h.binRoot, "tinyidp"), "serve-production",
		"--addr", h.idpAddress,
		"--listener-mode", "trusted-proxy-http",
		"--issuer", idpIssuer,
		"--message-desk-origin", messagePublicOrigin,
		"--signup-program-file", program,
		"--db", filepath.Join(h.root, "tinyidp", "state", "tinyidp.sqlite"),
		"--audit-path", h.idpAudit,
		"--token-secret-file", secret,
		"--trusted-proxy-cidrs", "127.0.0.1/32")
}

func (h *harness) startMessageDesk() {
	h.t.Helper()
	h.start("message-desk", h.messageLog, filepath.Join(h.binRoot, "message-desk"), "serve",
		"--state-root", h.messageState,
		"--addr", h.messageAddress,
		"--listener-mode", "trusted-proxy-http",
		"--trusted-proxy-cidrs", "127.0.0.1/32",
		"--external-issuer", idpIssuer,
		"--external-backchannel-url", h.idpProxy.URL+"/idp")
}

// startIDPProxy models the only component permitted to translate external
// HTTPS into the IdP's private trusted-proxy listener. It preserves the public
// Host and adds the exact forwarding metadata that the production Traefik
// listener requires; it is not an application shim.
func (h *harness) startIDPProxy() {
	h.t.Helper()
	transport := &http.Transport{}
	h.idpProxy = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		forwarded := request.Clone(request.Context())
		forwarded.URL.Scheme = "http"
		forwarded.URL.Host = h.idpAddress
		forwarded.Host = "idp.example.test"
		forwarded.Header = request.Header.Clone()
		forwarded.Header.Set("X-Forwarded-Proto", "https")
		forwarded.Header.Set("X-Forwarded-Host", "idp.example.test")
		response, err := transport.RoundTrip(forwarded)
		if err != nil {
			http.Error(w, "forward to Tiny-IDP", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		for key, values := range response.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, response.Body)
	}))
	h.t.Cleanup(func() {
		if h.idpProxy != nil {
			h.idpProxy.Close()
		}
		transport.CloseIdleConnections()
	})
}

func (h *harness) start(name, logPath, binary string, args ...string) {
	h.t.Helper()
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		h.t.Fatalf("create %s log directory: %v", name, err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		h.t.Fatalf("open %s log: %v", name, err)
	}
	command := exec.Command(binary, args...)
	command.Dir = h.repo
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		h.t.Fatalf("start %s: %v", name, err)
	}
	process := &startedProcess{name: name, command: command, logFile: logFile, logPath: logPath}
	h.processes = append(h.processes, process)
	h.t.Cleanup(func() { process.stop(h.t) })
}

func (h *harness) waitReady(address, publicOrigin, path string) {
	h.t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		response, err := h.proxyRequest(http.MethodGet, address, publicOrigin, path, nil)
		if err == nil && response.StatusCode == http.StatusOK {
			response.Body.Close()
			return
		}
		if err == nil {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			last = fmt.Errorf("status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
		} else {
			last = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	h.t.Fatalf("wait for %s readiness: %v\nTiny-IDP log:\n%s\nMessage Desk log:\n%s", publicOrigin, last, readFile(h.idpLog), readFile(h.messageLog))
}

func (h *harness) proxyRequest(method, address, publicOrigin, requestPath string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, "http://"+address+requestPath, body)
	if err != nil {
		return nil, err
	}
	publicURL, err := neturl(publicOrigin)
	if err != nil {
		return nil, err
	}
	request.Host = publicURL
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", publicURL)
	return (&http.Client{Timeout: 2 * time.Second}).Do(request)
}

func neturl(origin string) (string, error) {
	if !strings.HasPrefix(origin, "https://") || strings.Contains(strings.TrimPrefix(origin, "https://"), "/") {
		return "", fmt.Errorf("invalid test public origin %q", origin)
	}
	return strings.TrimPrefix(origin, "https://"), nil
}

func (h *harness) requireAudit(processName, event string) {
	h.t.Helper()
	path := h.idpAudit
	if processName != "tinyidp" {
		h.t.Fatalf("unknown audit process %q", processName)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(readFile(path), event) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	h.t.Fatalf("audit %s does not contain %q: %s", processName, event, readFile(path))
}

func (h *harness) requireLog(processName, fragment string) {
	h.t.Helper()
	path := h.idpLog
	if processName == "message-desk" {
		path = h.messageLog
	} else if processName != "tinyidp" {
		h.t.Fatalf("unknown log process %q", processName)
	}
	if !strings.Contains(readFile(path), fragment) {
		h.t.Fatalf("%s log does not contain %q: %s", processName, fragment, readFile(path))
	}
}

func (h *harness) runForeground(label, binary string, args ...string) {
	h.t.Helper()
	context, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(context, binary, args...)
	command.Dir = h.repo
	output, err := command.CombinedOutput()
	if err != nil {
		h.t.Fatalf("%s: %v\n%s", label, err, output)
	}
}

type startedProcess struct {
	name    string
	command *exec.Cmd
	logFile *os.File
	logPath string
}

func (p *startedProcess) stop(t *testing.T) {
	t.Helper()
	if p == nil || p.command == nil || p.command.Process == nil {
		return
	}
	_ = p.command.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- p.command.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("stop %s: %v\n%s", p.name, err, readFile(p.logPath))
		}
	case <-time.After(10 * time.Second):
		_ = p.command.Process.Kill()
		<-done
		t.Errorf("stop %s: forced kill after graceful shutdown deadline\n%s", p.name, readFile(p.logPath))
	}
	if p.logFile != nil {
		if err := p.logFile.Close(); err != nil {
			t.Errorf("close %s log: %v", p.name, err)
		}
	}
}

func readFile(path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "<unavailable: " + err.Error() + ">"
	}
	return string(contents)
}
