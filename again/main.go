// again: counts how many times you've had to repeat yourself to your AI
// agent this session.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Soldsoul86/AAA/again/internal/similar"
	"github.com/Soldsoul86/AAA/again/internal/transcript"
)

const usage = `again — counts how many times you've had to repeat yourself to your AI agent

Usage:
  again watch -file PATH [-threshold F] [-nudge N] [-interval MS]
  again watch -claude-code [-threshold F] [-nudge N] [-interval MS]

Flags:
  -file PATH      the session transcript to watch (JSONL, one message per line)
  -claude-code    best-effort: auto-locate the most recently modified
                  session file under ~/.claude/projects/<this-project>/
  -threshold F    similarity (0.0-1.0) above which two prompts count as "the same ask" (default 0.5)
  -nudge N        print a suggestion to restart the session after N repeats (default 3)
  -interval MS    how often to poll the file, in milliseconds (default 1000)

again compares each new prompt you send against your recent prompts using a
simple word-overlap score — not real language understanding, just a
transparent, explainable heuristic. See the README for what it can and
can't reliably catch.
`

func main() {
	if len(os.Args) < 2 || os.Args[1] != "watch" {
		fmt.Print(usage)
		os.Exit(2)
	}

	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	file := fs.String("file", "", "transcript file to watch")
	claudeCode := fs.Bool("claude-code", false, "auto-locate the current project's Claude Code session file")
	threshold := fs.Float64("threshold", 0.5, "similarity threshold")
	nudge := fs.Int("nudge", 3, "repeats before suggesting a restart")
	intervalMs := fs.Int("interval", 1000, "poll interval in milliseconds")
	fs.Parse(os.Args[2:])

	path := *file
	if *claudeCode {
		found, err := locateClaudeCodeSession()
		if err != nil {
			fmt.Fprintf(os.Stderr, "again: -claude-code: %v\n", err)
			os.Exit(1)
		}
		path = found
		fmt.Fprintf(os.Stderr, "again: watching %s\n", path)
	}
	if path == "" {
		fmt.Print(usage)
		os.Exit(2)
	}

	watch(path, *threshold, *nudge, time.Duration(*intervalMs)*time.Millisecond)
}

// locateClaudeCodeSession mirrors ctxmeter's auto-detection — same
// best-effort, unconfirmed path pattern, same caveat applies.
func locateClaudeCodeSession() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	encoded := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(home, ".claude", "projects", encoded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("no project directory found at %s (use -file <path> instead)", dir)
	}
	type candidate struct {
		path    string
		modTime time.Time
	}
	var found []candidate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		found = append(found, candidate{filepath.Join(dir, e.Name()), info.ModTime()})
	}
	if len(found) == 0 {
		return "", fmt.Errorf("no .jsonl session files found in %s", dir)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].modTime.After(found[j].modTime) })
	return found[0].path, nil
}

const historySize = 20

func watch(path string, threshold float64, nudgeAt int, interval time.Duration) {
	var offset int64
	var history []string
	repeats := 0

	fmt.Fprintln(os.Stderr, "again: waiting for prompts...")

	for {
		func() {
			f, err := os.Open(path)
			if err != nil {
				return
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				return
			}
			if info.Size() < offset {
				offset, history, repeats = 0, nil, 0
			}
			if info.Size() <= offset {
				return
			}
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				return
			}
			scanner := bufio.NewScanner(f)
			scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
			for scanner.Scan() {
				text, ok := transcript.ExtractUserText(scanner.Bytes())
				if !ok || strings.TrimSpace(text) == "" {
					continue
				}
				processPrompt(text, &history, &repeats, threshold, nudgeAt)
			}
			offset = info.Size()
		}()
		time.Sleep(interval)
	}
}

func processPrompt(text string, history *[]string, repeats *int, threshold float64, nudgeAt int) {
	best := 0.0
	bestAgo := 0
	for i, h := range *history {
		if s := similar.Jaccard(text, h); s > best {
			best = s
			bestAgo = len(*history) - i
		}
	}

	*history = append(*history, text)
	if len(*history) > historySize {
		*history = (*history)[len(*history)-historySize:]
	}

	if best >= threshold {
		*repeats++
		fmt.Fprintf(os.Stderr, "again: this looks similar (%.0f%%) to something you said %d prompt(s) ago — you've repeated yourself %d time(s) this session\n", best*100, bestAgo, *repeats)
		if *repeats == nudgeAt {
			fmt.Fprintln(os.Stderr, "again: that's a few repeats now — might be worth starting a fresh session instead of re-explaining again")
		}
	}
}
