// Package testrun deterministically classifies shell commands as test runs
// and, from their output text, as passed/failed/unknown. There is no
// structured exit-code field available anywhere in a Claude Code transcript
// — verified directly against a real session: Bash tool_result's "is_error"
// field reflects whether the tool call itself errored, not the shell
// command's exit status, and toolUseResult carries only raw stdout/stderr.
// So this is text-pattern matching against known test-runner output, same
// spirit as again's Jaccard heuristic: no ML, fully deterministic, and
// honest that unrecognized frameworks return Unknown rather than a guess.
package testrun

import "regexp"

// testCommandSubstrings is intentionally a precise, known-runner list
// rather than a loose "contains the word test" match, to avoid classifying
// things like "mkdir test" or "cd test-dir" as test runs.
var testCommandSubstrings = []string{
	"go test",
	"npm test",
	"npm run test",
	"yarn test",
	"pnpm test",
	"pytest",
	"python -m pytest",
	"python3 -m pytest",
	"cargo test",
	"mvn test",
	"mvn verify",
	"./gradlew test",
	"gradle test",
	"rspec",
	"jest",
	"dotnet test",
	"make test",
}

// IsTestCommand reports whether a shell command looks like a test-runner
// invocation, based on a fixed list of common tools. Custom test scripts
// under non-standard names (e.g. "./scripts/check.sh") are not recognized —
// a known, documented limitation rather than a loose guess that would risk
// false positives on unrelated commands.
func IsTestCommand(command string) bool {
	for _, s := range testCommandSubstrings {
		if containsSubstring(command, s) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	n, m := len(s), len(substr)
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == substr {
			return i
		}
	}
	return -1
}

type Result int

const (
	// Unknown means no recognized pass/fail marker was found in the
	// output — deliberately not guessed, so callers should not treat
	// this as either a pass or a failure.
	Unknown Result = iota
	Passed
	Failed
)

func (r Result) String() string {
	switch r {
	case Passed:
		return "passed"
	case Failed:
		return "failed"
	default:
		return "unknown"
	}
}

// failurePatterns are checked first: any match means Failed, regardless of
// whether a success pattern also appears (a run with one failing test among
// many passing ones is still a failed run).
var failurePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^--- FAIL:`),                  // go, verbose
	regexp.MustCompile(`(?m)^FAIL(\s|$)`),                 // go summary line
	regexp.MustCompile(`(?m)^FAILED\b`),                   // pytest verbose
	regexp.MustCompile(`\b[1-9]\d* failed\b`),             // pytest/jest summary (nonzero count only)
	regexp.MustCompile(`(?m)^\s*✕`),                       // jest failing mark
	regexp.MustCompile(`test result: FAILED`),             // cargo
	regexp.MustCompile(`(?m)^\s*\d+\)\s.*(?:\n.*)*Error`), // rspec-style numbered failure (loose)
	regexp.MustCompile(`\bexit (?:status|code):? [1-9]\d*\b`),
	regexp.MustCompile(`(?m)^panic: `),
}

var successPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^ok\s+\S+`),       // go summary line
	regexp.MustCompile(`\b[1-9]\d* passed\b`), // pytest/jest summary (nonzero count only)
	regexp.MustCompile(`test result: ok`),     // cargo
	regexp.MustCompile(`(?m)^--- PASS:`),      // go, verbose, no summary line
}

// Analyze inspects the combined stdout+stderr of a command believed to be a
// test run and returns a deterministic verdict. Failure markers are checked
// before success markers, so a run with any recognized failure is Failed
// even if it also contains passing output.
func Analyze(output string) Result {
	for _, p := range failurePatterns {
		if p.MatchString(output) {
			return Failed
		}
	}
	for _, p := range successPatterns {
		if p.MatchString(output) {
			return Passed
		}
	}
	return Unknown
}
