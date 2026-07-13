package transcript

import "testing"

func TestPlainStringContent(t *testing.T) {
	text, ok := ExtractUserText([]byte(`{"role":"user","content":"fix the login bug"}`))
	if !ok {
		t.Fatal("expected a user turn to be recognized")
	}
	if text != "fix the login bug" {
		t.Fatalf("got %q", text)
	}
}

func TestBlockArrayContent(t *testing.T) {
	line := `{"role":"user","content":[{"type":"text","text":"fix the login bug"}]}`
	text, ok := ExtractUserText([]byte(line))
	if !ok {
		t.Fatal("expected a user turn to be recognized")
	}
	if text != "fix the login bug" {
		t.Fatalf("got %q", text)
	}
}

func TestNestedMessageWrapper(t *testing.T) {
	line := `{"type":"user","message":{"role":"user","content":"fix the login bug"}}`
	text, ok := ExtractUserText([]byte(line))
	if !ok {
		t.Fatal("expected a user turn to be recognized")
	}
	if text != "fix the login bug" {
		t.Fatalf("got %q", text)
	}
}

func TestAssistantTurnNotExtracted(t *testing.T) {
	line := `{"role":"assistant","content":"I fixed the login bug"}`
	_, ok := ExtractUserText([]byte(line))
	if ok {
		t.Fatal("an assistant turn should not be extracted as user text")
	}
}

func TestToolResultBlocksIgnored(t *testing.T) {
	line := `{"role":"user","content":[{"type":"tool_result","content":"some output"},{"type":"text","text":"fix it"}]}`
	text, ok := ExtractUserText([]byte(line))
	if !ok {
		t.Fatal("expected a user turn to be recognized")
	}
	if text != "fix it" {
		t.Fatalf("got %q, want only the text block's content, tool_result ignored", text)
	}
}

func TestUnrecognizedShapeReturnsFalse(t *testing.T) {
	_, ok := ExtractUserText([]byte(`{"something":"else entirely"}`))
	if ok {
		t.Fatal("an unrecognized shape should return ok=false, not a guess")
	}
}

func TestInvalidJSON(t *testing.T) {
	_, ok := ExtractUserText([]byte(`not json`))
	if ok {
		t.Fatal("invalid JSON should return ok=false")
	}
}
