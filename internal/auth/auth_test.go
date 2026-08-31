package auth_test

import (
	"strings"
	"testing"

	"github.com/spndxyz/quiz/internal/auth"
)

func TestNewTokenLength(t *testing.T) {
	tok, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	if len(tok) != auth.TokenLength {
		t.Errorf("len(NewToken()) = %d, want %d", len(tok), auth.TokenLength)
	}
	if auth.TokenLength < 16 || auth.TokenLength > 32 {
		t.Errorf("TokenLength = %d, want between 16 and 32", auth.TokenLength)
	}
}

func TestNewTokenAlphabet(t *testing.T) {
	const allowed = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	tok, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	for _, r := range tok {
		if !strings.ContainsRune(allowed, r) {
			t.Errorf("NewToken() produced disallowed character %q in %q", r, tok)
		}
	}
}

func TestNewTokenIsRandom(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		tok, err := auth.NewToken()
		if err != nil {
			t.Fatalf("NewToken() error = %v", err)
		}
		if seen[tok] {
			t.Fatalf("NewToken() returned duplicate token %q", tok)
		}
		seen[tok] = true
	}
}
