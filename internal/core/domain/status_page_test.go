package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{
			name:  "simple name with spaces",
			input: "My Status Page",
			want:  "my-status-page",
		},
		{
			name:  "already lowercase",
			input: "hello world",
			want:  "hello-world",
		},
		{
			name:  "single word",
			input: "dashboard",
			want:  "dashboard",
		},
		{
			name:  "mixed case",
			input: "My Service",
			want:  "my-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateSlug_SpecialChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trailing exclamation marks",
			input: "Hello World!!!",
			want:  "hello-world",
		},
		{
			name:  "special characters throughout",
			input: "My @Service #1!",
			want:  "my-service-1",
		},
		{
			name:  "underscores and dots",
			input: "my_service.page",
			want:  "my-service-page",
		},
		{
			name:  "leading and trailing special chars",
			input: "---hello---",
			want:  "hello",
		},
		{
			name:  "multiple consecutive special chars",
			input: "hello***world",
			want:  "hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateSlug_Empty(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "status",
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  "status",
		},
		{
			name:  "only special characters",
			input: "!!!@@@###",
			want:  "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewStatusPage(t *testing.T) {
	userID := uuid.New()
	name := "Production Status"
	slug := "production-status"

	page := NewStatusPage(userID, name, slug)

	require.NotNil(t, page)
	assert.NotEqual(t, uuid.Nil, page.ID, "ID should be a generated UUID")
	assert.Equal(t, userID, page.UserID)
	assert.Equal(t, name, page.Name)
	assert.Equal(t, slug, page.Slug)
	assert.True(t, page.IsPublic, "new status pages should default to public")
	assert.Empty(t, page.MonitorIDs, "monitor IDs should be empty initially")
	assert.Empty(t, page.Description, "description should be empty initially")
	assert.False(t, page.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, page.UpdatedAt.IsZero(), "UpdatedAt should be set")
}
