// Package transcript extracts Bash tool_use commands from Claude Code
// session transcript lines (JSONL). Narrower than the parser built for
// actually (github.com/Soldsoul86/AAA/actually) — exists only needs the
// commands an agent ran, not assistant text or tool results — but built
// against the same confirmed-real JSON shape:
//
//	{"type":"assistant","message":{"content":[
//	  {"type":"tool_use","id":"toolu_...","name":"Bash","input":{"command":"..."}}
//	]}}
package transcript

import "encoding/json"

type rawLine struct {
	Type    string     `json:"type"`
	Message rawMessage `json:"message"`
}

type rawMessage struct {
	Content json.RawMessage `json:"content"`
}

type rawBlock struct {
	Type  string          `json:"type"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type bashInput struct {
	Command string `json:"command"`
}

// BashCommands extracts every Bash tool_use command from one raw
// transcript line. A line that isn't valid JSON, or isn't a recognized
// shape, yields no commands rather than an error — a transcript format
// drift should degrade to "nothing detected," not crash a long-running
// watch.
func BashCommands(line []byte) []string {
	var l rawLine
	if err := json.Unmarshal(line, &l); err != nil || l.Type != "assistant" {
		return nil
	}
	var blocks []rawBlock
	if err := json.Unmarshal(l.Message.Content, &blocks); err != nil {
		return nil
	}
	var commands []string
	for _, b := range blocks {
		if b.Type != "tool_use" || b.Name != "Bash" {
			continue
		}
		var in bashInput
		if err := json.Unmarshal(b.Input, &in); err == nil && in.Command != "" {
			commands = append(commands, in.Command)
		}
	}
	return commands
}
