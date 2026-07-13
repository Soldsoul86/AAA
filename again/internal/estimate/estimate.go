// Package estimate provides a rough, tokenizer-free token count — the same
// ~4-characters-per-token approximation used by ctxmeter, deliberately
// duplicated here rather than shared via a cross-tool import. Each tool in
// this repo is meant to build and install independently with zero
// dependency on the others; a shared internal package would violate that.
// If this formula ever needs to change, it needs to change in both places
// — that's a real, known tradeoff, not an oversight.
package estimate

// Tokens estimates the token count of raw text. Not a real tokenizer for
// any specific model — a rough proportional estimate, good enough to
// answer "roughly how many tokens was this," not to bill against.
func Tokens(text string) int {
	return len(text) / 4
}
