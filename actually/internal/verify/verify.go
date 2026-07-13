// Package verify correlates assistant claims that "tests pass" against the
// most recent real test run seen in the same transcript, distinguishing
// three distinct ways a claim can fail to hold up: no test was ever run,
// the last run predates a later file edit (stale), or the last run
// actually failed. Unknown-verdict runs get a separate, softer note rather
// than being folded into "failed" — the tool couldn't verify, which is not
// the same claim as "the tool verified this is wrong."
package verify

import (
	"fmt"

	"github.com/Soldsoul86/AAA/actually/internal/claims"
	"github.com/Soldsoul86/AAA/actually/internal/testrun"
	"github.com/Soldsoul86/AAA/actually/internal/transcript"
)

type MismatchKind string

const (
	NeverRun     MismatchKind = "never-run"
	Stale        MismatchKind = "stale"
	Failed       MismatchKind = "failed"
	Unverifiable MismatchKind = "unverifiable"
)

type Mismatch struct {
	Kind    MismatchKind
	Message string
}

type testRunRecord struct {
	Command string
	Verdict testrun.Result
}

// State is a single watched session's correlation state. Not safe for
// concurrent use — actually watches one transcript at a time.
type State struct {
	pendingBash map[string]string
	lastTestRun *testRunRecord
	editedSince bool
}

func NewState() *State {
	return &State{pendingBash: map[string]string{}}
}

// Feed processes one transcript event and returns a Mismatch if it was a
// claim that doesn't hold up against the session's actual test history, or
// nil for every other event (including a claim that does hold up).
func (s *State) Feed(ev transcript.Event) *Mismatch {
	switch ev.Kind {
	case transcript.BashCall:
		if testrun.IsTestCommand(ev.Command) {
			s.pendingBash[ev.ToolUseID] = ev.Command
		}
	case transcript.FileEditCall:
		s.editedSince = true
	case transcript.ToolResult:
		if cmd, ok := s.pendingBash[ev.ToolUseID]; ok {
			delete(s.pendingBash, ev.ToolUseID)
			s.lastTestRun = &testRunRecord{Command: cmd, Verdict: testrun.Analyze(ev.Output)}
			s.editedSince = false
		}
	case transcript.AssistantText:
		if claims.TestsPassClaimed(ev.Text) {
			return s.checkClaim()
		}
	}
	return nil
}

func (s *State) checkClaim() *Mismatch {
	switch {
	case s.lastTestRun == nil:
		return &Mismatch{
			Kind:    NeverRun,
			Message: "claimed tests pass, but no test command has been run yet this session",
		}
	case s.editedSince:
		return &Mismatch{
			Kind: Stale,
			Message: fmt.Sprintf(
				"claimed tests pass, but files were edited after the last test run (%s) — that result may be stale",
				s.lastTestRun.Command,
			),
		}
	case s.lastTestRun.Verdict == testrun.Failed:
		return &Mismatch{
			Kind: Failed,
			Message: fmt.Sprintf(
				"claimed tests pass, but the last test run (%s) failed",
				s.lastTestRun.Command,
			),
		}
	case s.lastTestRun.Verdict == testrun.Unknown:
		return &Mismatch{
			Kind: Unverifiable,
			Message: fmt.Sprintf(
				"claimed tests pass; the last test run's (%s) output didn't match any known pass/fail pattern, so this can't be verified",
				s.lastTestRun.Command,
			),
		}
	default:
		return nil
	}
}
