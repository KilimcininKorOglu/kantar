package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateAPIToken creates a new random API token and returns both
// the plaintext token (shown once to the user) and its SHA-256 hash (stored in DB).
func GenerateAPIToken() (plaintext string, hash string, prefix string, err error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", "", fmt.Errorf("generating random token: %w", err)
	}

	plaintext = "kntr_" + hex.EncodeToString(tokenBytes)
	prefix = plaintext[:12]
	hash = HashAPIToken(plaintext)

	return plaintext, hash, prefix, nil
}

// HashAPIToken returns the SHA-256 hash of an API token.
func HashAPIToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
