// loopkill: detects when your AI agent is stuck in a loop and interrupts it
// before it burns your quota.
//
// It wraps any command, passes its stdout straight through to your terminal
// byte-for-byte (so the agent's own UI is never touched), and separately
// watches a normalized copy of the output for lines that keep repeating.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/Soldsoul86/AAA/loopkill/internal/detector"
)

const usage = `loopkill — detects when your AI agent is stuck in a loop

Usage:
  loopkill [flags] -- <command> [args...]

Flags:
  -threshold N   how many times a line must repeat within the window to count as a loop (default 3)
  -window N      how many recent distinct lines to consider (default 50)
  -kill          send SIGINT to the wrapped process when a loop is detected (default: warn only)
  -quiet         suppress the startup banner

Example:
  loopkill -- claude
  loopkill -kill -- cursor-agent
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	dashIdx := -1
	for i, a := range args {
		if a == "--" {
			dashIdx = i
			break
		}
	}
	var flagArgs, cmdArgs []string
	if dashIdx >= 0 {
		flagArgs = args[:dashIdx]
		cmdArgs = args[dashIdx+1:]
	} else {
		flagArgs = args
	}

	fs := flag.NewFlagSet("loopkill", flag.ExitOnError)
	threshold := fs.Int("threshold", 3, "")
	window := fs.Int("window", 50, "")
	kill := fs.Bool("kill", false, "")
	quiet := fs.Bool("quiet", false, "")
	fs.Usage = func() { fmt.Print(usage) }
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if len(cmdArgs) == 0 {
		fmt.Print(usage)
		os.Exit(2)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "loopkill: watching %q (threshold=%d, window=%d)\n", cmdArgs[0], *threshold, *window)
	}

	c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	if err := c.Start(); err != nil {
		return err
	}

	det := detector.New(*threshold, *window)
	readBuf := make([]byte, 4096)
	var lineBuf bytes.Buffer

	for {
		n, readErr := stdout.Read(readBuf)
		if n > 0 {
			chunk := readBuf[:n]
			os.Stdout.Write(chunk) // pass through immediately, unmodified — never alters the agent's UI
			lineBuf.Write(chunk)
			for {
				line, err := lineBuf.ReadString('\n')
				if err != nil {
					// incomplete line (EOF from the buffer, not the process) — the
					// buffer has already drained itself of `line`; put it back and
					// wait for more bytes to complete it.
					lineBuf.Reset()
					lineBuf.WriteString(line)
					break
				}
				if matched, count, hit := det.Feed(line); hit {
					fmt.Fprintf(os.Stderr, "\a\n⚠️  loopkill: this line has repeated %d times in the last %d lines — looks stuck:\n    %q\n", count, *window, matched)
					if *kill {
						fmt.Fprintln(os.Stderr, "loopkill: sending SIGINT to interrupt it")
						_ = c.Process.Signal(syscall.SIGINT)
					}
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	waitErr := c.Wait()
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return waitErr
	}
	return nil
}
