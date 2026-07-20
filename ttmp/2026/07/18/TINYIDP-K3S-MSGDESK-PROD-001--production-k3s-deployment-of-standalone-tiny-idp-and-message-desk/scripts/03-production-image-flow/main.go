// Command production-image-flow proves the two production images cooperate
// through an actual TLS-terminating trusted proxy. It intentionally uses only
// disposable state, a checked-in non-secret signup program, and a generated
// owner-only test token.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	idpOrigin     = "https://idp.example.test"
	idpIssuer     = idpOrigin + "/idp"
	messageOrigin = "https://message.example.test"
	proxyAddress  = "127.0.0.1:18443"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "production image flow failed:", err)
		os.Exit(1)
	}
	fmt.Println("production image flow passed")
}

func run(ctx context.Context) error {
	if err := requireFreePort(); err != nil {
		return err
	}
	repoRoot, err := repositoryRoot()
	if err != nil {
		return err
	}
	imageVersion := "phase4-flow"
	if err := command(ctx, repoRoot, "make", "TINYIDP_IMAGE=tinyidp:"+imageVersion, "MESSAGE_DESK_IMAGE=tinyidp-message-desk:"+imageVersion, "IMAGE_VERSION="+imageVersion, "image-build"); err != nil {
		return err
	}

	names := containerNames(os.Getpid())
	if err := command(ctx, repoRoot, "docker", "network", "create", names.network); err != nil {
		return err
	}
	networkCreated := true
	defer func() {
		if networkCreated {
			cleanup(context.Background(), names)
			_ = command(context.Background(), repoRoot, "docker", "network", "rm", names.network)
		}
	}()
	subnet, err := output(ctx, repoRoot, "docker", "network", "inspect", "--format", "{{(index .IPAM.Config 0).Subnet}}", names.network)
	if err != nil {
		return err
	}
	subnet = strings.TrimSpace(subnet)
	if subnet == "" {
		return fmt.Errorf("docker network %s did not report an IPv4 subnet", names.network)
	}

	temporary, err := os.MkdirTemp("", "tinyidp-production-image-flow-")
	if err != nil {
		return fmt.Errorf("create temporary proxy material: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	if err := writeProxyMaterial(temporary); err != nil {
		return err
	}

	program := filepath.Join(repoRoot, "pkg", "idpsignup", "open_signup.js")
	if err := startTinyIDP(ctx, repoRoot, names, subnet, program, imageVersion); err != nil {
		return err
	}
	if err := startProxy(ctx, repoRoot, names, temporary); err != nil {
		return err
	}
	if err := startMessageDesk(ctx, repoRoot, names, subnet, imageVersion); err != nil {
		return err
	}

	browser := newBrowser()
	if err := waitReady(browser, idpOrigin+"/idp/readyz"); err != nil {
		return err
	}
	if err := waitReady(browser, messageOrigin+"/readyz"); err != nil {
		return err
	}
	return exerciseSignupAndMessage(browser)
}

type names struct{ network, idp, message, proxy string }

func containerNames(pid int) names {
	prefix := fmt.Sprintf("tinyidp-phase4-flow-%d", pid)
	return names{network: prefix, idp: prefix + "-idp", message: prefix + "-message", proxy: prefix + "-proxy"}
}

func startTinyIDP(ctx context.Context, root string, names names, subnet, program, version string) error {
	script := "umask 077; head -c 48 /dev/urandom > /run/tinyidp-secrets/token.key; exec tinyidp serve-production --addr :8081 --listener-mode trusted-proxy-http --issuer " + idpIssuer + " --message-desk-origin " + messageOrigin + " --signup-program-file /etc/tinyidp/signup/open-signup.js --db /var/lib/tinyidp/tinyidp.sqlite --audit-path /var/log/tinyidp/audit.jsonl --token-secret-file /run/tinyidp-secrets/token.key --trusted-proxy-cidrs " + subnet
	return command(ctx, root, "docker", "run", "-d", "--name", names.idp, "--network", names.network, "--network-alias", "tinyidp", "--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=16m", "--tmpfs", "/var/lib/tinyidp:uid=65532,gid=65532,mode=0750", "--tmpfs", "/var/log/tinyidp:uid=65532,gid=65532,mode=0750", "--tmpfs", "/run/tinyidp-secrets:uid=65532,gid=65532,mode=0700",
		"-v", program+":/etc/tinyidp/signup/open-signup.js:ro", "--entrypoint", "/bin/sh", "tinyidp:"+version, "-ec", script)
}

func startProxy(ctx context.Context, root string, names names, material string) error {
	return command(ctx, root, "docker", "run", "-d", "--name", names.proxy, "--network", names.network, "--network-alias", "edge-proxy",
		"-p", proxyAddress+":443", "-v", filepath.Join(material, "nginx.conf")+":/etc/nginx/nginx.conf:ro", "-v", filepath.Join(material, "tls.crt")+":/etc/nginx/tls/tls.crt:ro", "-v", filepath.Join(material, "tls.key")+":/etc/nginx/tls/tls.key:ro", "nginx:1.27-alpine")
}

func startMessageDesk(ctx context.Context, root string, names names, subnet, version string) error {
	script := "tinyidp-message-desk init --state-root /var/lib/tinyidp-message-desk --public-base-url " + messageOrigin + "; exec tinyidp-message-desk serve --state-root /var/lib/tinyidp-message-desk --addr :8080 --listener-mode trusted-proxy-http --trusted-proxy-cidrs " + subnet + " --external-issuer " + idpIssuer + " --external-backchannel-url http://" + names.proxy + ":8080/idp"
	return command(ctx, root, "docker", "run", "-d", "--name", names.message, "--network", names.network, "--network-alias", "message-desk", "--read-only", "--tmpfs", "/tmp:rw,noexec,nosuid,size=16m", "--tmpfs", "/var/lib/tinyidp-message-desk:uid=65532,gid=65532,mode=0750", "--entrypoint", "/bin/sh", "tinyidp-message-desk:"+version, "-ec", script)
}

func writeProxyMaterial(directory string) error {
	certificate, key, err := certificate()
	if err != nil {
		return err
	}
	for name, value := range map[string]struct {
		value []byte
		mode  os.FileMode
	}{
		"tls.crt": {certificate, 0o644}, "tls.key": {key, 0o600},
	} {
		if err := os.WriteFile(filepath.Join(directory, name), value.value, value.mode); err != nil {
			return fmt.Errorf("write proxy %s: %w", name, err)
		}
	}
	config := `events {}
http {
	  resolver 127.0.0.11 ipv6=off valid=10s;
  server {
    listen 443 ssl;
    server_name idp.example.test;
    ssl_certificate /etc/nginx/tls/tls.crt;
    ssl_certificate_key /etc/nginx/tls/tls.key;
    location / { set $tinyidp_upstream tinyidp:8081; proxy_pass http://$tinyidp_upstream; proxy_set_header Host idp.example.test; proxy_set_header X-Forwarded-Proto https; proxy_set_header X-Forwarded-Host idp.example.test; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
  }
  server {
    listen 443 ssl;
    server_name message.example.test;
    ssl_certificate /etc/nginx/tls/tls.crt;
    ssl_certificate_key /etc/nginx/tls/tls.key;
    location / { set $message_desk_upstream message-desk:8080; proxy_pass http://$message_desk_upstream; proxy_set_header Host message.example.test; proxy_set_header X-Forwarded-Proto https; proxy_set_header X-Forwarded-Host message.example.test; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
  }
  server {
    listen 8080;
    location / { set $tinyidp_upstream tinyidp:8081; proxy_pass http://$tinyidp_upstream; proxy_set_header Host idp.example.test; proxy_set_header X-Forwarded-Proto https; proxy_set_header X-Forwarded-Host idp.example.test; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; }
  }
}
`
	if err := os.WriteFile(filepath.Join(directory, "nginx.conf"), []byte(config), 0o600); err != nil {
		return fmt.Errorf("write proxy config: %w", err)
	}
	return nil
}

func certificate() ([]byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate proxy key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("generate proxy serial: %w", err)
	}
	now := time.Now()
	template := x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "tiny-idp Phase 4 image flow"}, DNSNames: []string{"idp.example.test", "message.example.test"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("generate proxy certificate: %w", err)
	}
	keyDER := x509.MarshalPKCS1PrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER}), nil
}

