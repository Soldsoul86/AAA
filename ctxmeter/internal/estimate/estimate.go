// Package estimate provides a rough, tokenizer-free approximation of how
// much context a chunk of text represents. It intentionally does not try to
// replicate any specific model's real tokenizer — that would require a
// vendored vocabulary file per model and would still go stale. Instead it
// uses the widely-cited ~4-characters-per-token approximation for English
// text, which is close enough to be useful as a live progress indicator,
// not as an exact count.
package estimate

import "encoding/json"

// Tokens estimates the token count of raw text.
func Tokens(text string) int {
	return len(text) / 4
}

// JSONContentLength sums the length of every string value found anywhere in
// a JSON document, recursively. This lets callers estimate how much text
// content a structured log line contributed without needing to know its
// exact field names or schema — useful because session-log formats vary
// between tools and can change without notice.
func JSONContentLength(line []byte) (int, bool) {
	var v interface{}
	if err := json.Unmarshal(line, &v); err != nil {
		return 0, false
	}
	return sumStrings(v), true
}

func sumStrings(v interface{}) int {
	switch t := v.(type) {
	case string:
		return len(t)
	case []interface{}:
		total := 0
		for _, e := range t {
			total += sumStrings(e)
		}
		return total
	case map[string]interface{}:
		total := 0
		for _, e := range t {
			total += sumStrings(e)
		}
		return total
	default:
		return 0
	}
}
