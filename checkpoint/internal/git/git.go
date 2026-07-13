// Package git wraps the small set of git plumbing commands checkpoint needs.
// Checkpoints are plain git commit objects created with `git stash create` —
// there is no custom storage format. Anyone can inspect or restore a
// checkpoint with plain git if this tool disappears tomorrow.
package git

import (
	"bytes"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

func run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// IsInsideWorkTree reports whether the current directory is inside a git working tree.
func IsInsideWorkTree() bool {
	out, err := run("rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// RepoRoot returns the absolute path to the top level of the working tree.
func RepoRoot() (string, error) {
	return run("rev-parse", "--show-toplevel")
}

// GitDir returns the absolute path to the repository's .git directory.
func GitDir() (string, error) {
	dir, err := run("rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	return dir, nil
}

// HasChanges reports whether there are any tracked, uncommitted changes
// (staged or unstaged) in the working tree.
func HasChanges() (bool, error) {
	out, err := run("status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// HasHead reports whether the repository has at least one commit.
func HasHead() bool {
	_, err := run("rev-parse", "--verify", "HEAD")
	return err == nil
}

// StashCreate snapshots the current index + working tree (tracked files only)
// into a commit object without touching HEAD, the index, the working tree,
// or the stash ref. Returns the empty string if there is nothing to snapshot.
//
// In a repository with no commits yet, `git stash create` cannot work — a
// stash entry needs an existing commit as its parent, and there isn't one.
// That's a real, common state: an AI agent scaffolding a brand-new project
// before the first commit. In that specific case we fall back to
// write-tree + commit-tree, which captures currently staged content into a
// standalone, parentless commit object — still just a plain git object,
// nothing custom, matching how every other checkpoint works.
func StashCreate(label string) (string, error) {
	if !HasHead() {
		return stashCreateNoHead(label)
	}
	args := []string{"stash", "create"}
	if label != "" {
		args = append(args, label)
	}

	// Concurrent invocations (a real scenario: an agent firing several
	// tool calls in quick succession, each triggering a hook-driven save)
	// can collide on git's own internal locks in more than one way — not
	// always with "index.lock" in the message. Retrying on any failure a
	// few times, briefly, is safe here (StashCreate has no side effects
	// to double up on failure) and means a burst of simultaneous saves
	// doesn't dump raw git internals to the terminal during otherwise
	// completely normal use. If it's still failing after 5 attempts, it's
	// no longer transient contention and should surface as a real error.
	const maxAttempts = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		hash, err := run(args...)
		if err == nil {
			return hash, nil
		}
		lastErr = err
		time.Sleep(time.Duration(20+rand.Intn(60)) * time.Millisecond * time.Duration(attempt+1))
	}
	return "", fmt.Errorf("gave up after %d attempts, still contended: %w", maxAttempts, lastErr)
}

func stashCreateNoHead(label string) (string, error) {
	tree, err := run("write-tree")
	if err != nil {
		return "", err
	}
	msg := "checkpoint (before the first commit)"
	if label != "" {
		msg = label
	}
	hash, err := run("commit-tree", tree, "-m", msg)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// Diff returns the diff between a checkpoint commit and the current working tree.
func Diff(hash string) (string, error) {
	// git diff exits 1 when there are differences — that's not an error for us.
	cmd := exec.Command("git", "diff", hash)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return out.String(), nil
		}
		return "", fmt.Errorf("git diff %s: %v: %s", hash, err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

// DiffStat returns a one-line summary ("N files changed, +X -Y") between a
// checkpoint and the current working tree.
func DiffStat(hash string) (string, error) {
	cmd := exec.Command("git", "diff", "--shortstat", hash)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return strings.TrimSpace(out.String()), nil
		}
		return "", fmt.Errorf("git diff --shortstat %s: %v: %s", hash, err, strings.TrimSpace(errBuf.String()))
	}
	stat := strings.TrimSpace(out.String())
	if stat == "" {
		return "no changes", nil
	}
	return stat, nil
}

// CheckoutAll overwrites every tracked file in the working tree with the
// contents from the given checkpoint commit.
func CheckoutAll(hash string) error {
	_, err := run("checkout", hash, "--", ".")
	return err
}
