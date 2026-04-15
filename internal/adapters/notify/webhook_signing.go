package notify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// SignWebhookPayload computes HMAC-SHA256 over "{timestamp}.{nonce}.{body}"
// and returns the lowercase hex-encoded signature. The signed string format
// is stable and documented in docs/webhooks.md — receivers must construct
// the same string to verify.
//
// The timestamp is serialized as unix seconds. The nonce is an opaque
// string — GenerateNonce produces a UUIDv4, but any unique identifier
// provided by the caller works.
func SignWebhookPayload(secret string, ts time.Time, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write([]byte(nonce))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateNonce returns a random UUIDv4 string suitable for use as a
// webhook nonce. Each outbound webhook should use a fresh nonce so
// receivers can build a replay-protection cache.
func GenerateNonce() string {
	return uuid.NewString()
}
