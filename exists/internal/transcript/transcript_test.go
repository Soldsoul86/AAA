package transcript

import "testing"

// Captured verbatim (trimmed) from a real Claude Code transcript.
const realLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Installing now."},{"type":"tool_use","id":"toolu_01WmM29K3jeZY2TbLZeFRYtC","name":"Bash","input":{"command":"npm install lodash","description":"Install lodash"}}]}}`

func TestBashCommandsRealLine(t *testing.T) {
	cmds := BashCommands([]byte(realLine))
	if len(cmds) != 1 || cmds[0] != "npm install lodash" {
		t.Fatalf("got %v, want [\"npm install lodash\"]", cmds)
	}
}

func TestBashCommandsMultipleInOneLine(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[
		{"type":"tool_use","name":"Bash","input":{"command":"npm install lodash"}},
		{"type":"tool_use","name":"Bash","input":{"command":"pip install requests"}}
	]}}`
	cmds := BashCommands([]byte(line))
	if len(cmds) != 2 || cmds[0] != "npm install lodash" || cmds[1] != "pip install requests" {
		t.Fatalf("got %v", cmds)
	}
}

func TestBashCommandsIgnoresNonBashTools(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"x.go"}}]}}`
	if cmds := BashCommands([]byte(line)); cmds != nil {
		t.Fatalf("got %v, want nil for a non-Bash tool", cmds)
	}
}

func TestBashCommandsIgnoresUserLines(t *testing.T) {
	line := `{"type":"user","message":{"content":[{"type":"tool_result","content":"npm install lodash"}]}}`
	if cmds := BashCommands([]byte(line)); cmds != nil {
		t.Fatalf("got %v, want nil for a user/tool_result line", cmds)
	}
}

func TestBashCommandsMalformedJSON(t *testing.T) {
	if cmds := BashCommands([]byte("{not valid")); cmds != nil {
		t.Fatalf("got %v, want nil for malformed JSON", cmds)
	}
}

func TestBashCommandsEmptyMessage(t *testing.T) {
	if cmds := BashCommands([]byte(`{"type":"assistant"}`)); cmds != nil {
		t.Fatalf("got %v, want nil when message/content is absent", cmds)
	}
}
