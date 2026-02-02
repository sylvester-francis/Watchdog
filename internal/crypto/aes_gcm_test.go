package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptor(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		key := "12345678901234567890123456789012" // 32 bytes

		enc, err := NewEncryptor(key)

		require.NoError(t, err)
		assert.NotNil(t, enc)
	})

	t.Run("invalid key length", func(t *testing.T) {
		tests := []struct {
			name string
			key  string
		}{
			{"too short", "short"},
			{"16 bytes", "1234567890123456"},
			{"33 bytes", "123456789012345678901234567890123"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				enc, err := NewEncryptor(tt.key)

				assert.ErrorIs(t, err, ErrInvalidKeySize)
				assert.Nil(t, enc)
			})
		}
	})
}

func TestEncryptor_Encrypt(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("Hello, World!")

	ciphertext, err := enc.Encrypt(plaintext)

	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	// Ciphertext should be longer than plaintext (nonce + tag)
	assert.Greater(t, len(ciphertext), len(plaintext))
	// Ciphertext should not contain plaintext
	assert.False(t, bytes.Contains(ciphertext, plaintext))
}

func TestEncryptor_Encrypt_UniqueNonce(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("Same plaintext")

	ciphertext1, err1 := enc.Encrypt(plaintext)
	require.NoError(t, err1)

	ciphertext2, err2 := enc.Encrypt(plaintext)
	require.NoError(t, err2)

	// Same plaintext should produce different ciphertext due to unique nonce
	assert.NotEqual(t, ciphertext1, ciphertext2)
}

func TestEncryptor_Decrypt(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	original := []byte("Secret message")

	ciphertext, err := enc.Encrypt(original)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)

	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestEncryptor_Decrypt_TooShort(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	shortCiphertext := []byte("short") // Less than nonce size

	_, err = enc.Decrypt(shortCiphertext)

	assert.ErrorIs(t, err, ErrCiphertextTooShort)
}

func TestEncryptor_Decrypt_TamperedCiphertext(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("Secret message")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	// Tamper with ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = enc.Decrypt(ciphertext)

	assert.Error(t, err)
}

func TestEncryptor_Decrypt_WrongKey(t *testing.T) {
	key1 := "12345678901234567890123456789012"
	key2 := "abcdefghijklmnopqrstuvwxyz123456"

	enc1, _ := NewEncryptor(key1)
	enc2, _ := NewEncryptor(key2)

	plaintext := []byte("Secret message")
	ciphertext, _ := enc1.Encrypt(plaintext)

	_, err := enc2.Decrypt(ciphertext)

	assert.Error(t, err)
}

func TestEncryptor_EncryptString(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := "Hello, World!"

	ciphertext, err := enc.EncryptString(plaintext)

	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
}

func TestEncryptor_DecryptString(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	original := "Secret message"

	ciphertext, err := enc.EncryptString(original)
	require.NoError(t, err)

	decrypted, err := enc.DecryptString(ciphertext)

	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestEncryptor_EmptyPlaintext(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte{}

	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)

	// Decrypting empty plaintext may return nil or empty slice - both are valid
	assert.Empty(t, decrypted)
}

func BenchmarkEncryptor_Encrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	enc, _ := NewEncryptor(key)
	plaintext := []byte("Benchmark plaintext for encryption")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encrypt(plaintext)
	}
}

func BenchmarkEncryptor_Decrypt(b *testing.B) {
	key := "12345678901234567890123456789012"
	enc, _ := NewEncryptor(key)
	plaintext := []byte("Benchmark plaintext for encryption")
	ciphertext, _ := enc.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Decrypt(ciphertext)
	}
}
