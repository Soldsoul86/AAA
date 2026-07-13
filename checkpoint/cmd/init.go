package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Soldsoul86/AAA/checkpoint/internal/git"
	"github.com/Soldsoul86/AAA/checkpoint/internal/hooks"
)

func Init(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	user := fs.Bool("user", false, "install the hook in ~/.claude/settings.json instead of the project's .claude/settings.json")
	fs.Parse(args)

	if !git.IsInsideWorkTree() {
		return fmt.Errorf("checkpoint init: not inside a git repository")
	}
	root, err := git.RepoRoot()
	if err != nil {
		return err
	}

	var settingsPath string
	if *user {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		settingsPath = filepath.Join(home, ".claude", "settings.json")
	} else {
		settingsPath = filepath.Join(root, ".claude", "settings.json")
	}

	changed, err := hooks.InstallClaudeCode(settingsPath)
	if err != nil {
		return fmt.Errorf("checkpoint init: %w", err)
	}

	if changed {
		fmt.Printf("checkpoint: installed a PreToolUse hook in %s\n", settingsPath)
		fmt.Println("checkpoint: Claude Code will ask you to approve the new hook the next time you start it — that's expected.")
	} else {
		fmt.Printf("checkpoint: hook already installed in %s\n", settingsPath)
	}

	fmt.Println()
	fmt.Println("For tools without a hook system (Cursor, Codex, Aider, Gemini CLI), wrap the session instead:")
	fmt.Println("  checkpoint run -- cursor-agent")
	fmt.Println()
	fmt.Println("Try it now:")
	fmt.Println("  checkpoint save      # take a manual snapshot")
	fmt.Println("  checkpoint list      # see your checkpoints")
	fmt.Println("  checkpoint diff      # see what changed since the last one")
	fmt.Println("  checkpoint restore   # roll back to one")
	return nil
}
