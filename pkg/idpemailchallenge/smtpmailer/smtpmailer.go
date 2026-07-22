// Package smtpmailer delivers TinyIDP email challenges through a reviewed
// SMTP endpoint and fixed native templates.
package smtpmailer

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
)

type TLSMode string

const (
	TLSStartTLS         TLSMode = "starttls"
	TLSImplicit         TLSMode = "implicit"
	TLSPrivatePlaintext TLSMode = "private-plaintext"
)

type Template struct {
	Subject string
	Render  func(idpemailchallenge.MailRequest) (string, error)
}

type Config struct {
	Address        string
	TLSMode        TLSMode
	ServerName     string
	Username       string
	Password       []byte
	FromAddress    string
	FromName       string
	ConnectTimeout time.Duration
	SendTimeout    time.Duration
	Templates      map[string]Template
}

type Mailer struct {
	config   Config
	from     string
	fromHead string
	host     string
}

var _ idpemailchallenge.Mailer = (*Mailer)(nil)

func SignupTemplates() map[string]Template {
	return map[string]Template{
		"signup-code": {
			Subject: "Verify your TinyIDP email address",
			Render: func(request idpemailchallenge.MailRequest) (string, error) {
				return fmt.Sprintf("Your verification code is: %s\r\n\r\nThis code expires at: %s\r\nIf you did not request this code, ignore this message.\r\n", request.Code, request.ExpiresAt.UTC().Format(time.RFC3339)), nil
			},
		},
	}
}

func New(config Config) (*Mailer, error) {
	host, _, err := net.SplitHostPort(strings.TrimSpace(config.Address))
	if err != nil || host == "" {
		return nil, errors.New("SMTP address must be a host:port pair")
	}
	if config.TLSMode != TLSStartTLS && config.TLSMode != TLSImplicit && config.TLSMode != TLSPrivatePlaintext {
		return nil, errors.New("SMTP TLS mode must be starttls, implicit, or private-plaintext")
	}
	if config.ConnectTimeout <= 0 || config.SendTimeout <= 0 {
		return nil, errors.New("SMTP connect and send timeouts must be positive")
	}
	if (strings.TrimSpace(config.Username) == "") != (len(config.Password) == 0) {
		return nil, errors.New("SMTP username and password must be configured together")
	}
	if config.TLSMode == TLSPrivatePlaintext && strings.TrimSpace(config.Username) != "" {
		return nil, errors.New("authenticated SMTP requires TLS")
	}
	from, err := singleMailbox(config.FromAddress)
	if err != nil {
		return nil, errors.Wrap(err, "invalid SMTP sender")
	}
	if len(config.Templates) == 0 {
		return nil, errors.New("at least one SMTP template is required")
	}
	for id, template := range config.Templates {
		if strings.TrimSpace(id) == "" || invalidHeader(template.Subject) || strings.TrimSpace(template.Subject) == "" || template.Render == nil {
			return nil, errors.New("SMTP template catalog is invalid")
		}
	}
	serverName := strings.TrimSpace(config.ServerName)
	if serverName == "" {
		serverName = host
	}
	config.ServerName = serverName
	config.Address = strings.TrimSpace(config.Address)
	config.Username = strings.TrimSpace(config.Username)
	config.FromAddress = from
	config.Password = append([]byte(nil), config.Password...)
	fromHead := from
	if name := strings.TrimSpace(config.FromName); name != "" {
		fromHead = (&mail.Address{Name: name, Address: from}).String()
	}
	return &Mailer{config: config, from: from, fromHead: fromHead, host: host}, nil
}

func (m *Mailer) SendEmailChallenge(ctx context.Context, request idpemailchallenge.MailRequest) error {
	if m == nil {
		return permanent(errors.New("SMTP mailer is unavailable"))
	}
	recipient, err := singleMailbox(request.Recipient)
	if err != nil {
		return permanent(errors.Wrap(err, "invalid email challenge recipient"))
	}
	template, ok := m.config.Templates[request.Template]
	if !ok {
		return permanent(errors.New("email challenge template is not allowed"))
	}
	body, err := template.Render(request)
	if err != nil {
		return permanent(errors.Wrap(err, "render email challenge template"))
	}
	message := renderMessage(time.Now().UTC(), m.fromHead, recipient, template.Subject, body)
	if err := m.send(ctx, recipient, strings.NewReader(message)); err != nil {
		return classify(err)
	}
	return nil
}

