package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashRefreshToken generates a SHA256 hash of a refresh token.
func HashRefreshToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token)) // Hash the token string
	return hex.EncodeToString(hasher.Sum(nil))
}

// CompareRefreshTokenHash compares a plain refresh token with its stored SHA256 hash.
// It's important that the `token` parameter here is the raw token string, not a hash.
func CompareRefreshTokenHash(token string, storedHash string) bool {
	return HashRefreshToken(token) == storedHash
}
