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
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	if err := smtp.SendMail(addr, s.auth, s.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send transactional mail: %w", err)
	}
	return nil
}

// Configured reports whether the sender has the minimum SMTP_HOST + SMTP_FROM
// to actually deliver mail. Callers use this to gate features at startup.
func (s *TransactionalSender) Configured() bool {
	return s.cfg.Host != "" && s.cfg.From != ""
}
