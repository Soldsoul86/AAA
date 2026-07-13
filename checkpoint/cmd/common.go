package cmd

import (
	"fmt"
	"time"

	"github.com/Soldsoul86/AAA/checkpoint/internal/git"
)

func requireRepo() (gitDir string, err error) {
	if !git.IsInsideWorkTree() {
		return "", fmt.Errorf("checkpoint: not inside a git repository")
	}
	return git.GitDir()
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
