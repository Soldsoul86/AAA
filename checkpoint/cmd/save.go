package cmd

import (
	"flag"
	"fmt"

	"github.com/Soldsoul86/AAA/checkpoint/internal/git"
	"github.com/Soldsoul86/AAA/checkpoint/internal/store"
)

func Save(args []string) error {
	fs := flag.NewFlagSet("save", flag.ExitOnError)
	label := fs.String("label", "", "short label for this checkpoint")
	source := fs.String("source", "manual", "who triggered this checkpoint")
	quiet := fs.Bool("quiet", false, "suppress output when there is nothing new to checkpoint")
	fs.Parse(args)

	gitDir, err := requireRepo()
	if err != nil {
		return err
	}

	changed, err := git.HasChanges()
	if err != nil {
		return err
	}
	if !changed {
		if !*quiet {
			fmt.Println("checkpoint: nothing to snapshot (working tree matches last commit)")
		}
		return nil
	}

	hash, err := git.StashCreate(*label)
	if err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}
	if hash == "" {
		if !*quiet {
			fmt.Println("checkpoint: nothing to snapshot")
		}
		return nil
	}

	cp := store.Checkpoint{
		Time:   nowUTC(),
		Hash:   hash,
		Label:  *label,
		Source: *source,
	}
	if err := store.Append(gitDir, cp); err != nil {
		return fmt.Errorf("checkpoint: failed to record snapshot: %w", err)
	}

	if !*quiet {
		fmt.Printf("checkpoint: saved %s\n", hash[:12])
	}
	return nil
}
