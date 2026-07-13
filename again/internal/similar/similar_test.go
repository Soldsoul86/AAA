package similar

import "testing"

func approxEqual(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func TestIdenticalStrings(t *testing.T) {
	if got := Jaccard("fix the login bug", "fix the login bug"); got != 1.0 {
		t.Fatalf("Jaccard(identical) = %v, want 1.0", got)
	}
}

func TestCompletelyDifferent(t *testing.T) {
	if got := Jaccard("fix the login bug", "write unit tests"); got != 0.0 {
		t.Fatalf("Jaccard(disjoint) = %v, want 0.0", got)
	}
}

func TestPartialOverlap(t *testing.T) {
	// a = {please, fix, the, login, bug} (5)
	// b = {can, you, fix, the, login, bug, please} (7)
	// intersection = 5 (a is a subset of b), union = 5+7-5 = 7
	got := Jaccard("please fix the login bug", "can you fix the login bug please")
	want := 5.0 / 7.0
	if !approxEqual(got, want, 1e-9) {
		t.Fatalf("Jaccard(partial) = %v, want %v", got, want)
	}
}

func TestEmptyStringIsNeverSimilar(t *testing.T) {
	if got := Jaccard("", ""); got != 0.0 {
		t.Fatalf("Jaccard(empty, empty) = %v, want 0.0 (two empty prompts should not count as repeating)", got)
	}
	if got := Jaccard("hello", ""); got != 0.0 {
		t.Fatalf("Jaccard(text, empty) = %v, want 0.0", got)
	}
}

func TestCaseAndPunctuationInsensitive(t *testing.T) {
	got := Jaccard("Fix the LOGIN bug!!", "fix the login bug")
	if got != 1.0 {
		t.Fatalf("Jaccard should ignore case/punctuation, got %v, want 1.0", got)
	}
}
