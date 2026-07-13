package verify

import (
	"testing"

	"github.com/Soldsoul86/AAA/actually/internal/transcript"
)

func bashCall(id, cmd string) transcript.Event {
	return transcript.Event{Kind: transcript.BashCall, ToolUseID: id, Command: cmd}
}
func toolResult(id, output string) transcript.Event {
	return transcript.Event{Kind: transcript.ToolResult, ToolUseID: id, Output: output}
}
func text(s string) transcript.Event {
	return transcript.Event{Kind: transcript.AssistantText, Text: s}
}
func fileEdit(id string) transcript.Event {
	return transcript.Event{Kind: transcript.FileEditCall, ToolUseID: id, ToolName: "Edit"}
}

func TestClaimBeforeAnyTestRun(t *testing.T) {
	s := NewState()
	m := s.Feed(text("All tests pass, ready to ship."))
	if m == nil || m.Kind != NeverRun {
		t.Fatalf("got %+v, want NeverRun", m)
	}
}

func TestClaimAfterRealPassMatchesSilently(t *testing.T) {
	s := NewState()
	s.Feed(bashCall("t1", "go test ./..."))
	s.Feed(toolResult("t1", "PASS\nok  \tpkg\t0.01s"))
	if m := s.Feed(text("Tests pass, done.")); m != nil {
		t.Fatalf("got %+v, want nil (claim matches reality)", m)
	}
}

func TestClaimAfterRealFailureIsFlagged(t *testing.T) {
	s := NewState()
	s.Feed(bashCall("t1", "go test ./..."))
	s.Feed(toolResult("t1", "--- FAIL: TestX (0.00s)\nFAIL\nFAIL\tpkg\t0.01s"))
	m := s.Feed(text("Tests pass now."))
	if m == nil || m.Kind != Failed {
		t.Fatalf("got %+v, want Failed", m)
	}
}

func TestClaimAfterEditSinceLastRunIsStale(t *testing.T) {
	s := NewState()
	s.Feed(bashCall("t1", "go test ./..."))
	s.Feed(toolResult("t1", "PASS\nok  \tpkg\t0.01s"))
	s.Feed(fileEdit("e1"))
	m := s.Feed(text("Tests pass."))
	if m == nil || m.Kind != Stale {
		t.Fatalf("got %+v, want Stale", m)
	}
}

func TestClaimAfterUnrecognizedOutputIsUnverifiable(t *testing.T) {
	s := NewState()
	s.Feed(bashCall("t1", "go test ./..."))
	s.Feed(toolResult("t1", "Building...\nDone.\n"))
	m := s.Feed(text("Tests pass."))
	if m == nil || m.Kind != Unverifiable {
		t.Fatalf("got %+v, want Unverifiable", m)
	}
}

func TestNonTestBashCallDoesNotBecomePendingOrOverwriteLastRun(t *testing.T) {
	s := NewState()
	s.Feed(bashCall("t1", "go test ./..."))
	s.Feed(toolResult("t1", "PASS\nok  \tpkg\t0.01s"))
	// An unrelated command, coincidentally with a matching tool_use_id
	// pattern, must not be tracked or disturb the last real test run.
	s.Feed(bashCall("t2", "ls -la"))
	s.Feed(toolResult("t2", "total 0\ndrwxr-xr-x file.txt"))
	if m := s.Feed(text("Tests pass.")); m != nil {
		t.Fatalf("got %+v, want nil — unrelated command shouldn't affect the verdict", m)
	}
}

func TestMostRecentTestRunWins(t *testing.T) {
	s := NewState()
	s.Feed(bashCall("t1", "go test ./..."))
	s.Feed(toolResult("t1", "--- FAIL: TestX\nFAIL\nFAIL\tpkg\t0.01s"))
	s.Feed(bashCall("t2", "go test ./..."))
	s.Feed(toolResult("t2", "PASS\nok  \tpkg\t0.01s"))
	if m := s.Feed(text("Tests pass now.")); m != nil {
		t.Fatalf("got %+v, want nil — the most recent run passed", m)
	}
}

func TestNonClaimTextProducesNoMismatch(t *testing.T) {
	s := NewState()
	if m := s.Feed(text("I fixed the login bug.")); m != nil {
		t.Fatalf("got %+v, want nil for text with no test-pass claim", m)
	}
}

func TestEditResetsStaleFlagOnNextRealRun(t *testing.T) {
	s := NewState()
	s.Feed(bashCall("t1", "go test ./..."))
	s.Feed(toolResult("t1", "PASS\nok  \tpkg\t0.01s"))
	s.Feed(fileEdit("e1"))
	s.Feed(bashCall("t2", "go test ./..."))
	s.Feed(toolResult("t2", "PASS\nok  \tpkg\t0.01s"))
	if m := s.Feed(text("Tests pass.")); m != nil {
		t.Fatalf("got %+v, want nil — a fresh test run after the edit clears staleness", m)
	}
}
