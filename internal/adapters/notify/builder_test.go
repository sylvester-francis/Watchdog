package notify_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
)

func TestBuildFromChannel_Webhook_WithSigningSecret(t *testing.T) {
	ch := &domain.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   domain.AlertChannelWebhook,
		Name:   "Signed webhook",
		Config: map[string]string{
			"url":            "https://example.com/hook",
			"signing_secret": "top-secret",
		},
	}

	n, err := notify.BuildFromChannel(ch)
	require.NoError(t, err)
	require.NotNil(t, n)

	_, isWebhook := n.(*notify.WebhookNotifier)
	assert.True(t, isWebhook)
}

func TestBuildFromChannel_Webhook_WithoutSigningSecret(t *testing.T) {
	ch := &domain.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   domain.AlertChannelWebhook,
		Name:   "Unsigned webhook",
		Config: map[string]string{
			"url": "https://example.com/hook",
		},
	}

	n, err := notify.BuildFromChannel(ch)
	require.NoError(t, err)
	require.NotNil(t, n)
}

func TestBuildFromChannel_Webhook_MissingURL(t *testing.T) {
	ch := &domain.AlertChannel{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   domain.AlertChannelWebhook,
		Name:   "Bad webhook",
		Config: map[string]string{
			"signing_secret": "s",
		},
	}

	_, err := notify.BuildFromChannel(ch)
	assert.Error(t, err, "missing url should fail")
}
