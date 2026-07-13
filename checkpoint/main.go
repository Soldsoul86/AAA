// checkpoint: automatic, zero-config git checkpoints before your AI agent
// touches anything.
package main

import (
	"fmt"
	"os"

	"github.com/Soldsoul86/AAA/checkpoint/cmd"
)

const usage = `checkpoint — automatic git checkpoints for AI coding agents

Usage:
  checkpoint init [--user]        install the Claude Code auto-checkpoint hook
  checkpoint save [--label X]     take a manual snapshot now
  checkpoint list [-n N]          show recent checkpoints
  checkpoint diff [N]             diff checkpoint #N against the working tree (default: 1, most recent)
  checkpoint restore [N] [--yes]  restore the working tree to checkpoint #N (default: 1)
  checkpoint run -- <command>     wrap any agent CLI: snapshot before and after the session

Checkpoints are plain git commit objects (git stash create) stored in
.git/checkpoint/log.jsonl. No history is rewritten, no branches are
touched, nothing outside .git is modified. Delete that one file to
forget everything checkpoint has ever recorded.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = cmd.Init(os.Args[2:])
	case "save":
		err = cmd.Save(os.Args[2:])
	case "list":
		err = cmd.List(os.Args[2:])
	case "diff":
		err = cmd.Diff(os.Args[2:])
	case "restore":
		err = cmd.Restore(os.Args[2:])
	case "run":
		err = cmd.Run(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "checkpoint: unknown command %q\n\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
