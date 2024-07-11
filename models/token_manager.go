package models

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/alexproskurov/web-app/rand"
)

const (
	// The minimum numbers of bytes to be used for each session token.
	MinBytesPerToken = 32
)

type TokenManager struct {
	// BytesPerToken is used to determine how many bytes to use
	// when generating each token. If this value is not set
	// or is less than the MinBytesPerToken const it will be ignored
	// and MinBytesPerToken will be used.
	BytesPerToken int
}

// New generates a token and its hash.
func (t TokenManager) New() (token, tokenHash string, err error) {
	bytesPerToken := t.BytesPerToken
	if bytesPerToken < MinBytesPerToken {
		bytesPerToken = MinBytesPerToken
	}
	token, err = rand.String(bytesPerToken)
	if err != nil {
		return "", "", fmt.Errorf("new token: %w", err)
	}

	tokenHash = t.Hash(token)

	return
}

// Hash returns a URL-safe base64-encoded SHA-256 hash of the token.
func (t TokenManager) Hash(token string) string {
	tokenHash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(tokenHash[:])
}
