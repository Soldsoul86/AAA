// ctxmeter: a live context-window gauge for any AI coding CLI.
//
// It either tails a log/transcript file, or reads stdin in a pipeline
// (passing everything through unmodified, so it never swallows the output
// of whatever it's watching), estimating cumulative token usage as it goes.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Soldsoul86/AAA/ctxmeter/internal/estimate"
)

const usage = `ctxmeter — a live context-window gauge for any AI coding CLI

Usage:
  ctxmeter watch -file PATH [-max N] [-interval MS]
  ctxmeter watch -claude-code [-max N] [-interval MS]
  <some-cli> | ctxmeter watch [-max N]

Flags:
  -file PATH     tail this file and estimate cumulative content size
  -claude-code   best-effort: auto-locate the most recently modified
                 session file under ~/.claude/projects/<this-project>/
  -max N         context window size to gauge against (default 200000)
  -interval MS   how often to poll a watched file, in milliseconds (default 500)

ctxmeter estimates tokens as roughly (character count of text content) / 4 —
the standard rough approximation, not a real tokenizer for any specific
model. Treat the number as "roughly how full", not an exact count.
`

func main() {
	if len(os.Args) < 2 || os.Args[1] != "watch" {
		fmt.Print(usage)
		os.Exit(2)
	}

	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	file := fs.String("file", "", "file to tail")
	claudeCode := fs.Bool("claude-code", false, "auto-locate the current project's Claude Code session file")
	max := fs.Int("max", 200000, "context window size to gauge against")
	intervalMs := fs.Int("interval", 500, "poll interval in milliseconds")
	fs.Parse(os.Args[2:])

	path := *file
	if *claudeCode {
		found, err := locateClaudeCodeSession()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ctxmeter: -claude-code: %v\n", err)
			fmt.Fprintln(os.Stderr, "ctxmeter: falling back to reading stdin instead")
		} else {
			path = found
			fmt.Fprintf(os.Stderr, "ctxmeter: watching %s\n", path)
		}
	}

	if path == "" {
		watchStdin(*max)
		fmt.Fprintln(os.Stderr)
		return
	}
	watchFile(path, *max, time.Duration(*intervalMs)*time.Millisecond)
}

// locateClaudeCodeSession scans every project directory under
// ~/.claude/projects/ and returns the most recently modified .jsonl file
// across all of them.
//
// This used to encode the current working directory (cwd with / replaced
// by -) and look up a single matching project folder. That was wrong:
// confirmed directly against a real session where the process's cwd (a
// subdirectory the agent had cd'd into) didn't match the directory Claude
// Code actually keyed the session under (the directory the session was
// launched from). Scanning for "whichever session is actively being
// written to right now" sidesteps the encoding question entirely — same
// fix applied to actually's auto-detection, which was built this way from
// the start.
func locateClaudeCodeSession() (string, error) {
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
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
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

func watchFile(path string, max int, interval time.Duration) {
	var total int
	var offset int64

	for {
		func() {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nctxmeter: %v\n", err)
				return
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				return
			}
			if info.Size() < offset {
				// file was truncated or replaced — start over
				offset = 0
				total = 0
			}
			if info.Size() > offset {
				if _, err := f.Seek(offset, io.SeekStart); err != nil {
					return
				}
				scanner := bufio.NewScanner(f)
				scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
				for scanner.Scan() {
					total += estimateLine(scanner.Bytes())
				}
				offset = info.Size()
			}
		}()
		renderGauge(total, max)
		time.Sleep(interval)
	}
}

func watchStdin(max int) {
	var total int
	buf := make([]byte, 4096)
	var lineBuf bytes.Buffer

	for {
		n, readErr := os.Stdin.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			os.Stdout.Write(chunk) // pass through unmodified — never swallows the piped output
			lineBuf.Write(chunk)
			for {
				line, err := lineBuf.ReadString('\n')
				if err != nil {
					lineBuf.Reset()
					lineBuf.WriteString(line)
					break
				}
				total += estimateLine([]byte(line))
				renderGauge(total, max)
			}
		}
		if readErr != nil {
			break
		}
	}
}

func estimateLine(line []byte) int {
	if n, ok := estimate.JSONContentLength(line); ok {
		return n / 4
	}
	return estimate.Tokens(string(line))
}

func renderGauge(total, max int) {
	pct := float64(total) / float64(max) * 100
	if pct > 100 {
		pct = 100
	}
	const barWidth = 20
	filled := int(pct / 100 * float64(barWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Fprintf(os.Stderr, "\r[%s] %3.0f%% (~%s / %s est. tokens)   ", bar, pct, formatNum(total), formatNum(max))
}

func formatNum(n int) string {
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
