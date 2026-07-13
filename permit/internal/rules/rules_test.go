package rules

import "testing"

func TestParseBareTool(t *testing.T) {
	r, err := Parse("Bash")
	if err != nil {
		t.Fatal(err)
	}
	if r.Tool != "Bash" || r.Specifier != "" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseWithSpecifier(t *testing.T) {
	r, err := Parse("Bash(git commit *)")
	if err != nil {
		t.Fatal(err)
	}
	if r.Tool != "Bash" || r.Specifier != "git commit *" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseWebFetchDomain(t *testing.T) {
	r, err := Parse("WebFetch(domain:example.com)")
	if err != nil {
		t.Fatal(err)
	}
	if r.Tool != "WebFetch" || r.Specifier != "domain:example.com" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseToolNameGlob(t *testing.T) {
	if _, err := Parse("mcp__*"); err != nil {
		t.Fatalf("expected mcp__* to parse as a valid bare tool-name glob: %v", err)
	}
}

func TestParseRejectsUnclosedParen(t *testing.T) {
	if _, err := Parse("Bash(git commit *"); err == nil {
		t.Fatal("expected an error for an unclosed paren")
	}
}

func TestParseRejectsEmptyParens(t *testing.T) {
	if _, err := Parse("Bash()"); err == nil {
		t.Fatal("expected an error for empty parentheses")
	}
}

func TestParseRejectsEmpty(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Fatal("expected an error for an empty rule")
	}
	if _, err := Parse("   "); err == nil {
		t.Fatal("expected an error for a whitespace-only rule")
	}
}
