// exists: checks whether a package your AI agent just told it to install
// actually exists in the real registry, before you find out the hard way
// — either the build breaks, or worse, something with that exact
// hallucinated name has been slopsquatted and actually resolves.
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

	"github.com/Soldsoul86/AAA/exists/internal/transcript"
	"github.com/Soldsoul86/AAA/exists/internal/watch"
)

const usage = `exists — checks whether a package your agent just installed is real

Usage:
  exists watch -file PATH [-interval MS]
  exists watch -claude-code [-interval MS]

Flags:
  -file PATH      the session transcript to watch (JSONL, one message per line)
  -claude-code    auto-locate the most recently modified session file across
                  every project under ~/.claude/projects/
  -interval MS    how often to poll the file, in milliseconds (default 1000)

v1 checks npm and PyPI installs (npm/yarn/pnpm install, pip/pip3/poetry/uv
add) against the real registry over the network. Every check is a live
HTTP call — this is the only tool in this collection that needs network
access. See the README for exactly what it recognizes and what it doesn't.
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
			fmt.Fprintf(os.Stderr, "exists: -claude-code: %v\n", err)
			os.Exit(1)
		}
		path = found
		fmt.Fprintf(os.Stderr, "exists: watching %s\n", path)
	}
	if path == "" {
		fmt.Print(usage)
		os.Exit(2)
	}

	watchFile(path, time.Duration(*intervalMs)*time.Millisecond)
}

// locateMostRecentSession scans every project directory under
// ~/.claude/projects/ and returns the most recently modified .jsonl file
// across all of them — same approach actually uses, chosen for the same
// reason: encoding the current working directory to guess a single project
// folder was confirmed unreliable in that tool.
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
// silently before saying something — see the identical comment in
// actually's main.go, where this exact silent-hang behavior was confirmed
// directly against a typo'd path before this fix.
const missingFileWarnAfter = 5

func watchFile(path string, interval time.Duration) {
	var offset int64
	consecutiveMisses := 0
	warned := false

	fmt.Fprintln(os.Stderr, "exists: waiting for activity...")

	for {
		func() {
			f, err := os.Open(path)
			if err != nil {
				consecutiveMisses++
				if consecutiveMisses == missingFileWarnAfter && !warned {
					fmt.Fprintf(os.Stderr, "exists: still can't open %s (%v) — if you expected it to exist, check the path\n", path, err)
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
			}
			if info.Size() <= offset {
				return
			}
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				return
			}
			// bufio.Scanner has a hard per-line buffer cap; a transcript
			// line built from a large tool result can exceed it. Confirmed
			// directly on actually (same watch-loop shape): hitting that
			// cap made the scanner fail silently and skip every remaining
			// line in the file, permanently, with zero error output.
			// bufio.Reader.ReadString grows to fit an arbitrarily long
			// line instead, and lets offset track exactly how many bytes
			// were actually consumed, so a partial line still being
			// written is correctly left for the next poll rather than
			// processed early or dropped.
			reader := bufio.NewReader(f)
			for {
				line, err := reader.ReadString('\n')
				if strings.HasSuffix(line, "\n") {
					for _, cmd := range transcript.BashCommands([]byte(strings.TrimRight(line, "\n"))) {
						for _, finding := range watch.Check(cmd) {
							report(finding)
						}
					}
					offset += int64(len(line))
				}
				if err != nil {
					break
				}
			}
		}()
		time.Sleep(interval)
	}
}

func report(f watch.Finding) {
	switch f.Status {
	case watch.Missing:
		fmt.Fprintf(os.Stderr, "exists: %q was just installed from %s, but no package by that name exists in the real registry — check for a typo or a hallucinated name before trusting this\n", f.Package, f.Ecosystem)
	case watch.Unverifiable:
		fmt.Fprintf(os.Stderr, "exists: couldn't verify %q against %s (network error or timeout) — not claiming it's missing, just couldn't check\n", f.Package, f.Ecosystem)
	}
}
