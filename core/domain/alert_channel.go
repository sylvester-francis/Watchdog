package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AlertChannelType represents the type of alert channel.
type AlertChannelType string

const (
	AlertChannelDiscord   AlertChannelType = "discord"
	AlertChannelSlack     AlertChannelType = "slack"
	AlertChannelEmail     AlertChannelType = "email"
	AlertChannelTelegram  AlertChannelType = "telegram"
	AlertChannelPagerDuty AlertChannelType = "pagerduty"
	AlertChannelWebhook   AlertChannelType = "webhook"
)

// ValidAlertChannelTypes is the set of supported alert channel types.
var ValidAlertChannelTypes = map[AlertChannelType]bool{
	AlertChannelDiscord:   true,
	AlertChannelSlack:     true,
	AlertChannelEmail:     true,
	AlertChannelTelegram:  true,
	AlertChannelPagerDuty: true,
	AlertChannelWebhook:   true,
}

// AlertChannel represents a user-configured notification channel.
type AlertChannel struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      AlertChannelType
	Name      string
	Config    map[string]string // decrypted config, never persisted in plaintext
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewAlertChannel creates a new AlertChannel with defaults.
func NewAlertChannel(userID uuid.UUID, channelType AlertChannelType, name string, config map[string]string) *AlertChannel {
	return &AlertChannel{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      channelType,
		Name:      name,
		Config:    config,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate checks that required config fields are present for the channel type.
func (ac *AlertChannel) Validate() error {
	if !ValidAlertChannelTypes[ac.Type] {
		return fmt.Errorf("invalid channel type: %s", ac.Type)
	}
	if ac.Name == "" {
		return fmt.Errorf("channel name is required")
	}

	switch ac.Type {
	case AlertChannelDiscord, AlertChannelSlack:
		if ac.Config["webhook_url"] == "" {
			return fmt.Errorf("webhook_url is required for %s", ac.Type)
		}
	case AlertChannelWebhook:
		if ac.Config["url"] == "" {
			return fmt.Errorf("url is required for webhook")
		}
	case AlertChannelEmail:
		if ac.Config["host"] == "" || ac.Config["from"] == "" || ac.Config["to"] == "" {
			return fmt.Errorf("host, from, and to are required for email")
		}
	case AlertChannelTelegram:
		if ac.Config["bot_token"] == "" || ac.Config["chat_id"] == "" {
			return fmt.Errorf("bot_token and chat_id are required for telegram")
		}
	case AlertChannelPagerDuty:
		if ac.Config["routing_key"] == "" {
			return fmt.Errorf("routing_key is required for pagerduty")
		}
	}

	return nil
}
