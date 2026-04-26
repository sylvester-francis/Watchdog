package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_SetRateLimitOnReject_StoresCallback(t *testing.T) {
	// Bare Router struct is sufficient for this test — we're only verifying
	// the setter wiring, not the limiter construction (which happens in
	// RegisterRoutes and is covered by the existing rate-limit unit tests).
	r := &Router{}

	var called bool
	var capturedIP string
	r.SetRateLimitOnReject(func(ip string) {
		called = true
		capturedIP = ip
	})

	require.NotNil(t, r.rateLimitOnReject, "setter must store the callback")
	r.rateLimitOnReject("203.0.113.7")
	assert.True(t, called)
	assert.Equal(t, "203.0.113.7", capturedIP)
}

func TestRouter_SetRateLimitOnReject_NilSetterAccepted(t *testing.T) {
	r := &Router{}
	r.SetRateLimitOnReject(nil)
	assert.Nil(t, r.rateLimitOnReject, "nil callback explicitly clears the field")
}
