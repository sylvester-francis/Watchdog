package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Crypto   CryptoConfig
	Notify   NotifyConfig
	Feature  FeatureConfig
}

// FeatureConfig holds feature flags.
type FeatureConfig struct {
	DurableAlerts bool `envconfig:"WATCHDOG_DURABLE_ALERTS" default:"false"`
}

// NotifyConfig holds notification configuration.
// All fields are optional. Set the relevant config to activate a notifier.
type NotifyConfig struct {
	SlackWebhookURL   string `envconfig:"SLACK_WEBHOOK_URL"`
	DiscordWebhookURL string `envconfig:"DISCORD_WEBHOOK_URL"`
	WebhookURL        string `envconfig:"WEBHOOK_URL"`

	// Email (SMTP)
	SMTPHost     string `envconfig:"SMTP_HOST"`
	SMTPPort     int    `envconfig:"SMTP_PORT" default:"587"`
	SMTPUsername string `envconfig:"SMTP_USERNAME"`
	SMTPPassword string `envconfig:"SMTP_PASSWORD"`
	SMTPFrom     string `envconfig:"SMTP_FROM"`
	SMTPTo       string `envconfig:"SMTP_TO"`

	// Telegram
	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN"`
	TelegramChatID   string `envconfig:"TELEGRAM_CHAT_ID"`

	// PagerDuty
	PagerDutyRoutingKey string `envconfig:"PAGERDUTY_ROUTING_KEY"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host           string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port           int           `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout    time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"15s"`
	WriteTimeout   time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"15s"`
	IdleTimeout    time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"60s"`
	SecureCookies  bool          `envconfig:"SERVER_SECURE_COOKIES" default:"false"`
	AllowedOrigins []string      `envconfig:"ALLOWED_ORIGINS"`
}

// Address returns the server address in host:port format.
func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	URL             string        `envconfig:"DATABASE_URL" required:"true"`
	MaxConns        int32         `envconfig:"DATABASE_MAX_CONNS" default:"25"`
	MinConns        int32         `envconfig:"DATABASE_MIN_CONNS" default:"5"`
	MaxConnLifetime time.Duration `envconfig:"DATABASE_MAX_CONN_LIFETIME" default:"1h"`
	MaxConnIdleTime time.Duration `envconfig:"DATABASE_MAX_CONN_IDLE_TIME" default:"30m"`
}

// CryptoConfig holds encryption and security configuration.
type CryptoConfig struct {
	EncryptionKey string `envconfig:"ENCRYPTION_KEY" required:"true"`
	SessionSecret string `envconfig:"SESSION_SECRET" required:"true"`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// validate checks configuration constraints.
func (c *Config) validate() error {
	if len(c.Crypto.EncryptionKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes for AES-256")
	}

	if len(c.Crypto.SessionSecret) < 32 {
		return fmt.Errorf("SESSION_SECRET must be at least 32 bytes")
	}

	return nil
}
