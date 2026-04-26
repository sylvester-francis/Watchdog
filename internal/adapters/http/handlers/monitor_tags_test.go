package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTagsQuery(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want map[string]string
	}{
		{"empty input", nil, map[string]string{}},
		{"single tag", []string{"env:prod"}, map[string]string{"env": "prod"}},
		{"multiple tags", []string{"env:prod", "tier:web"}, map[string]string{"env": "prod", "tier": "web"}},
		{"value with colon kept intact", []string{"endpoint:host:443"}, map[string]string{"endpoint": "host:443"}},
		{"missing colon dropped", []string{"malformed"}, map[string]string{}},
		{"empty key dropped", []string{":value"}, map[string]string{}},
		{"empty value dropped", []string{"key:"}, map[string]string{}},
		{"mixed valid + malformed keeps valid", []string{"env:prod", "broken", ":x", "y:"}, map[string]string{"env": "prod"}},
		{"duplicate key — last wins (map semantics)", []string{"env:prod", "env:staging"}, map[string]string{"env": "staging"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseTagsQuery(tc.in))
		})
	}
}
