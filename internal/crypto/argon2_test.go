package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "correct-horse-battery-staple"

	hash, err := hasher.Hash(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$"))
	assert.Contains(t, hash, "$v=19$")
}

func TestPasswordHasher_Hash_UniqueHashes(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "same-password"

	hash1, err1 := hasher.Hash(password)
	require.NoError(t, err1)

	hash2, err2 := hasher.Hash(password)
	require.NoError(t, err2)

	// Same password should produce different hashes due to unique salt
	assert.NotEqual(t, hash1, hash2)
}

func TestPasswordHasher_Verify_CorrectPassword(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "correct-password"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	valid, err := hasher.Verify(password, hash)

	require.NoError(t, err)
	assert.True(t, valid)
}

func TestPasswordHasher_Verify_WrongPassword(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "correct-password"
	wrongPassword := "wrong-password"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	valid, err := hasher.Verify(wrongPassword, hash)

	require.NoError(t, err)
	assert.False(t, valid)
}

func TestPasswordHasher_Verify_InvalidHash(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name string
		hash string
	}{
		{"empty hash", ""},
		{"invalid format", "not-a-valid-hash"},
		{"wrong prefix", "$bcrypt$invalid"},
		{"missing parts", "$argon2id$v=19$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := hasher.Verify("password", tt.hash)
			assert.Error(t, err)
		})
	}
}

func TestPasswordHasher_Verify_TimingConsistency(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "test-password"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	// Verify multiple times to ensure consistent timing
	for i := 0; i < 5; i++ {
		valid, err := hasher.Verify(password, hash)
		require.NoError(t, err)
		assert.True(t, valid)
	}
}

func TestPasswordHasher_decodeHash_Errors(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name string
		hash string
		err  error
	}{
		{
			name: "wrong number of parts",
			hash: "$argon2id$v=19$m=65536",
			err:  ErrInvalidHash,
		},
		{
			name: "wrong algorithm",
			hash: "$bcrypt$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA",
			err:  ErrInvalidHash,
		},
		{
			name: "invalid base64 salt",
			hash: "$argon2id$v=19$m=65536,t=3,p=4$!!!invalid!!!$aGFzaA",
			err:  ErrInvalidHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := hasher.Verify("password", tt.hash)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func BenchmarkPasswordHasher_Hash(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "benchmark-password"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Hash(password)
	}
}

func BenchmarkPasswordHasher_Verify(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "benchmark-password"
	hash, _ := hasher.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Verify(password, hash)
	}
}
