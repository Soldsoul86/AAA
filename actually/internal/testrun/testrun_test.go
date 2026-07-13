package testrun

import "testing"

func TestIsTestCommand(t *testing.T) {
	cases := map[string]bool{
		"go test ./... -v 2>&1":                  true,
		"go test ./internal/similar/... -v 2>&1": true,
		"npm test":                               true,
		"npm run test -- --watch=false":          true,
		"pytest -q":                              true,
		"python -m pytest tests/":                true,
		"cargo test":                             true,
		"./gradlew test":                         true,
		"mkdir test":                             false,
		"cd test-dir && ls":                      false,
		"git commit -m 'fix test helper'":        false,
		"echo hello":                             false,
	}
	for cmd, want := range cases {
		if got := IsTestCommand(cmd); got != want {
			t.Errorf("IsTestCommand(%q) = %v, want %v", cmd, got, want)
		}
	}
}

// Real output captured from this project's own transcript.
const realGoPassOutput = `=== RUN   TestBasicTrigger
--- PASS: TestBasicTrigger (0.00s)
=== RUN   TestBelowThresholdDoesNotTrigger
--- PASS: TestBelowThresholdDoesNotTrigger (0.00s)
PASS
ok  	github.com/Soldsoul86/AAA/loopkill/internal/detector	0.576s
`

// Real failure text captured from this project's own transcript (checkpoint's
// own error wrapping around a failed git command, not a test framework, but
// representative of the generic "exit status/code" signal).
const realExitStatusFailure = `checkpoint: git stash create staged, no HEAD yet: exit status 1: You do not have the initial commit yet
exit code: 1
`

func TestAnalyzeRealGoPass(t *testing.T) {
	if got := Analyze(realGoPassOutput); got != Passed {
		t.Fatalf("Analyze(real go pass output) = %v, want Passed", got)
	}
}

func TestAnalyzeRealExitStatusFailure(t *testing.T) {
	if got := Analyze(realExitStatusFailure); got != Failed {
		t.Fatalf("Analyze(real exit-status failure output) = %v, want Failed", got)
	}
}

func TestAnalyzeGoFail(t *testing.T) {
	out := `=== RUN   TestSomething
--- FAIL: TestSomething (0.00s)
    thing_test.go:10: expected 1, got 2
FAIL
FAIL	github.com/example/pkg	0.002s
`
	if got := Analyze(out); got != Failed {
		t.Fatalf("Analyze(go fail output) = %v, want Failed", got)
	}
}

func TestAnalyzePytestPass(t *testing.T) {
	out := "collected 5 items\n\ntests/test_x.py .....                                    [100%]\n\n5 passed in 0.12s\n"
	if got := Analyze(out); got != Passed {
		t.Fatalf("Analyze(pytest pass) = %v, want Passed", got)
	}
}

func TestAnalyzePytestFail(t *testing.T) {
	out := "tests/test_x.py FF                                                [100%]\n\n2 failed, 3 passed in 0.12s\n"
	if got := Analyze(out); got != Failed {
		t.Fatalf("Analyze(pytest with failures) = %v, want Failed", got)
	}
}

func TestAnalyzeCargoPass(t *testing.T) {
	out := "running 3 tests\ntest result: ok. 3 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out\n"
	if got := Analyze(out); got != Passed {
		t.Fatalf("Analyze(cargo pass) = %v, want Passed", got)
	}
}

func TestAnalyzeCargoFail(t *testing.T) {
	out := "running 3 tests\ntest result: FAILED. 2 passed; 1 failed; 0 ignored; 0 measured; 0 filtered out\n"
	if got := Analyze(out); got != Failed {
		t.Fatalf("Analyze(cargo fail) = %v, want Failed", got)
	}
}

func TestAnalyzeUnknownWhenNoRecognizedMarker(t *testing.T) {
	out := "Building project...\nDone in 1.2s\n"
	if got := Analyze(out); got != Unknown {
		t.Fatalf("Analyze(unrecognized output) = %v, want Unknown", got)
	}
}

func TestAnalyzeFailureWinsOverPartialSuccessMarkers(t *testing.T) {
	// A run with one failing test among passing ones must still be Failed.
	out := "--- PASS: TestA (0.00s)\n--- FAIL: TestB (0.00s)\nFAIL\nFAIL\tpkg\t0.01s\n"
	if got := Analyze(out); got != Failed {
		t.Fatalf("Analyze(mixed pass/fail) = %v, want Failed", got)
	}
}
