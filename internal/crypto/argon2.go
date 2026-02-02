package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters following OWASP recommendations.
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

var (
	ErrInvalidHash         = errors.New("invalid password hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

// PasswordHasher handles password hashing and verification using Argon2id.
type PasswordHasher struct{}

// NewPasswordHasher creates a new password hasher.
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

// Hash generates an Argon2id hash of the password.
// Returns a string in the format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>.
func (h *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// Verify checks if the password matches the hash.
// Uses constant-time comparison to prevent timing attacks.
func (h *PasswordHasher) Verify(password, encodedHash string) (bool, error) {
	params, salt, hash, err := h.decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.time,
		params.memory,
		params.threads,
		params.keyLen,
	)

	return subtle.ConstantTimeCompare(hash, otherHash) == 1, nil
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
}

func (h *PasswordHasher) decodeHash(encodedHash string) (params *argon2Params, salt, hash []byte, err error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", &version); scanErr != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params = &argon2Params{}
	if _, scanErr := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &params.threads); scanErr != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	// #nosec G115 - hash length is always small (32 bytes for argon2id).
	params.keyLen = uint32(len(hash))

	return params, salt, hash, nil
}
