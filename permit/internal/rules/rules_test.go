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

// Every case below is taken directly from code.claude.com/docs/en/permissions,
// not invented — this is testing against real documented ground truth.
func TestMatchesCommandDocumentedExamples(t *testing.T) {
	cases := []struct {
		pattern string
		command string
		want    bool
	}{
		// "Bash(ls *) matches ls -la but not lsof"
		{"ls *", "ls -la", true},
		{"ls *", "lsof", false},
		// "Bash(ls*) without a space matches both ls -la and lsof"
		{"ls*", "ls -la", true},
		{"ls*", "lsof", true},
		// "Bash(npm *) matches any command starting with npm "
		{"npm *", "npm install", true},
		{"npm *", "npm run build", true},
		{"npm *", "npmx fake", false},
		// "Bash(* install) matches any command ending with  install"
		{"* install", "npm install", true},
		{"* install", "apt-get install", true},
		{"* install", "installer", false}, // no leading space before "install" here
		// "Bash(git * main) matches git checkout main ... git push origin main"
		{"git * main", "git checkout main", true},
		{"git * main", "git push origin main", true},
		{"git * main", "git merge main", true},
		{"git * main", "git checkout master", false},
		// exact match, no wildcard at all
		{"npm run build", "npm run build", true},
		{"npm run build", "npm run build --verbose", false},
		// ":*" suffix is equivalent to a trailing " *"
		{"ls:*", "ls -la", true},
		{"ls:*", "lsof", false},
	}
	for _, c := range cases {
		got := MatchesCommand(c.pattern, c.command)
		if got != c.want {
			t.Errorf("MatchesCommand(%q, %q) = %v, want %v", c.pattern, c.command, got, c.want)
		}
	}
}

func TestMatchesCommandWildcardSpansSpaces(t *testing.T) {
	// "A single * matches any sequence of characters including spaces,
	// so one wildcard can span multiple arguments. Bash(git *) matches
	// git log --oneline --all"
	if !MatchesCommand("git *", "git log --oneline --all") {
		t.Fatal("a single * should span multiple arguments/spaces")
	}
}
