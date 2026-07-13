package claims

import "testing"

func TestTestsPassClaimed_TruePositives(t *testing.T) {
	cases := []string{
		"I ran the suite and tests pass now.",
		"All tests passing, ready to commit.",
		"Tests passed, so I'll push this.",
		"The tests are passing after that fix.",
		"All 5 tools build, vet, and test clean — tests pass.",
	}
	for _, c := range cases {
		if !TestsPassClaimed(c) {
			t.Errorf("TestsPassClaimed(%q) = false, want true", c)
		}
	}
}

// These are all real phrasings pulled from this project's own transcript —
// discussion ABOUT test-pass claims, not an assertion that tests currently
// pass. A naive substring match on "tests pass" would false-positive on
// every one of these.
func TestTestsPassClaimed_RealFalsePositiveRisks(t *testing.T) {
	cases := []string{
		`until burned, users take "tests pass"/"verified" at face value`,
		`the agent asserts outcomes (tests passed, file read, dependency exists)`,
		`the agent reports "done"/"verified"/"tests pass" without checking`,
		`a check that confirms an agent's claim that tests passed actually matches reality`,
		`does the agent's claim of "done"/"tests pass" match an independently-run check`,
		`AI agent told you something was done or tests passed, and it wasn't true`,
		`A contributor's agent claims tests pass or mischaracterizes a change`,
		`make sure tests pass before shipping`,
		`let's verify tests pass before we merge`,
		`once tests pass, restore is itself undoable`,
		`we need to check that tests pass first`,
	}
	for _, c := range cases {
		if TestsPassClaimed(c) {
			t.Errorf("TestsPassClaimed(%q) = true, want false (discussion/intent, not a claim)", c)
		}
	}
}

func TestTestsPassClaimed_NoMentionAtAll(t *testing.T) {
	if TestsPassClaimed("I fixed the bug and pushed the commit.") {
		t.Error("expected false when tests aren't mentioned at all")
	}
}

func TestTestsPassClaimed_MixedTextFindsRealClaimPastIntentDiscussion(t *testing.T) {
	text := `Let's make sure tests pass before shipping. Ran the suite — tests pass now.`
	if !TestsPassClaimed(text) {
		t.Error("expected true: a real claim appears later in the same text as an intent phrase")
	}
}

func TestTestsPassClaimed_NounPhraseNotAClaim(t *testing.T) {
	cases := []string{
		"some run a second independent test pass",
		"let's do one more test pass before shipping",
		"I'll do a final test pass tomorrow",
	}
	for _, c := range cases {
		if TestsPassClaimed(c) {
			t.Errorf("TestsPassClaimed(%q) = true, want false (noun phrase, not a verb claim)", c)
		}
	}
}
