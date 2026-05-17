// Package email provides a hub-wide transactional email sender, distinct from
// per-channel alert email which lives in internal/adapters/notify/email.go.
//
// Use TransactionalSender for one-shot system messages addressed to a single
// recipient computed at runtime (password reset links, signup verification,
// SLA reports, etc.). It reuses the existing SMTP_HOST/USERNAME/PASSWORD/FROM
// env vars but ignores SMTP_TO (which is the alert-channel static recipient).
package email

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"strings"
)

// Config holds SMTP relay settings.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// InsecureSkipVerify disables TLS certificate verification on the
	// STARTTLS upgrade. Use this when relaying to a trusted local Postfix
	// reached by IP whose cert is for a different hostname (matches the
	// `?skip_ssl_verify=true` flag the EE Kratos courier uses).
	InsecureSkipVerify bool
}

// TransactionalSender sends plain-text emails to dynamic recipients via SMTP.
type TransactionalSender struct {
	cfg  Config
	auth smtp.Auth
}

// NewTransactionalSender constructs a sender. If Username is empty, no SMTP AUTH
// is performed (matches our hostinger Postfix relay which accepts by IP allowlist).
func NewTransactionalSender(cfg Config) *TransactionalSender {
	s := &TransactionalSender{cfg: cfg}
	if cfg.Username != "" {
		s.auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	return s
}

// Send delivers a plain-text message to a single recipient.
func (s *TransactionalSender) Send(_ context.Context, to, subject, body string) error {
	if s.cfg.Host == "" || s.cfg.From == "" {
		return fmt.Errorf("transactional email not configured (SMTP_HOST/SMTP_FROM missing)")
	}
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.From, to, subject, body,
	)
	return s.deliver(to, msg)
}

// Configured reports whether the sender has the minimum SMTP_HOST + SMTP_FROM
// to actually deliver mail. Callers use this to gate features at startup.
func (s *TransactionalSender) Configured() bool {
	return s.cfg.Host != "" && s.cfg.From != ""
}

// Attachment is a single file attached to an email — typically a small PDF
// generated at request time (SLA reports). Large payloads should be sent as
// a download link instead; SMTP relays vary in size limits.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// SendAttachment delivers a plain-text body plus one attachment via SMTP
// multipart/mixed.
func (s *TransactionalSender) SendAttachment(_ context.Context, to, subject, body string, att Attachment) error {
	if s.cfg.Host == "" || s.cfg.From == "" {
		return fmt.Errorf("transactional email not configured (SMTP_HOST/SMTP_FROM missing)")
	}
	boundaryBytes := make([]byte, 16)
	if _, err := rand.Read(boundaryBytes); err != nil {
		return fmt.Errorf("boundary rand: %w", err)
	}
	boundary := "wd_" + hex.EncodeToString(boundaryBytes)

	var msg strings.Builder
	fmt.Fprintf(&msg, "From: %s\r\n", s.cfg.From)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary)

	fmt.Fprintf(&msg, "--%s\r\n", boundary)
	fmt.Fprintf(&msg, "Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	msg.WriteString(body)
	msg.WriteString("\r\n")

	fmt.Fprintf(&msg, "--%s\r\n", boundary)
	fmt.Fprintf(&msg, "Content-Type: %s\r\n", att.ContentType)
	fmt.Fprintf(&msg, "Content-Transfer-Encoding: base64\r\n")
	fmt.Fprintf(&msg, "Content-Disposition: attachment; filename=%q\r\n\r\n", att.Filename)
	encoded := base64.StdEncoding.EncodeToString(att.Data)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		msg.WriteString(encoded[i:end])
		msg.WriteString("\r\n")
	}
	fmt.Fprintf(&msg, "--%s--\r\n", boundary)

	return s.deliver(to, msg.String())
}

// deliver drives the SMTP conversation manually (rather than smtp.SendMail) so
// we can pass a custom *tls.Config to StartTLS — needed when the relay's cert
// hostname doesn't match how we address it (e.g. dialing by IP).
func (s *TransactionalSender) deliver(to, msg string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{
			ServerName:         s.cfg.Host,
			InsecureSkipVerify: s.cfg.InsecureSkipVerify, //nolint:gosec // intentional for local-network relays addressed by IP
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if s.auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(s.auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := c.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return c.Quit()
}
