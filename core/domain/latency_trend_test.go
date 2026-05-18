package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTrendWindow(t *testing.T) {
	cases := map[string]TrendWindow{
		"7d":      TrendWindow7d,
		"30d":     TrendWindow30d,
		"90d":     TrendWindow90d,
		"":        TrendWindow7d,
		"garbage": TrendWindow7d,
		"1d":      TrendWindow7d,
	}
	for in, want := range cases {
		assert.Equal(t, want, ParseTrendWindow(in), "input=%q", in)
	}
}

func TestTrendWindow_BucketIntervalFor(t *testing.T) {
	assert.Equal(t, "1 hour", TrendWindow7d.BucketIntervalFor())
	assert.Equal(t, "6 hours", TrendWindow30d.BucketIntervalFor())
	assert.Equal(t, "1 day", TrendWindow90d.BucketIntervalFor())
}

func TestTrendWindow_Duration(t *testing.T) {
	assert.Equal(t, 7*24*time.Hour, TrendWindow7d.Duration())
	assert.Equal(t, 30*24*time.Hour, TrendWindow30d.Duration())
	assert.Equal(t, 90*24*time.Hour, TrendWindow90d.Duration())
}
