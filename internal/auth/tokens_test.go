package auth

import (
	"bytes"
	"testing"
)

func TestTokensAreRandomAndOnlyTheirHashIsKept(t *testing.T) {
	a, hashA, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := NewToken()
	if a == b || len(a) < 40 {
		t.Errorf("tokens %q %q", a, b)
	}
	if !bytes.Equal(HashToken(a), hashA) || bytes.Equal(HashToken(b), hashA) || len(hashA) != 32 {
		t.Error("the hash matches only its own token")
	}
}
