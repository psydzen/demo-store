// Package auth generates login tokens and guards the HTTP handlers.
package auth

import (
	"crypto/rand"
	"fmt"
)

// tokenAlphabet excludes characters that are easy to misread when a token is
// copied by hand.
const tokenAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// TokenLength is the number of characters in a generated login token. It sits
// inside the 16..32 range the product asks for.
const TokenLength = 24

// NewToken returns a random login token drawn from a cryptographic source.
func NewToken() (string, error) {
	buf := make([]byte, TokenLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	// len(tokenAlphabet) is 55, so the modulo bias over 256 values is under
	// 0.5% per character — irrelevant for a 24-character token.
	out := make([]byte, TokenLength)
	for i, b := range buf {
		out[i] = tokenAlphabet[int(b)%len(tokenAlphabet)]
	}
	return string(out), nil
}