type browser struct {
	client  *http.Client
	cookies map[string]map[string]*http.Cookie
}

func newBrowser() *browser {
	return &browser{cookies: map[string]map[string]*http.Cookie{}, client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}}
}

func (b *browser) get(rawURL string) (*http.Response, error) {
	return b.do(http.MethodGet, rawURL, nil, "", nil)
}
func (b *browser) postForm(rawURL string, form url.Values) (*http.Response, error) {
	return b.do(http.MethodPost, rawURL, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil)
}
func (b *browser) postJSON(rawURL string, body any, headers http.Header) (*http.Response, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return b.do(http.MethodPost, rawURL, bytes.NewReader(encoded), "application/json", headers)
}

func (b *browser) do(method, rawURL string, body io.Reader, contentType string, headers http.Header) (*http.Response, error) {
	public, err := url.Parse(rawURL)
	if err != nil || public.Scheme != "https" || (public.Host != "idp.example.test" && public.Host != "message.example.test") {
		return nil, fmt.Errorf("invalid public browser URL %q", rawURL)
	}
	target := *public
	target.Host = proxyAddress
	request, err := http.NewRequest(method, target.String(), body)
	if err != nil {
		return nil, err
	}
	request.Host = public.Host
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if method != http.MethodGet && method != http.MethodHead {
		request.Header.Set("Origin", public.Scheme+"://"+public.Host)
	}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	if values := b.cookies[public.Host]; len(values) != 0 {
		values := make([]string, 0, len(values))
		for _, cookie := range b.cookies[public.Host] {
			values = append(values, cookie.Name+"="+cookie.Value)
		}
		request.Header.Set("Cookie", strings.Join(values, "; "))
	}
	response, err := b.client.Do(request)
	if err != nil {
		return nil, err
	}
	if b.cookies[public.Host] == nil {
		b.cookies[public.Host] = map[string]*http.Cookie{}
	}
	for _, cookie := range response.Cookies() {
		if cookie.MaxAge < 0 {
			delete(b.cookies[public.Host], cookie.Name)
		} else {
			cookieCopy := *cookie
			b.cookies[public.Host][cookie.Name] = &cookieCopy
		}
	}
	return response, nil
}

