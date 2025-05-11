package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateSecureRandomString generates a cryptographically secure random string of the specified byte length,
// then hex encodes it. For example, lengthInBytes=32 will result in a 64-character hex string.
func GenerateSecureRandomString(lengthInBytes int) (string, error) {
	if lengthInBytes <= 0 {
		return "", fmt.Errorf("lengthInBytes must be positive")
	}
	b := make([]byte, lengthInBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
