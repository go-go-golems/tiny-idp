package smtpmailer_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge/smtpmailer"
)

func TestPrivateSMTPDeliversFixedSignupTemplate(t *testing.T) {
	server := newSMTPServer(t, 250)
	mailer, err := smtpmailer.New(smtpmailer.Config{
		Address: server.address(), TLSMode: smtpmailer.TLSPrivatePlaintext,
		FromAddress: "accounts@example.test", FromName: "TinyIDP",
		ConnectTimeout: time.Second, SendTimeout: time.Second,
		Templates: smtpmailer.SignupTemplates(),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := idpemailchallenge.MailRequest{Recipient: "user@example.test", Template: "signup-code", Code: "ABC23456", ExpiresAt: time.Date(2026, 7, 21, 21, 0, 0, 0, time.UTC)}
	beforeSend := time.Now().UTC().Add(-time.Second)
	if err := mailer.SendEmailChallenge(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	afterSend := time.Now().UTC().Add(time.Second)
	message := server.message(t)
	for _, expected := range []string{"From: \"TinyIDP\" <accounts@example.test>", "To: user@example.test", "ABC23456", "2026-07-21T21:00:00Z"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("message does not contain %q:\n%s", expected, message)
		}
	}
	parsed, err := mail.ReadMessage(strings.NewReader(message))
	if err != nil {
		t.Fatalf("parse SMTP message: %v", err)
	}
	messageDate, err := mail.ParseDate(parsed.Header.Get("Date"))
	if err != nil {
		t.Fatalf("parse Date header: %v", err)
	}
	if messageDate.Before(beforeSend) || messageDate.After(afterSend) {
		t.Fatalf("Date header %s is outside send window [%s, %s]", messageDate, beforeSend, afterSend)
	}
}

func TestMailerRejectsUnsafeOrIncompleteConfiguration(t *testing.T) {
	valid := smtpmailer.Config{Address: "smtp.example.test:587", TLSMode: smtpmailer.TLSStartTLS, FromAddress: "accounts@example.test", ConnectTimeout: time.Second, SendTimeout: time.Second, Templates: smtpmailer.SignupTemplates()}
	tests := []struct {
		name   string
		mutate func(*smtpmailer.Config)
	}{
		{name: "invalid address", mutate: func(c *smtpmailer.Config) { c.Address = "smtp.example.test" }},
		{name: "unknown TLS", mutate: func(c *smtpmailer.Config) { c.TLSMode = "auto" }},
		{name: "missing timeout", mutate: func(c *smtpmailer.Config) { c.SendTimeout = 0 }},
		{name: "password without user", mutate: func(c *smtpmailer.Config) { c.Password = []byte("secret") }},
		{name: "plaintext authentication", mutate: func(c *smtpmailer.Config) {
			c.TLSMode = smtpmailer.TLSPrivatePlaintext
			c.Username = "u"
			c.Password = []byte("secret")
		}},
		{name: "sender injection", mutate: func(c *smtpmailer.Config) { c.FromAddress = "a@example.test\r\nBcc: x@example.test" }},
		{name: "no templates", mutate: func(c *smtpmailer.Config) { c.Templates = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			if _, err := smtpmailer.New(config); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestMailerRejectsRecipientAndTemplateWithoutLeakingCode(t *testing.T) {
	server := newSMTPServer(t, 250)
	mailer, err := smtpmailer.New(smtpmailer.Config{Address: server.address(), TLSMode: smtpmailer.TLSPrivatePlaintext, FromAddress: "accounts@example.test", ConnectTimeout: time.Second, SendTimeout: time.Second, Templates: smtpmailer.SignupTemplates()})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []idpemailchallenge.MailRequest{
		{Recipient: "user@example.test\r\nBcc: thief@example.test", Template: "signup-code", Code: "SECRET23"},
		{Recipient: "user@example.test", Template: "arbitrary", Code: "SECRET23"},
	} {
		err := mailer.SendEmailChallenge(context.Background(), request)
		if err == nil || strings.Contains(err.Error(), request.Code) {
			t.Fatalf("unsafe request error = %v", err)
		}
		var failure idpemailchallenge.MailFailure
		if !errors.As(err, &failure) || failure.RetryClass() != idpemailchallenge.RetryPermanent {
			t.Fatalf("failure class = %v", err)
		}
	}
}

func TestSMTPStatusClassifiesRetry(t *testing.T) {
	server := newSMTPServer(t, 451)
	mailer, err := smtpmailer.New(smtpmailer.Config{Address: server.address(), TLSMode: smtpmailer.TLSPrivatePlaintext, FromAddress: "accounts@example.test", ConnectTimeout: time.Second, SendTimeout: time.Second, Templates: smtpmailer.SignupTemplates()})
	if err != nil {
		t.Fatal(err)
	}
	err = mailer.SendEmailChallenge(context.Background(), idpemailchallenge.MailRequest{Recipient: "user@example.test", Template: "signup-code", Code: "ABC23456", ExpiresAt: time.Now().Add(time.Minute)})
	var failure idpemailchallenge.MailFailure
	if !errors.As(err, &failure) || failure.RetryClass() != idpemailchallenge.RetryTransient {
		t.Fatalf("error = %v, want transient MailFailure", err)
	}
}

type smtpServer struct {
	listener net.Listener
	messages chan string
	dataCode int
}

func newSMTPServer(t *testing.T, dataCode int) *smtpServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &smtpServer{listener: listener, messages: make(chan string, 1), dataCode: dataCode}
	t.Cleanup(func() { _ = listener.Close() })
	go server.serve()
	return server
}

func (s *smtpServer) address() string { return s.listener.Addr().String() }

func (s *smtpServer) message(t *testing.T) string {
	t.Helper()
	select {
	case message := <-s.messages:
		return message
	case <-time.After(time.Second):
		t.Fatal("SMTP message was not received")
		return ""
	}
}

func (s *smtpServer) serve() {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writeResponse(writer, 220, "test SMTP")
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"):
			_, _ = fmt.Fprint(writer, "250-test\r\n250 HELP\r\n")
			_ = writer.Flush()
		case strings.HasPrefix(command, "MAIL FROM"), strings.HasPrefix(command, "RCPT TO"):
			writeResponse(writer, 250, "ok")
		case command == "DATA":
			writeResponse(writer, 354, "continue")
			var message strings.Builder
			for {
				dataLine, dataErr := reader.ReadString('\n')
				if dataErr != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				message.WriteString(dataLine)
			}
			if s.dataCode == 250 {
				s.messages <- message.String()
				writeResponse(writer, 250, "queued")
			} else {
				writeResponse(writer, s.dataCode, "temporary delivery failure")
			}
		case command == "QUIT":
			writeResponse(writer, 221, "bye")
			return
		default:
			writeResponse(writer, 250, "ok")
		}
	}
}

func writeResponse(writer *bufio.Writer, code int, text string) {
	_, _ = fmt.Fprintf(writer, "%d %s\r\n", code, text)
	_ = writer.Flush()
}
