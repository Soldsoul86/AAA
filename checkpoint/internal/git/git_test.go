package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testRepo creates a throwaway git repo and chdirs the test process into it,
// restoring the original working directory on cleanup.
func testRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHasHead(t *testing.T) {
	dir := testRepo(t)
	if HasHead() {
		t.Fatal("expected HasHead() to be false before any commit exists")
	}
	writeFile(t, dir, "f.txt", "v1")
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-q", "-m", "v1").Run()
	if !HasHead() {
		t.Fatal("expected HasHead() to be true after a commit exists")
	}
}

// This is the bug found during the hardening pass: an AI agent scaffolding
// a brand-new project stages files before the first commit exists, and
// `git stash create` cannot work at all in that state (it needs an
// existing commit as a parent). StashCreate must fall back to
// write-tree + commit-tree instead of failing outright.
func TestStashCreateBeforeFirstCommit(t *testing.T) {
	dir := testRepo(t)
	writeFile(t, dir, "f.txt", "staged before any commit")
	if out, err := exec.Command("git", "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}

	hash, err := StashCreate("pre-HEAD checkpoint")
	if err != nil {
		t.Fatalf("StashCreate before first commit: %v", err)
	}
	if hash == "" {
		t.Fatal("expected a non-empty commit hash for staged content before the first commit")
	}

	diff, err := Diff(hash)
	if err != nil {
		t.Fatalf("Diff against a pre-HEAD checkpoint: %v", err)
	}
	if diff != "" {
		t.Fatalf("expected no diff (checkpoint IS current state), got: %s", diff)
	}
}

// A pre-HEAD checkpoint must still be restorable after the first real
// commit is later made — it's just a plain git commit object.
func TestPreHeadCheckpointSurvivesFirstCommit(t *testing.T) {
	dir := testRepo(t)
	writeFile(t, dir, "f.txt", "v1")
	exec.Command("git", "add", ".").Run()
	preHeadHash, err := StashCreate("before first commit")
	if err != nil || preHeadHash == "" {
		t.Fatalf("StashCreate before first commit failed: %v", err)
	}

	// now make the real first commit, and further modify the file
	exec.Command("git", "commit", "-q", "-m", "init").Run()
	writeFile(t, dir, "f.txt", "v2")

	if err := CheckoutAll(preHeadHash); err != nil {
		t.Fatalf("CheckoutAll(preHeadHash) after a real commit exists: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v1" {
		t.Fatalf("expected restore to bring back %q, got %q", "v1", string(got))
	}
}

func TestHasChangesDetectsOnlyTrackedChanges(t *testing.T) {
	dir := testRepo(t)
	writeFile(t, dir, "tracked.txt", "v1")
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-q", "-m", "v1").Run()

	if changed, err := HasChanges(); err != nil || changed {
		t.Fatalf("expected no changes on a clean tree, got changed=%v err=%v", changed, err)
	}

	writeFile(t, dir, "untracked.txt", "new file, never added")
	if changed, err := HasChanges(); err != nil || changed {
		t.Fatalf("an untracked file must not count as a change, got changed=%v err=%v", changed, err)
	}

	writeFile(t, dir, "tracked.txt", "v2")
	if changed, err := HasChanges(); err != nil || !changed {
		t.Fatalf("a modified tracked file must count as a change, got changed=%v err=%v", changed, err)
	}
}

// Regression test for the concurrency bug found during hardening: firing
// many StashCreate calls at once against the same repo used to fail
// intermittently on git's own lock contention. This should now succeed
// every time via the internal retry.
func TestStashCreateUnderConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency stress test in -short mode")
	}
	dir := testRepo(t)
	writeFile(t, dir, "f.txt", "v0")
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-q", "-m", "v0").Run()

	const n = 20
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			// Each goroutine needs its own file mutation to have
			// something real to snapshot; git itself serializes the
			// actual index/object writes, which is exactly the
			// contention this test is checking StashCreate survives.
			path := filepath.Join(dir, "f.txt")
			content := []byte(strings.Repeat("x", i+1))
			if err := os.WriteFile(path, content, 0o644); err != nil {
				errs <- err
				return
			}
			_, err := StashCreate("concurrent")
			errs <- err
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent StashCreate failed: %v", err)
		}
	}
}
