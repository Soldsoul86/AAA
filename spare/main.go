// spare: makes destructive delete commands recoverable. Diverts rm (and,
// via a PowerShell profile function, Remove-Item/del on Windows) to a local
// trash instead of deleting for real, regardless of which AI agent — or
// human — issued the command.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Soldsoul86/AAA/spare/internal/shim"
	"github.com/Soldsoul86/AAA/spare/internal/trash"
)

const usage = `spare — makes rm (and Remove-Item/del on Windows) recoverable

Usage:
  spare init              install the shim (adds spare's rm to PATH / your
                           PowerShell profile) — takes effect in new sessions
  spare status             show whether the shim is installed
  spare disable            remove everything spare init added
  spare rm [-rf] paths...  the actual interception target — you normally
                           never type this yourself, "rm" resolves to it
  spare list               show what's currently in the trash
  spare restore <id>       move a trashed item back to where it came from
  spare purge [DAYS]       permanently delete trash older than DAYS
                           (default: 30)

After "spare init", start a new terminal / new agent session — the shim
takes effect for new sessions, not the one currently running.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	// Multi-call dispatch: invoked as "rm" via the shim symlink, behave
	// exactly like "spare rm ...". Also reachable explicitly as
	// "spare rm ..." (which is what the Windows PowerShell function calls).
	if filepath.Base(os.Args[0]) == "rm" || filepath.Base(os.Args[0]) == "rm.exe" {
		cmdRM(os.Args[1:])
		return
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "status":
		cmdStatus()
	case "disable":
		cmdDisable()
	case "rm":
		cmdRM(os.Args[2:])
	case "list":
		cmdList()
	case "restore":
		cmdRestore(os.Args[2:])
	case "purge":
		cmdPurge(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Print(usage)
		os.Exit(2)
	}
}

func cmdInit() {
	touched, err := shim.Install()
	if err != nil {
		fmt.Fprintln(os.Stderr, "spare: init failed:", err)
		os.Exit(1)
	}
	fmt.Println("spare: installed. Updated:")
	for _, f := range touched {
		fmt.Println("  -", f)
	}
	fmt.Println()
	fmt.Println("Start a new terminal (or new agent session) for this to take effect —")
	fmt.Println("it doesn't apply retroactively to the shell you ran this in.")
}

func cmdStatus() {
	s, err := shim.CurrentStatus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "spare: status failed:", err)
		os.Exit(1)
	}
	if !s.RMLinkExists && len(s.RCFilesTouched) == 0 {
		fmt.Println("spare: not installed. Run `spare init`.")
		return
	}
	fmt.Printf("spare: shim directory: %s (rm link present: %v)\n", s.ShimDir, s.RMLinkExists)
	if len(s.RCFilesTouched) == 0 {
		fmt.Println("spare: no shell startup files are configured yet — run `spare init`.")
		return
	}
	fmt.Println("spare: active in:")
	for _, f := range s.RCFilesTouched {
		fmt.Println("  -", f)
	}
}

func cmdDisable() {
	if err := shim.Uninstall(); err != nil {
		fmt.Fprintln(os.Stderr, "spare: disable failed:", err)
		os.Exit(1)
	}
	fmt.Println("spare: removed. `rm` reverts to the real thing in new sessions.")
}

type rmOptions struct {
	recursive bool
	force     bool
	verbose   bool
	paths     []string
}

// parseRmArgs is intentionally permissive about flags it doesn't recognize
// (accepted and ignored rather than a hard error) — the goal is to never be
// the reason a script or agent command that worked with real rm suddenly
// fails, only to add the safety net underneath it.
func parseRmArgs(args []string) rmOptions {
	var opts rmOptions
	endOfFlags := false
	for _, a := range args {
		if endOfFlags || a == "-" {
			opts.paths = append(opts.paths, a)
			continue
		}
		if a == "--" {
			endOfFlags = true
			continue
		}
		if strings.HasPrefix(a, "--") {
			switch a {
			case "--recursive":
				opts.recursive = true
			case "--force":
				opts.force = true
			case "--verbose":
				opts.verbose = true
			}
			continue
		}
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			for _, c := range a[1:] {
				switch c {
				case 'r', 'R':
					opts.recursive = true
				case 'f':
					opts.force = true
				case 'v':
					opts.verbose = true
				}
			}
			continue
		}
		opts.paths = append(opts.paths, a)
	}
	return opts
}

func cmdRM(args []string) {
	opts := parseRmArgs(args)
	if len(opts.paths) == 0 {
		if opts.force {
			return // "rm -f" with nothing to remove is a silent success, matches real rm
		}
		fmt.Fprintln(os.Stderr, "usage: rm [-r] [-f] [-v] file ...")
		os.Exit(1)
	}

	exitCode := 0
	moved := 0
	for _, p := range opts.paths {
		info, err := os.Lstat(p)
		if err != nil {
			if opts.force {
				continue
			}
			fmt.Fprintf(os.Stderr, "rm: %s: No such file or directory\n", p)
			exitCode = 1
			continue
		}
		if info.IsDir() && !opts.recursive {
			fmt.Fprintf(os.Stderr, "rm: %s: is a directory\n", p)
			exitCode = 1
			continue
		}
		if _, err := trash.Move(p); err != nil {
			fmt.Fprintf(os.Stderr, "rm: %s: %v\n", p, err)
			exitCode = 1
			continue
		}
		moved++
		if opts.verbose {
			fmt.Fprintf(os.Stderr, "removed %q\n", p)
		}
	}

	if moved > 0 && os.Getenv("SPARE_QUIET") == "" {
		fmt.Fprintf(os.Stderr, "spare: moved %d item(s) to trash — `spare list` to view, `spare restore <id>` to undo\n", moved)
	}
	os.Exit(exitCode)
}

func cmdList() {
	entries, err := trash.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "spare:", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Println("spare: trash is empty")
		return
	}
	fmt.Printf("%-24s %-8s %-10s %s\n", "ID", "TYPE", "AGE", "ORIGINAL PATH")
	for _, e := range entries {
		kind := "file"
		if e.IsDir {
			kind = "dir"
		}
		fmt.Printf("%-24s %-8s %-10s %s\n", e.ID, kind, humanAge(e.DeletedAt), e.OriginalPath)
	}
}

func humanAge(t time.Time) string {
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

func cmdRestore(args []string) {
	force := false
	var id string
	for _, a := range args {
		if a == "--force" || a == "-f" {
			force = true
			continue
		}
		id = a
	}
	if id == "" {
		fmt.Fprintln(os.Stderr, "usage: spare restore <id> [--force]")
		os.Exit(2)
	}
	e, err := trash.Restore(id, force)
	if err == trash.ErrTargetExists {
		fmt.Fprintf(os.Stderr, "spare: %s already exists — use --force to overwrite it\n", e.OriginalPath)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "spare:", err)
		os.Exit(1)
	}
	fmt.Printf("spare: restored %s\n", e.OriginalPath)
}

func cmdPurge(args []string) {
	days := 30
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			days = n
		}
	}
	n, err := trash.Purge(time.Duration(days) * 24 * time.Hour)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spare:", err)
		os.Exit(1)
	}
	fmt.Printf("spare: purged %d item(s) older than %d days\n", n, days)
}
