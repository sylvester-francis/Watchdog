package notify_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
)

func TestSignWebhookPayload_KnownVector(t *testing.T) {
	secret := "test-secret"
	ts := time.Unix(1700000000, 0)
	nonce := "00000000-0000-4000-8000-000000000000"
	body := []byte(`{"event":"incident.opened"}`)

	sig := notify.SignWebhookPayload(secret, ts, nonce, body)

	// Recompute with:
	//   printf '%s' '1700000000.00000000-0000-4000-8000-000000000000.{"event":"incident.opened"}' \
	//     | openssl dgst -sha256 -hmac "test-secret" -hex
	assert.Equal(t, "1569aa24a0a34b8f2c8b42d97870930071cb0479eedaab0d0c281ba7732a52f3", sig)
}

func TestSignWebhookPayload_DifferentBody_DifferentSig(t *testing.T) {
	secret := "s"
	ts := time.Unix(1700000000, 0)
	nonce := "n"

	a := notify.SignWebhookPayload(secret, ts, nonce, []byte(`{"a":1}`))
	b := notify.SignWebhookPayload(secret, ts, nonce, []byte(`{"a":2}`))

	assert.NotEqual(t, a, b, "signature must change when body changes")
}

func TestSignWebhookPayload_DifferentSecret_DifferentSig(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	nonce := "n"
	body := []byte(`{"a":1}`)

	a := notify.SignWebhookPayload("secret1", ts, nonce, body)
	b := notify.SignWebhookPayload("secret2", ts, nonce, body)

	assert.NotEqual(t, a, b, "signature must change when secret changes")
}

func TestSignWebhookPayload_DifferentNonce_DifferentSig(t *testing.T) {
	secret := "s"
	ts := time.Unix(1700000000, 0)
	body := []byte(`{"a":1}`)

	a := notify.SignWebhookPayload(secret, ts, "nonce-a", body)
	b := notify.SignWebhookPayload(secret, ts, "nonce-b", body)

	assert.NotEqual(t, a, b, "signature must change when nonce changes")
}

func TestSignWebhookPayload_DifferentTimestamp_DifferentSig(t *testing.T) {
	secret := "s"
	nonce := "n"
	body := []byte(`{"a":1}`)

	a := notify.SignWebhookPayload(secret, time.Unix(1700000000, 0), nonce, body)
	b := notify.SignWebhookPayload(secret, time.Unix(1700000001, 0), nonce, body)

	assert.NotEqual(t, a, b, "signature must change when timestamp changes")
}

func TestSignWebhookPayload_HexOutput(t *testing.T) {
	sig := notify.SignWebhookPayload("s", time.Unix(1700000000, 0), "n", []byte("b"))
	assert.Len(t, sig, 64, "sha256 hex is 64 chars")
	for _, c := range sig {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'), "lowercase hex only: %q", sig)
	}
}

func TestGenerateNonce_UUIDv4Format(t *testing.T) {
	n := notify.GenerateNonce()
	// UUIDv4: 8-4-4-4-12 hex chars, version 4, variant 10xx
	require.Len(t, n, 36)
	parts := strings.Split(n, "-")
	require.Len(t, parts, 5)
	assert.Len(t, parts[0], 8)
	assert.Len(t, parts[1], 4)
	assert.Len(t, parts[2], 4)
	assert.Len(t, parts[3], 4)
	assert.Len(t, parts[4], 12)
	assert.Equal(t, byte('4'), parts[2][0], "version 4")
}

func TestGenerateNonce_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		n := notify.GenerateNonce()
		assert.False(t, seen[n], "duplicate nonce: %s", n)
		seen[n] = true
	}
}
