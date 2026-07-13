package cmd

import (
	"flag"
	"fmt"
	"time"

	"github.com/Soldsoul86/AAA/checkpoint/internal/git"
	"github.com/Soldsoul86/AAA/checkpoint/internal/store"
)

func List(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	n := fs.Int("n", 20, "max checkpoints to show")
	fs.Parse(args)

	gitDir, err := requireRepo()
	if err != nil {
		return err
	}

	cps, err := store.List(gitDir)
	if err != nil {
		return err
	}
	if len(cps) == 0 {
		fmt.Println("checkpoint: no checkpoints yet. Run 'checkpoint save' to create one.")
		return nil
	}
	if len(cps) > *n {
		cps = cps[:*n]
	}

	fmt.Printf("%-4s %-9s %-8s %-14s %s\n", "#", "AGE", "HASH", "SOURCE", "SUMMARY")
	for i, cp := range cps {
		stat, err := git.DiffStat(cp.Hash)
		if err != nil {
			stat = "(unreadable — checkpoint may be gone)"
		}
		label := cp.Label
		if label != "" {
			stat = label + " — " + stat
		}
		fmt.Printf("%-4d %-9s %-8s %-14s %s\n", i+1, ago(cp.Time), cp.Hash[:8], cp.Source, stat)
	}
	return nil
}

func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