func waitReady(browser *browser, rawURL string) error {
	var last error
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
		response, err := browser.get(rawURL)
		if err == nil && response.StatusCode == http.StatusOK {
			response.Body.Close()
			return nil
		}
		if err == nil {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			last = fmt.Errorf("%s returned %d: %s", rawURL, response.StatusCode, strings.TrimSpace(string(body)))
		} else {
			last = err
		}
	}
	return fmt.Errorf("wait for readiness: %w", last)
}

func exerciseSignupAndMessage(browser *browser) error {
	registration, err := browser.get(messageOrigin + "/auth/register?return_to=/messages")
	if err != nil {
		return err
	}
	if err := status(registration, http.StatusSeeOther); err != nil {
		return err
	}
	authorizeURL, err := location(registration)
	if err != nil {
		return err
	}
	if err := registrationContract(authorizeURL); err != nil {
		return err
	}
	formPage, err := browser.get(authorizeURL)
	if err != nil {
		return err
	}
	if err := status(formPage, http.StatusOK); err != nil {
		return err
	}
	page, err := io.ReadAll(formPage.Body)
	formPage.Body.Close()
	if err != nil {
		return err
	}
	form := url.Values{"action": {"submit"}, "interaction": {hidden(page, "interaction")}, "workflow_continuation": {hidden(page, "workflow_continuation")}, "csrf_token": {hidden(page, "csrf_token")}, "display_name": {"Container Ada"}, "email": {"container-ada@example.test"}, "password": {"correct horse battery staple 2026"}, "password_confirmation": {"correct horse battery staple 2026"}}
	completed, err := browser.postForm(idpIssuer+"/authorize", form)
	if err != nil {
		return err
	}
	if err := status(completed, http.StatusOK); err != nil {
		return err
	}
	consentPage, err := io.ReadAll(completed.Body)
	completed.Body.Close()
	if err != nil {
		return fmt.Errorf("read authorization consent: %w", err)
	}
	approval, err := browser.postForm(idpIssuer+"/authorize", url.Values{"action": {"approve"}, "interaction": {hidden(consentPage, "interaction")}, "csrf_token": {hidden(consentPage, "csrf_token")}})
	if err != nil {
		return err
	}
	if err := status(approval, http.StatusSeeOther); err != nil {
		return err
	}
	callbackURL, err := location(approval)
	if err != nil {
		return err
	}
	callback, err := url.Parse(callbackURL)
	if err != nil || callback.Scheme != "https" || callback.Host != "message.example.test" || callback.Path != "/auth/callback" || callback.Query().Get("code") == "" || callback.Query().Get("state") == "" {
		return fmt.Errorf("unexpected signup callback %q", callbackURL)
	}
	finished, err := browser.get(callbackURL)
	if err != nil {
		return err
	}
	if err := status(finished, http.StatusSeeOther); err != nil {
		return err
	}
	returnTo, err := location(finished)
	if err != nil {
		return err
	}
	if returnTo != "/messages" {
		return fmt.Errorf("signup return location = %q, want /messages", returnTo)
	}
	session, err := browser.get(messageOrigin + "/api/session")
	if err != nil {
		return err
	}
	if err := status(session, http.StatusOK); err != nil {
		return err
	}
	var sessionBody struct {
		Authenticated bool   `json:"authenticated"`
		Subject       string `json:"subject"`
		CSRFToken     string `json:"csrfToken"`
	}
	if err := json.NewDecoder(session.Body).Decode(&sessionBody); err != nil {
		session.Body.Close()
		return fmt.Errorf("decode application session: %w", err)
	}
	session.Body.Close()
	if !sessionBody.Authenticated || sessionBody.Subject == "" || sessionBody.CSRFToken == "" {
		return fmt.Errorf("invalid application session after signup")
	}
	messageText := "Hello from the production image flow."
	created, err := browser.postJSON(messageOrigin+"/api/messages", map[string]string{"body": messageText}, http.Header{"X-CSRF-Token": []string{sessionBody.CSRFToken}})
	if err != nil {
		return err
	}
	if err := status(created, http.StatusCreated); err != nil {
		return err
	}
	var createdBody struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(created.Body).Decode(&createdBody); err != nil {
		created.Body.Close()
		return fmt.Errorf("decode created message: %w", err)
	}
	created.Body.Close()
	if createdBody.Body != messageText {
		return fmt.Errorf("created message body = %q, want %q", createdBody.Body, messageText)
	}
	return nil
}

