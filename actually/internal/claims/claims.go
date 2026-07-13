// Package claims detects, in assistant text, a claim that tests currently
// pass — as distinct from merely discussing the phrase. This distinction is
// not academic: a real Claude Code transcript is full of sentences like
// "make sure tests pass before shipping" or a design doc that says
// "the check confirms tests passed" as a description of a *tool*, not a
// completed action. A naive substring match on "tests pass" flags both
// identically. This package filters out the intent/discussion cases
// deterministically — no ML, just a word-window exclusion list — biased
// toward under-catching rather than nagging on a false claim, matching the
// project's general stance against overclaiming.
package claims

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// The bare, unsuffixed form of "pass" only counts as the verb ("tests
// pass" = tests succeed) when "test" is plural. Singular "test pass" (no
// suffix) is almost always the noun phrase "a test pass" — a round of
// testing — as in "ran a second independent test pass"; real transcript
// text confirmed this exact false positive. Suffixed forms (passes/
// passed/passing) are unambiguous verb forms regardless of singular or
// plural "test", so those aren't restricted.
var claimPattern = regexp.MustCompile(
	`(?i)\b(?:all\s+)?(?:tests\s+pass\b|tests?\s+(?:is\s+|are\s+)?(?:now\s+|already\s+|successfully\s+|finally\s+|still\s+)?pass(?:es|ed|ing)\b)`,
)

// intentMarkers precede a claim-shaped phrase when it's actually a plan,
// requirement, or description rather than a report of something that
// already happened. Checked against the text immediately before a match.
var intentMarkers = []string{
	"make sure", "make certain", "ensure", "ensuring",
	"let's", "let us", "let me",
	"i'll", "we'll", "you'll", "will ",
	"should ",
	"once ", "after ", "before ",
	"need to", "needs to", "needed to",
	"want to", "wants to", "wanted to",
	"going to", "gonna",
	"plan to", "planning to",
	"verify that", "verifying that",
	"confirm that", "confirming that",
	"check that", "checking that",
	"until ", "so that",
	"if ", "when ", "whether ",
	"hope ", "hoping ",
	"trying to", "try to",
	"claims", "claim that", "claimed",
	"asserts", "assert that", "asserted",
	"says", "said that", "saying",
	"reports", "reported that", "reporting",
	"something was done or",
}

// falsityMarkers follow a claim-shaped phrase when the sentence is actually
// describing a claim that turned out to be wrong — "tests passed, and it
// wasn't true" is a report of deception, not a report of passing tests.
var falsityMarkers = []string{
	"wasn't true", "isn't true", "was false", "is false",
	"wasn't the case", "isn't the case", "turned out",
	"didn't actually", "hadn't actually", "actually didn't", "actually hadn't",
	"but it wasn't", "but they weren't",
}

const windowChars = 60

// TestsPassClaimed reports whether text contains an assertion that tests
// currently pass, excluding discussion of the concept or a stated
// intent/requirement rather than a completed report.
func TestsPassClaimed(text string) bool {
	matches := claimPattern.FindAllStringIndex(text, -1)
	for _, m := range matches {
		start, end := m[0], m[1]
		if isQuoted(text, start) {
			continue
		}

		windowStart := sentenceBoundedStart(text, start)
		before := strings.ToLower(text[windowStart:start])
		excluded := false
		for _, marker := range intentMarkers {
			if strings.Contains(before, marker) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		windowEnd := end + windowChars
		if windowEnd > len(text) {
			windowEnd = len(text)
		}
		after := strings.ToLower(text[end:windowEnd])
		for _, marker := range falsityMarkers {
			if strings.Contains(after, marker) {
				excluded = true
				break
			}
		}
		if !excluded {
			return true
		}
	}
	return false
}

// sentenceBoundedStart returns the start of the exclusion-check window
// before pos: up to windowChars back, but never crossing a sentence
// boundary — otherwise an intent marker in a previous, unrelated sentence
// ("Let's ship before Friday. Tests pass now.") would wrongly suppress a
// genuine claim in the following sentence.
func sentenceBoundedStart(text string, pos int) int {
	limit := pos - windowChars
	if limit < 0 {
		limit = 0
	}
	for i := pos - 1; i >= limit; i-- {
		switch text[i] {
		case '.', '!', '?', '\n':
			return i + 1
		}
	}
	return limit
}

// isQuoted reports whether the character immediately preceding pos (skipping
// a leading "the " if present) is a quote mark — a strong signal the phrase
// is being cited or discussed rather than asserted as fact.
func isQuoted(text string, pos int) bool {
	i := pos
	for i > 0 && text[i-1] == ' ' {
		i--
	}
	if i >= 4 && strings.EqualFold(text[i-4:i], "the ") {
		i -= 4
		for i > 0 && text[i-1] == ' ' {
			i--
		}
	}
	if i == 0 {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(text[:i])
	switch r {
	case '"', '\'', '“', '‘':
		return true
	default:
		return false
	}
}
