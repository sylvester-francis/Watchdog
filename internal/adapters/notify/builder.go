package notify

import (
	"fmt"
	"strconv"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// ChannelNotifierFactory implements ports.NotifierFactory by building
// notifiers from AlertChannel configurations.
type ChannelNotifierFactory struct{}

// NewChannelNotifierFactory creates a new ChannelNotifierFactory.
func NewChannelNotifierFactory() *ChannelNotifierFactory {
	return &ChannelNotifierFactory{}
}

// BuildFromChannel creates a Notifier from an AlertChannel's type and config.
func (f *ChannelNotifierFactory) BuildFromChannel(channel *domain.AlertChannel) (ports.Notifier, error) {
	return BuildFromChannel(channel)
}

// BuildFromChannel creates a Notifier from an AlertChannel's type and config.
func BuildFromChannel(channel *domain.AlertChannel) (Notifier, error) {
	switch channel.Type {
	case domain.AlertChannelDiscord:
		url := channel.Config["webhook_url"]
		if url == "" {
			return nil, fmt.Errorf("discord: webhook_url is required")
		}
		return NewDiscordNotifier(url), nil

	case domain.AlertChannelSlack:
		url := channel.Config["webhook_url"]
		if url == "" {
			return nil, fmt.Errorf("slack: webhook_url is required")
		}
		return NewSlackNotifier(url), nil

	case domain.AlertChannelWebhook:
		url := channel.Config["url"]
		if url == "" {
			return nil, fmt.Errorf("webhook: url is required")
		}
		return NewWebhookNotifier(url), nil

	case domain.AlertChannelEmail:
		port := 587
		if p := channel.Config["port"]; p != "" {
			parsed, err := strconv.Atoi(p)
			if err == nil {
				port = parsed
			}
		}
		return NewEmailNotifier(EmailConfig{
			Host:     channel.Config["host"],
			Port:     port,
			Username: channel.Config["username"],
			Password: channel.Config["password"],
			From:     channel.Config["from"],
			To:       channel.Config["to"],
		}), nil

	case domain.AlertChannelTelegram:
		token := channel.Config["bot_token"]
		chatID := channel.Config["chat_id"]
		if token == "" || chatID == "" {
			return nil, fmt.Errorf("telegram: bot_token and chat_id are required")
		}
		return NewTelegramNotifier(token, chatID), nil

	case domain.AlertChannelPagerDuty:
		key := channel.Config["routing_key"]
		if key == "" {
			return nil, fmt.Errorf("pagerduty: routing_key is required")
		}
		return NewPagerDutyNotifier(key), nil

	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}