func status(response *http.Response, want int) error {
	if response.StatusCode == want {
		return nil
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	return fmt.Errorf("response status %d, want %d: %s", response.StatusCode, want, strings.TrimSpace(string(body)))
}
func location(response *http.Response) (string, error) {
	defer response.Body.Close()
	value := response.Header.Get("Location")
	if value == "" {
		return "", fmt.Errorf("response %d has no Location", response.StatusCode)
	}
	return value, nil
}
func hidden(page []byte, name string) string {
	match := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]+)"`).FindSubmatch(page)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}
func registrationContract(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "idp.example.test" || parsed.Path != "/idp/authorize" {
		return fmt.Errorf("unexpected authorize URL %q", rawURL)
	}
	query := parsed.Query()
	if query.Get("client_id") != "tinyidp-message-app" || query.Get("redirect_uri") != messageOrigin+"/auth/callback" || query.Get("code_challenge_method") != "S256" || query.Get("tinyidp_signup") != "1" {
		return fmt.Errorf("invalid signup authorization contract: %s", query.Encode())
	}
	return nil
}

func requireFreePort() error {
	command := exec.Command("lsof", "-n", "-iTCP:18443", "-sTCP:LISTEN")
	if err := command.Run(); err == nil {
		return fmt.Errorf("port 18443 is already in use")
	}
	return nil
}
func repositoryRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("locate script source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../../.."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return "", fmt.Errorf("locate repository root %s: %w", root, err)
	}
	return root, nil
}
func command(ctx context.Context, directory, name string, arguments ...string) error {
	value := exec.CommandContext(ctx, name, arguments...)
	value.Dir = directory
	output, err := value.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(arguments, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
func output(ctx context.Context, directory, name string, arguments ...string) (string, error) {
	value := exec.CommandContext(ctx, name, arguments...)
	value.Dir = directory
	result, err := value.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(arguments, " "), err, strings.TrimSpace(string(result)))
	}
	return string(result), nil
}
func cleanup(ctx context.Context, names names) {
	root, err := repositoryRoot()
	if err != nil {
		return
	}
	for _, name := range []string{names.message, names.proxy, names.idp} {
		_ = command(ctx, root, "docker", "rm", "-f", name)
	}
}
