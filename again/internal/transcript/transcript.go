// Package transcript extracts user-authored messages from a JSONL session
// log of unknown, possibly-varying schema.
//
// This is the least certain part of this tool: it was built against the
// commonly-documented shape of the Anthropic Messages API (a "role" field
// of "user"/"assistant", with "content" as either a plain string or an
// array of {"type":"text","text":"..."} blocks) and an observed top-level
// {"type":"user", "message": {...}} wrapper some session-log formats use.
// It has NOT been verified against a real Claude Code transcript file. If
// it finds nothing, that's the most likely reason — please open an issue
// with a redacted sample line rather than assume it's silently working.
package transcript

import "encoding/json"

// ExtractUserText returns the user-authored text of a JSONL line, and
// whether this line was recognized as a user turn at all.
func ExtractUserText(line []byte) (string, bool) {
	var raw map[string]interface{}
	if err := json.Unmarshal(line, &raw); err != nil {
		return "", false
	}

	// Shape 1: top-level {"role": "user", "content": ...}
	if role, _ := raw["role"].(string); role == "user" {
		return extractContent(raw["content"]), true
	}

	// Shape 2: {"type": "user", "message": {"role": "user", "content": ...}}
	if typ, _ := raw["type"].(string); typ == "user" {
		if msg, ok := raw["message"].(map[string]interface{}); ok {
			if role, _ := msg["role"].(string); role == "user" || role == "" {
				return extractContent(msg["content"]), true
			}
		}
	}

	return "", false
}

// extractContent handles content as a plain string, or as an array of
// {"type":"text","text":"..."} blocks (the Anthropic Messages API shape),
// concatenating any text blocks found. Non-text blocks (tool_result, etc.)
// are ignored — "again" only cares about what the user actually typed.
func extractContent(v interface{}) string {
	switch c := v.(type) {
	case string:
		return c
	case []interface{}:
		var out string
		for _, block := range c {
			m, ok := block.(map[string]interface{})
			if !ok {
				continue
			}
			if t, _ := m["type"].(string); t == "text" {
				if text, ok := m["text"].(string); ok {
					if out != "" {
						out += " "
					}
					out += text
				}
			}
		}
		return out
	default:
		return ""
	}
}