func (m *Mailer) send(ctx context.Context, recipient string, message io.Reader) error {
	dialer := net.Dialer{Timeout: m.config.ConnectTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", m.config.Address)
	if err != nil {
		return errors.Wrap(err, "connect SMTP server")
	}
	defer conn.Close()
	deadline := time.Now().Add(m.config.SendTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return errors.Wrap(err, "set SMTP deadline")
	}
	if m.config.TLSMode == TLSImplicit {
		tlsConn := tls.Client(conn, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: m.config.ServerName})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return errors.Wrap(err, "establish implicit SMTP TLS")
		}
		conn = tlsConn
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return errors.Wrap(err, "initialize SMTP session")
	}
	defer client.Close()
	if m.config.TLSMode == TLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not advertise STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: m.config.ServerName}); err != nil {
			return errors.Wrap(err, "establish SMTP STARTTLS")
		}
	}
	if m.config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.config.Username, string(m.config.Password), m.host)); err != nil {
			return errors.Wrap(err, "authenticate SMTP session")
		}
	}
	if err := client.Mail(m.from); err != nil {
		return errors.Wrap(err, "set SMTP sender")
	}
	if err := client.Rcpt(recipient); err != nil {
		return errors.Wrap(err, "set SMTP recipient")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.Wrap(err, "start SMTP message")
	}
	if _, err := io.Copy(writer, message); err != nil {
		_ = writer.Close()
		return errors.Wrap(err, "write SMTP message")
	}
	if err := writer.Close(); err != nil {
		return errors.Wrap(err, "finish SMTP message")
	}
	if err := client.Quit(); err != nil {
		return errors.Wrap(err, "finish SMTP session")
	}
	return nil
}

func renderMessage(date time.Time, from, recipient, subject, body string) string {
	var builder strings.Builder
	writer := bufio.NewWriter(&builder)
	_, _ = fmt.Fprintf(writer, "Date: %s\r\n", date.UTC().Format(time.RFC1123Z))
	_, _ = fmt.Fprintf(writer, "From: %s\r\n", from)
	_, _ = fmt.Fprintf(writer, "To: %s\r\n", recipient)
	_, _ = fmt.Fprintf(writer, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	_, _ = fmt.Fprint(writer, "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	_, _ = fmt.Fprint(writer, strings.ReplaceAll(body, "\n", "\r\n"))
	_ = writer.Flush()
	return strings.ReplaceAll(builder.String(), "\r\r\n", "\r\n")
}

func singleMailbox(value string) (string, error) {
	if invalidHeader(value) {
		return "", errors.New("mailbox contains a line break")
	}
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil || address.Address == "" || address.Name != "" && address.String() != strings.TrimSpace(value) {
		return "", errors.New("mailbox must contain exactly one address")
	}
	if invalidHeader(address.Address) {
		return "", errors.New("mailbox is invalid")
	}
	return address.Address, nil
}

func invalidHeader(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

type deliveryError struct {
	err   error
	retry idpemailchallenge.RetryClass
}

func (e deliveryError) Error() string                            { return e.err.Error() }
func (e deliveryError) Unwrap() error                            { return e.err }
func (e deliveryError) RetryClass() idpemailchallenge.RetryClass { return e.retry }
func permanent(err error) error {
	return deliveryError{err: err, retry: idpemailchallenge.RetryPermanent}
}
func transient(err error) error {
	return deliveryError{err: err, retry: idpemailchallenge.RetryTransient}
}

func classify(err error) error {
	var protocolError *textproto.Error
	if errors.As(err, &protocolError) {
		if protocolError.Code >= 400 && protocolError.Code < 500 {
			return transient(err)
		}
		return permanent(err)
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return transient(err)
	}
	return permanent(err)
}
