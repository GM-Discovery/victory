// Package mailer provides a small delivery abstraction for Kernel 77
// self-service recovery email, so auth code depends on an interface rather
// than one hard-coded vendor (Kernel 77 §8.1).
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// RecoveryMailer is the delivery interface identity code depends on.
type RecoveryMailer interface {
	SendPasswordReset(ctx context.Context, recipient, resetURL string) error
	SendEmailVerification(ctx context.Context, recipient, verifyURL string) error
}

// Config is read from environment in cmd/victory/main.go and never logged.
type Config struct {
	Enabled  bool
	From     string
	FromName string
	Host     string
	Port     string
	Username string
	Password string
	TLSMode  string // "starttls" (default) or "tls"
}

// SMTPMailer sends through a standard SMTP relay (Brevo, or any other
// transactional provider exposing SMTP). It is intentionally small: one
// relay, no templates engine, no queue.
type SMTPMailer struct {
	cfg Config
}

func NewSMTPMailer(cfg Config) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) SendPasswordReset(ctx context.Context, recipient, resetURL string) error {
	subject := "Reset your Victory password"
	body := "" +
		"Someone (hopefully you) asked to reset the password for this Victory account.\n\n" +
		"Reset it here — this link works once and expires in 1 hour:\n" +
		resetURL + "\n\n" +
		"If you didn't request this, you can ignore this email; your password will not change.\n"
	return m.send(ctx, recipient, subject, body)
}

func (m *SMTPMailer) SendEmailVerification(ctx context.Context, recipient, verifyURL string) error {
	subject := "Verify your Victory recovery email"
	body := "" +
		"Confirm this email address so it can be used for Victory account recovery.\n\n" +
		"Verify it here — this link works once and expires in 24 hours:\n" +
		verifyURL + "\n\n" +
		"If you didn't request this, you can ignore this email.\n"
	return m.send(ctx, recipient, subject, body)
}

func (m *SMTPMailer) send(ctx context.Context, recipient, subject, body string) error {
	if !m.cfg.Enabled {
		return fmt.Errorf("recovery email is not enabled")
	}
	if strings.TrimSpace(m.cfg.Host) == "" {
		return fmt.Errorf("recovery email is enabled but SMTP_HOST is not configured")
	}

	from := m.cfg.From
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.From)
	}

	msg := "From: " + from + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"\r\n" + body

	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)

	done := make(chan error, 1)
	go func() {
		if strings.EqualFold(m.cfg.TLSMode, "tls") {
			done <- m.sendImplicitTLS(addr, auth, recipient, msg)
			return
		}
		done <- smtp.SendMail(addr, auth, m.cfg.From, []string{recipient}, []byte(msg))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (m *SMTPMailer) sendImplicitTLS(addr string, auth smtp.Auth, recipient, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// NoopMailer is used when recovery email is not configured. Every send
// fails loudly rather than silently, so misconfiguration in production is
// visible (Kernel 77 §8.2 "fail closed... when recovery is advertised but
// mail is not configured").
type NoopMailer struct{}

func (NoopMailer) SendPasswordReset(ctx context.Context, recipient, resetURL string) error {
	return fmt.Errorf("recovery email is not configured")
}

func (NoopMailer) SendEmailVerification(ctx context.Context, recipient, verifyURL string) error {
	return fmt.Errorf("recovery email is not configured")
}
