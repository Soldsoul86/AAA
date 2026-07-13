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
