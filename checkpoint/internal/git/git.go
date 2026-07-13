// Package git wraps the small set of git plumbing commands checkpoint needs.
// Checkpoints are plain git commit objects created with `git stash create` —
// there is no custom storage format. Anyone can inspect or restore a
// checkpoint with plain git if this tool disappears tomorrow.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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

// StashCreate snapshots the current index + working tree (tracked files only)
// into a commit object without touching HEAD, the index, the working tree,
// or the stash ref. Returns the empty string if there is nothing to snapshot.
func StashCreate(label string) (string, error) {
	args := []string{"stash", "create"}
	if label != "" {
		args = append(args, label)
	}
	hash, err := run(args...)
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
