// Package email provides a hub-wide transactional email sender, distinct from
// per-channel alert email which lives in internal/adapters/notify/email.go.
//
// Use TransactionalSender for one-shot system messages addressed to a single
// recipient computed at runtime (password reset links, signup verification,
// etc.). It reuses the existing SMTP_HOST/USERNAME/PASSWORD/FROM env vars but
// ignores SMTP_TO (which is the alert-channel static recipient).
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
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

// Send delivers a plain-text message to a single recipient. Drives the SMTP
// conversation manually (rather than calling smtp.SendMail) so we can pass a
// custom *tls.Config to StartTLS — needed when the relay's cert hostname
// doesn't match how we address it (e.g. dialing by IP).
func (s *TransactionalSender) Send(_ context.Context, to, subject, body string) error {
	if s.cfg.Host == "" || s.cfg.From == "" {
		return fmt.Errorf("transactional email not configured (SMTP_HOST/SMTP_FROM missing)")
	}
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.From, to, subject, body,
	)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}

	// Upgrade to TLS if the server advertises STARTTLS (most modern relays do).
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

// Configured reports whether the sender has the minimum SMTP_HOST + SMTP_FROM
// to actually deliver mail. Callers use this to gate features at startup.
func (s *TransactionalSender) Configured() bool {
	return s.cfg.Host != "" && s.cfg.From != ""
}
