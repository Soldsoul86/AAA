package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Soldsoul86/AAA/checkpoint/internal/git"
	"github.com/Soldsoul86/AAA/checkpoint/internal/store"
)

// Run wraps an arbitrary agent CLI for tools that don't support hooks yet:
// snapshot before it starts, snapshot after it exits, regardless of how the
// session went.
func Run(args []string) error {
	dashIdx := -1
	for i, a := range args {
		if a == "--" {
			dashIdx = i
			break
		}
	}
	var cmdArgs []string
	if dashIdx >= 0 {
		cmdArgs = args[dashIdx+1:]
	} else {
		cmdArgs = args
	}
	if len(cmdArgs) == 0 {
		return fmt.Errorf("usage: checkpoint run -- <command> [args...]")
	}

	gitDir, err := requireRepo()
	if err != nil {
		return err
	}

	snapshot := func(label, source string) {
		changed, err := git.HasChanges()
		if err != nil || !changed {
			return
		}
		hash, err := git.StashCreate(label)
		if err != nil || hash == "" {
			return
		}
		_ = store.Append(gitDir, store.Checkpoint{
			Time:   nowUTC(),
			Hash:   hash,
			Label:  label,
			Source: source,
		})
	}

	snapshot("session-start", "run-start")

	c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	runErr := c.Run()

	snapshot("session-end", "run-end")

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return runErr
	}
	return nil
}
