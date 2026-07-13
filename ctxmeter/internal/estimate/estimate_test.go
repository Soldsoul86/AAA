package estimate

import "testing"

func TestTokens(t *testing.T) {
	if got := Tokens("abcdefgh"); got != 2 {
		t.Fatalf("Tokens(8 chars) = %d, want 2", got)
	}
}

func TestJSONContentLengthFlatObject(t *testing.T) {
	n, ok := JSONContentLength([]byte(`{"role":"user","content":"hello world"}`))
	if !ok {
		t.Fatal("expected valid JSON to parse")
	}
	want := len("user") + len("hello world")
	if n != want {
		t.Fatalf("JSONContentLength = %d, want %d", n, want)
	}
}

func TestJSONContentLengthNested(t *testing.T) {
	n, ok := JSONContentLength([]byte(`{"message":{"content":[{"type":"text","text":"abcdefghij"}]}}`))
	if !ok {
		t.Fatal("expected valid JSON to parse")
	}
	// sumStrings only walks map *values*, never keys — so "content" and the
	// field name "text" don't count, only the actual string values do:
	// "text" (the value of "type") and "abcdefghij" (the value of "text").
	want := len("text") + len("abcdefghij")
	if n != want {
		t.Fatalf("JSONContentLength = %d, want %d", n, want)
	}
}

func TestJSONContentLengthIgnoresNumbersAndBools(t *testing.T) {
	n, ok := JSONContentLength([]byte(`{"count":12345,"done":true,"text":"ab"}`))
	if !ok {
		t.Fatal("expected valid JSON to parse")
	}
	// Only the string value counts — "count" and "done" are keys, and their
	// values (a number, a bool) aren't strings.
	want := len("ab")
	if n != want {
		t.Fatalf("JSONContentLength = %d, want %d (numbers/bools should not be counted)", n, want)
	}
}

func TestJSONContentLengthInvalidJSON(t *testing.T) {
	_, ok := JSONContentLength([]byte(`not json at all`))
	if ok {
		t.Fatal("expected invalid JSON to return ok=false")
	}
}
