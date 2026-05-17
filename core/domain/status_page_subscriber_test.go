package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateStatusPageSubscriber(t *testing.T) {
	pageID := uuid.New()
	sub, plaintext, err := GenerateStatusPageSubscriber(pageID, "alice@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.True(t, strings.HasPrefix(plaintext, "wd_sub_"), "plaintext must use wd_sub_ prefix")
	assert.Equal(t, pageID, sub.StatusPageID)
	assert.Equal(t, "alice@example.com", sub.Email)
	assert.NotEmpty(t, sub.TokenHash)
	assert.Equal(t, HashStatusPageSubscriberToken(plaintext), sub.TokenHash)
	assert.Nil(t, sub.ConfirmedAt)
	assert.Nil(t, sub.UnsubscribedAt)
}

func TestGenerateStatusPageSubscriber_Unique(t *testing.T) {
	pageID := uuid.New()
	_, p1, _ := GenerateStatusPageSubscriber(pageID, "a@b.co")
	_, p2, _ := GenerateStatusPageSubscriber(pageID, "a@b.co")
	assert.NotEqual(t, p1, p2, "successive tokens must differ")
}

func TestStatusPageSubscriber_IsActive(t *testing.T) {
	now := time.Now()
	confirmed := &StatusPageSubscriber{ConfirmedAt: &now}
	unconfirmed := &StatusPageSubscriber{}
	unsub := &StatusPageSubscriber{ConfirmedAt: &now, UnsubscribedAt: &now}

	assert.True(t, confirmed.IsActive())
	assert.False(t, unconfirmed.IsActive(), "must be confirmed")
	assert.False(t, unsub.IsActive(), "unsubscribed = not active")
}
