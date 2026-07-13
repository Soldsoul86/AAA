package transcript

import "testing"

// realAssistantLine is captured verbatim (trimmed) from this project's own
// live transcript.
const realAssistantLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Building now."},{"type":"tool_use","id":"toolu_01WmM29K3jeZY2TbLZeFRYtC","name":"Bash","input":{"command":"go test ./... -v 2>&1","description":"Run tests"},"caller":{"type":"direct"}}]}}`

const realToolResultLine = `{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_01WmM29K3jeZY2TbLZeFRYtC","type":"tool_result","content":"PASS\nok  \tpkg\t0.01s","is_error":false}]}}`

func TestParseLineRealAssistant(t *testing.T) {
	events := ParseLine([]byte(realAssistantLine))
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (text + bash call): %+v", len(events), events)
	}
	if events[0].Kind != AssistantText || events[0].Text != "Building now." {
		t.Errorf("event 0 = %+v, want AssistantText %q", events[0], "Building now.")
	}
	if events[1].Kind != BashCall || events[1].Command != "go test ./... -v 2>&1" {
		t.Errorf("event 1 = %+v, want BashCall with the real command", events[1])
	}
	if events[1].ToolUseID != "toolu_01WmM29K3jeZY2TbLZeFRYtC" {
		t.Errorf("ToolUseID = %q, want the real captured id", events[1].ToolUseID)
	}
}

func TestParseLineRealToolResult(t *testing.T) {
	events := ParseLine([]byte(realToolResultLine))
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	e := events[0]
	if e.Kind != ToolResult || e.ToolUseID != "toolu_01WmM29K3jeZY2TbLZeFRYtC" {
		t.Errorf("event = %+v, want matching ToolResult", e)
	}
	if e.Output != "PASS\nok  \tpkg\t0.01s" {
		t.Errorf("Output = %q", e.Output)
	}
	if e.IsError {
		t.Error("IsError = true, want false")
	}
}

func TestParseLineFileEditCall(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_x","name":"Edit","input":{"file_path":"main.go"}}]}}`
	events := ParseLine([]byte(line))
	if len(events) != 1 || events[0].Kind != FileEditCall || events[0].ToolName != "Edit" {
		t.Fatalf("got %+v, want a single FileEditCall for Edit", events)
	}
}

func TestParseLineIgnoresOtherToolUse(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_x","name":"Grep","input":{"pattern":"foo"}}]}}`
	events := ParseLine([]byte(line))
	if len(events) != 0 {
		t.Fatalf("got %+v, want no events for an unrelated tool", events)
	}
}

func TestParseLineToolResultArrayContent(t *testing.T) {
	line := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_y","content":[{"type":"text","text":"hello "},{"type":"text","text":"world"}],"is_error":false}]}}`
	events := ParseLine([]byte(line))
	if len(events) != 1 || events[0].Output != "hello world" {
		t.Fatalf("got %+v, want flattened array content \"hello world\"", events)
	}
}

func TestParseLineMalformedJSON(t *testing.T) {
	events := ParseLine([]byte("{not valid json"))
	if events != nil {
		t.Fatalf("got %+v, want nil for malformed JSON", events)
	}
}

func TestParseLineUnknownType(t *testing.T) {
	events := ParseLine([]byte(`{"type":"queue-operation","message":{}}`))
	if events != nil {
		t.Fatalf("got %+v, want nil for an unrecognized top-level type", events)
	}
}

func TestParseLineEmptyMessage(t *testing.T) {
	events := ParseLine([]byte(`{"type":"assistant"}`))
	if events != nil {
		t.Fatalf("got %+v, want nil when message/content is absent", events)
	}
}

func TestParseLineIsErrorTrue(t *testing.T) {
	line := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_z","content":"boom","is_error":true}]}}`
	events := ParseLine([]byte(line))
	if len(events) != 1 || !events[0].IsError {
		t.Fatalf("got %+v, want IsError true", events)
	}
}
