package cmd

import (
	"fmt"
	"strconv"

	"github.com/Soldsoul86/AAA/checkpoint/internal/git"
	"github.com/Soldsoul86/AAA/checkpoint/internal/store"
)

func Diff(args []string) error {
	n := 1
	if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("checkpoint diff: expected a checkpoint number, got %q", args[0])
		}
		n = v
	}

	gitDir, err := requireRepo()
	if err != nil {
		return err
	}

	cp, err := store.ByIndex(gitDir, n)
	if err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}

	diff, err := git.Diff(cp.Hash)
	if err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}
	if diff == "" {
		fmt.Println("checkpoint: no difference between that checkpoint and the current working tree")
		return nil
	}
	fmt.Print(diff)
	return nil
}
