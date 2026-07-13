// Package store persists the checkpoint log as newline-delimited JSON inside
// .git/checkpoint/log.jsonl — colocated with the repo, never committed
// (nothing outside .git is touched), and trivially inspectable with `cat`.
package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Checkpoint struct {
	Time   time.Time `json:"time"`
	Hash   string    `json:"hash"`
	Label  string    `json:"label"`
	Source string    `json:"source"` // "manual", "hook", "run-start", "run-end", "pre-restore"
}

func logPath(gitDir string) string {
	return filepath.Join(gitDir, "checkpoint", "log.jsonl")
}

// Append adds a checkpoint to the log, creating the directory on first use.
func Append(gitDir string, cp Checkpoint) error {
	dir := filepath.Join(gitDir, "checkpoint")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(logPath(gitDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// List returns all checkpoints, most recent first.
func List(gitDir string) ([]Checkpoint, error) {
	f, err := os.Open(logPath(gitDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cps []Checkpoint
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var cp Checkpoint
		if err := json.Unmarshal(line, &cp); err != nil {
			continue // skip malformed lines rather than fail the whole read
		}
		cps = append(cps, cp)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// reverse in place: most recent first
	for i, j := 0, len(cps)-1; i < j; i, j = i+1, j-1 {
		cps[i], cps[j] = cps[j], cps[i]
	}
	return cps, nil
}

// ByIndex returns the nth most recent checkpoint, 1-based (1 = most recent).
func ByIndex(gitDir string, n int) (*Checkpoint, error) {
	cps, err := List(gitDir)
	if err != nil {
		return nil, err
	}
	if n < 1 || n > len(cps) {
		return nil, fmt.Errorf("no checkpoint #%d (have %d)", n, len(cps))
	}
	return &cps[n-1], nil
}
