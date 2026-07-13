// actually: cross-checks what your AI agent claims it did against what
// actually happened, starting with the single most common instance of that
// gap — a "tests pass" claim that doesn't match the real last test run.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Soldsoul86/AAA/actually/internal/transcript"
	"github.com/Soldsoul86/AAA/actually/internal/verify"
)

const usage = `actually — cross-checks agent claims against what actually happened

Usage:
  actually watch -file PATH [-interval MS]
  actually watch -claude-code [-interval MS]

Flags:
  -file PATH      the session transcript to watch (JSONL, one message per line)
  -claude-code    auto-locate the most recently modified session file across
                  every project under ~/.claude/projects/
  -interval MS    how often to poll the file, in milliseconds (default 1000)

v1 scope is deliberately narrow: it catches an assistant claiming "tests
pass" when the last real test run in the same session actually failed, was
never run, or was run before a later file edit (making it stale). Broader
claim-vs-diff checking ("I added error handling to X" — did the diff touch
X?) is a real next step, not built yet. See the README for exactly how the
pass/fail detection works and what it can't catch.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "watch":
		cmdWatch(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Print(usage)
		os.Exit(2)
	}
}

func cmdWatch(args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	file := fs.String("file", "", "transcript file to watch")
	claudeCode := fs.Bool("claude-code", false, "auto-locate the most recently active Claude Code session file")
	intervalMs := fs.Int("interval", 1000, "poll interval in milliseconds")
	fs.Parse(args)

	path := *file
	if *claudeCode {
		found, err := locateMostRecentSession()
		if err != nil {
			fmt.Fprintf(os.Stderr, "actually: -claude-code: %v\n", err)
			os.Exit(1)
		}
		path = found
		fmt.Fprintf(os.Stderr, "actually: watching %s\n", path)
	}
	if path == "" {
		fmt.Print(usage)
		os.Exit(2)
	}

	watch(path, time.Duration(*intervalMs)*time.Millisecond)
}

// locateMostRecentSession scans every project directory under
// ~/.claude/projects/ and returns the most recently modified .jsonl file
// across all of them.
//
// This is deliberately not "encode the current working directory and look
// up its matching project folder" — that approach was tried in again and
// ctxmeter first, and turned out to be unreliable: verified directly
// against a real setup where the process's cwd (a subdirectory the agent
// had cd'd into) didn't match the directory Claude Code actually keyed the
// session under (the directory the session was launched from). Scanning
// for "whichever session file is actively being written to right now" side
// steps the encoding question entirely.
func locateMostRecentSession() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".claude", "projects")

	projectDirs, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("no %s directory found (use -file <path> instead)", root)
	}

	type candidate struct {
		path    string
		modTime time.Time
	}
	var found []candidate
	for _, pd := range projectDirs {
		if !pd.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(root, pd.Name()))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			found = append(found, candidate{filepath.Join(root, pd.Name(), e.Name()), info.ModTime()})
		}
	}
	if len(found) == 0 {
		return "", fmt.Errorf("no .jsonl session files found under %s (use -file <path> instead)", root)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].modTime.After(found[j].modTime) })
	return found[0].path, nil
}

// missingFileWarnAfter is how many consecutive failed opens to tolerate
// silently before saying something. A file that doesn't exist yet is a
// normal, transient state (e.g. -claude-code racing a session that hasn't
// written its first line), but a file that never appears is very likely a
// typo'd path — and going forever with zero output looks identical to
// "working correctly, nothing to report" from a first-time user's seat.
// Confirmed directly: watching a nonexistent path printed only the initial
// "waiting for activity..." message forever, with no further signal.
const missingFileWarnAfter = 5

func watch(path string, interval time.Duration) {
	var offset int64
	state := verify.NewState()
	consecutiveMisses := 0
	warned := false

	fmt.Fprintln(os.Stderr, "actually: waiting for activity...")

	for {
		func() {
			f, err := os.Open(path)
			if err != nil {
				consecutiveMisses++
				if consecutiveMisses == missingFileWarnAfter && !warned {
					fmt.Fprintf(os.Stderr, "actually: still can't open %s (%v) — if you expected it to exist, check the path\n", path, err)
					warned = true
				}
				return
			}
			consecutiveMisses = 0
			warned = false
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				return
			}
			if info.Size() < offset {
				offset = 0
				state = verify.NewState()
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
				for _, ev := range transcript.ParseLine(scanner.Bytes()) {
					if m := state.Feed(ev); m != nil {
						fmt.Fprintf(os.Stderr, "actually: %s\n", m.Message)
					}
				}
			}
			offset = info.Size()
		}()
		time.Sleep(interval)
	}
}
