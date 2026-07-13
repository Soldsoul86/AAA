// Package transcript parses Claude Code session transcript lines (JSONL)
// into the events actually cares about: assistant text (potential claims),
// Bash tool calls and their results, and file-editing tool calls (used to
// detect a test result gone stale after a later edit).
//
// The shape parsed here was confirmed directly against a real, live Claude
// Code transcript file, not just documentation or synthetic fixtures:
//
//	{"type":"assistant","message":{"content":[
//	  {"type":"text","text":"..."},
//	  {"type":"tool_use","id":"toolu_...","name":"Bash","input":{"command":"..."}}
//	]}}
//	{"type":"user","message":{"content":[
//	  {"type":"tool_result","tool_use_id":"toolu_...","content":"...","is_error":false}
//	]}}
//
// tool_result's "content" was observed as a plain string in every real Bash
// result sampled, but the array-of-text-blocks shape from the Anthropic
// Messages API is handled defensively too, same posture as again's
// transcript parser. This is an observed, not officially documented, log
// format and may change without notice.
package transcript

import "encoding/json"

type EventKind int

const (
	AssistantText EventKind = iota
	BashCall
	FileEditCall
	ToolResult
)

type Event struct {
	Kind      EventKind
	Text      string // AssistantText
	ToolUseID string // BashCall, FileEditCall, ToolResult
	Command   string // BashCall
	ToolName  string // FileEditCall: Edit, Write, MultiEdit, or NotebookEdit
	Output    string // ToolResult: flattened text content
	IsError   bool   // ToolResult
}

var fileEditToolNames = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"MultiEdit":    true,
	"NotebookEdit": true,
}

type rawLine struct {
	Type    string     `json:"type"`
	Message rawMessage `json:"message"`
}

type rawMessage struct {
	Content json.RawMessage `json:"content"`
}

type rawBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

type bashInput struct {
	Command string `json:"command"`
}

// ParseLine extracts zero or more Events from one raw transcript line. A
// line that isn't valid JSON, or isn't a recognized shape, yields no events
// rather than an error — a transcript format drift should degrade to
// "nothing detected," not crash a long-running watch.
func ParseLine(line []byte) []Event {
	var l rawLine
	if err := json.Unmarshal(line, &l); err != nil {
		return nil
	}
	if l.Type != "assistant" && l.Type != "user" {
		return nil
	}

	var blocks []rawBlock
	if err := json.Unmarshal(l.Message.Content, &blocks); err != nil {
		return nil
	}

	var events []Event
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if l.Type == "assistant" && b.Text != "" {
				events = append(events, Event{Kind: AssistantText, Text: b.Text})
			}
		case "tool_use":
			if l.Type != "assistant" {
				continue
			}
			if b.Name == "Bash" {
				var in bashInput
				if err := json.Unmarshal(b.Input, &in); err == nil {
					events = append(events, Event{Kind: BashCall, ToolUseID: b.ID, Command: in.Command})
				}
			} else if fileEditToolNames[b.Name] {
				events = append(events, Event{Kind: FileEditCall, ToolUseID: b.ID, ToolName: b.Name})
			}
		case "tool_result":
			if l.Type != "user" {
				continue
			}
			events = append(events, Event{
				Kind:      ToolResult,
				ToolUseID: b.ToolUseID,
				Output:    flattenContent(b.Content),
				IsError:   b.IsError,
			})
		}
	}
	return events
}

// flattenContent handles tool_result's "content" being either a plain
// string (the shape observed for every real Bash result sampled) or an
// array of {"type":"text","text":"..."} blocks (the general Anthropic
// Messages API shape, handled defensively though not directly observed
// here).
func flattenContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []rawBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		out := ""
		for _, b := range blocks {
			if b.Type == "text" {
				out += b.Text
			}
		}
		return out
	}
	return ""
}
