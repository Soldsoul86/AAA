package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/Soldsoul86/AAA/checkpoint/internal/git"
	"github.com/Soldsoul86/AAA/checkpoint/internal/store"
)

func Restore(args []string) error {
	yes := false
	var rest []string
	for _, a := range args {
		if a == "--yes" || a == "-yes" || a == "-y" {
			yes = true
			continue
		}
		rest = append(rest, a)
	}

	n := 1
	if len(rest) > 0 {
		v, err := strconv.Atoi(rest[0])
		if err != nil {
			return fmt.Errorf("checkpoint restore: expected a checkpoint number, got %q", rest[0])
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

	stat, err := git.DiffStat(cp.Hash)
	if err == nil {
		fmt.Printf("checkpoint: restoring #%d will overwrite tracked files to match that checkpoint (%s)\n", n, stat)
	}

	if !yes {
		fmt.Print("Continue? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if line != "y\n" && line != "Y\n" {
			fmt.Println("checkpoint: aborted")
			return nil
		}
	}

	// Save the current state first so restore is itself always undoable.
	if changed, _ := git.HasChanges(); changed {
		if hash, err := git.StashCreate("before-restore"); err == nil && hash != "" {
			_ = store.Append(gitDir, store.Checkpoint{
				Time:   nowUTC(),
				Hash:   hash,
				Label:  "before-restore",
				Source: "pre-restore",
			})
			fmt.Printf("checkpoint: current state saved as a new checkpoint first (%s)\n", hash[:12])
		}
	}

	if err := git.CheckoutAll(cp.Hash); err != nil {
		return fmt.Errorf("checkpoint: restore failed: %w", err)
	}
	fmt.Printf("checkpoint: restored to #%d (%s)\n", n, cp.Hash[:12])
	return nil
}
