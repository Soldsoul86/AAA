// Package similar computes a simple, transparent similarity score between
// two pieces of text — no ML, no API calls, nothing that can silently
// change behavior between versions. It's Jaccard similarity over lowercased
// word sets: intentionally crude, so its behavior is easy to reason about
// and to unit test exactly.
package similar

import "strings"

// Words normalizes text into a lowercased set of word tokens.
func Words(s string) map[string]struct{} {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	set := map[string]struct{}{}
	for _, w := range strings.Fields(b.String()) {
		set[w] = struct{}{}
	}
	return set
}

// Jaccard returns the Jaccard similarity (|intersection| / |union|) between
// two strings' word sets, from 0.0 (nothing in common) to 1.0 (identical
// word sets). Two empty strings are defined as similarity 0, not 1 — an
// empty prompt shouldn't count as "repeating" another empty prompt.
func Jaccard(a, b string) float64 {
	wa, wb := Words(a), Words(b)
	if len(wa) == 0 || len(wb) == 0 {
		return 0
	}
	intersection := 0
	for w := range wa {
		if _, ok := wb[w]; ok {
			intersection++
		}
	}
	union := len(wa) + len(wb) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
