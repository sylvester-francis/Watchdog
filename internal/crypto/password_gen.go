package crypto

import (
	"crypto/rand"
	"math/big"
)

const passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%&*"

// GenerateRandomPassword generates a cryptographically random password of the given length.
func GenerateRandomPassword(length int) (string, error) {
	result := make([]byte, length)
	max := big.NewInt(int64(len(passwordCharset)))

	for i := range length {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = passwordCharset[n.Int64()]
	}

	return string(result), nil
}
