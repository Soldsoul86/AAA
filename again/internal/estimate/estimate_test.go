package estimate

import "testing"

func TestTokens(t *testing.T) {
	if got := Tokens("abcdefgh"); got != 2 {
		t.Fatalf("Tokens(8 chars) = %d, want 2", got)
	}
}

func TestTokensEmpty(t *testing.T) {
	if got := Tokens(""); got != 0 {
		t.Fatalf("Tokens(\"\") = %d, want 0", got)
	}
}
