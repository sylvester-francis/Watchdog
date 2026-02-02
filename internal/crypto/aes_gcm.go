package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

const (
	// AES-256 requires a 32-byte key
	aesKeySize = 32
	// GCM standard nonce size
	gcmNonceSize = 12
)

var (
	ErrInvalidKeySize    = errors.New("encryption key must be 32 bytes for AES-256")
	ErrCiphertextTooShort = errors.New("ciphertext too short")
)

// Encryptor handles AES-GCM encryption and decryption.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates a new AES-GCM encryptor with the given key.
// The key must be exactly 32 bytes for AES-256.
func NewEncryptor(key string) (*Encryptor, error) {
	if len(key) != aesKeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts the plaintext using AES-GCM.
// Returns nonce + ciphertext as a single byte slice.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, gcmNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts the ciphertext using AES-GCM.
// Expects nonce + ciphertext as input (as returned by Encrypt).
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < gcmNonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce := ciphertext[:gcmNonceSize]
	encryptedData := ciphertext[gcmNonceSize:]

	plaintext, err := e.gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns the ciphertext as bytes.
func (e *Encryptor) EncryptString(plaintext string) ([]byte, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptString decrypts ciphertext and returns the plaintext as a string.
func (e *Encryptor) DecryptString(ciphertext []byte) (string, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
