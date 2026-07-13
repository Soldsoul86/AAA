// Package rules parses and matches Claude Code's own permission rule
// syntax (documented at code.claude.com/docs/en/permissions), so permit
// speaks the exact same language Claude Code does rather than inventing a
// competing one.
package rules

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule is a parsed permission rule of the form "Tool" or "Tool(specifier)".
type Rule struct {
	Tool      string
	Specifier string // empty for a bare tool name
	Raw       string
}

var toolNameGlobRe = regexp.MustCompile(`^[A-Za-z0-9_*]+$`)

// Parse validates and parses a raw rule string using Claude Code's documented
// "Tool" / "Tool(specifier)" syntax.
func Parse(raw string) (Rule, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Rule{}, fmt.Errorf("empty rule")
	}
	open := strings.Index(s, "(")
	if open == -1 {
		if !toolNameGlobRe.MatchString(s) {
			return Rule{}, fmt.Errorf("invalid tool name %q — expected letters, digits, underscore, or *", s)
		}
		return Rule{Tool: s, Raw: s}, nil
	}
	if !strings.HasSuffix(s, ")") {
		return Rule{}, fmt.Errorf("rule %q has an opening ( but doesn't end with )", s)
	}
	tool := s[:open]
	specifier := s[open+1 : len(s)-1]
	if !toolNameGlobRe.MatchString(tool) {
		return Rule{}, fmt.Errorf("invalid tool name %q — expected letters, digits, underscore, or *", tool)
	}
	if specifier == "" {
		return Rule{}, fmt.Errorf("rule %q has empty parentheses", s)
	}
	return Rule{Tool: tool, Specifier: specifier, Raw: s}, nil
}

// MatchesCommand reports whether a Bash-style specifier pattern (using
// Claude Code's wildcard syntax) matches a given command string.
//
// This implements exactly the semantics documented at
// code.claude.com/docs/en/permissions: "*" matches any sequence of
// characters including spaces, and can appear anywhere in the pattern. The
// documented word-boundary behavior ("Bash(ls *) matches ls -la but not
// lsof") falls out naturally from treating everything except "*" as a
// literal match — no special-casing needed, and verified against the
// documentation's own examples in the test suite.
//
// The ":*" trailing suffix is normalized to " *" first, per the
// documented equivalence.
func MatchesCommand(pattern, command string) bool {
	if strings.HasSuffix(pattern, ":*") {
		pattern = strings.TrimSuffix(pattern, ":*") + " *"
	}
	// Join every literal segment (regex-escaped) with ".*" for each "*" in
	// the pattern. Splitting "ls *" on "*" gives ["ls ", ""], which joins
	// to "ls .*" — the literal trailing space in "ls " is what produces
	// the documented word-boundary behavior (it must match an actual
	// space in the command), with no special-casing required.
	segments := strings.Split(pattern, "*")
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = regexp.QuoteMeta(s)
	}
	pat := "^" + strings.Join(escaped, ".*") + "$"

	re, err := regexp.Compile(pat)
	if err != nil {
		return false
	}
	return re.MatchString(command)
}
