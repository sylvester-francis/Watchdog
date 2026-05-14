package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePasswordResetToken(t *testing.T) {
	userID := uuid.New()
	tok, plaintext, err := GeneratePasswordResetToken(userID, "1.2.3.4")
	assert.NoError(t, err)
	assert.NotNil(t, tok)
	assert.True(t, strings.HasPrefix(plaintext, "wd_pr_"), "plaintext must use wd_pr_ prefix")
	assert.Equal(t, userID, tok.UserID)
	assert.Equal(t, "1.2.3.4", tok.IPAddress)
	assert.NotEmpty(t, tok.TokenHash)
	assert.Equal(t, HashPasswordResetToken(plaintext), tok.TokenHash, "stored hash must match plaintext digest")
	assert.True(t, tok.ExpiresAt.After(time.Now()))
	assert.True(t, tok.ExpiresAt.Before(time.Now().Add(31*time.Minute)))
	assert.Nil(t, tok.UsedAt)
}

func TestGeneratePasswordResetToken_Unique(t *testing.T) {
	userID := uuid.New()
	_, p1, _ := GeneratePasswordResetToken(userID, "")
	_, p2, _ := GeneratePasswordResetToken(userID, "")
	assert.NotEqual(t, p1, p2)
}

func TestPasswordResetToken_IsValid(t *testing.T) {
	future := time.Now().Add(10 * time.Minute)
	past := time.Now().Add(-1 * time.Minute)
	used := time.Now().Add(-30 * time.Second)

	assert.True(t, (&PasswordResetToken{ExpiresAt: future}).IsValid())
	assert.False(t, (&PasswordResetToken{ExpiresAt: past}).IsValid(), "expired token must be invalid")
	assert.False(t, (&PasswordResetToken{ExpiresAt: future, UsedAt: &used}).IsValid(), "used token must be invalid")
}
