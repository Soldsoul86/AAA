// permit: makes Claude Code permission rules easy to write correctly and
// easy to diagnose when they silently don't apply.
//
// It does not reimplement or compete with Claude Code's own permission
// engine — it writes rules using Claude Code's real syntax into the real
// settings files Claude Code reads, and checks the documented reasons a
// rule can silently fail to take effect.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Soldsoul86/AAA/permit/internal/rules"
	"github.com/Soldsoul86/AAA/permit/internal/settings"
	"github.com/Soldsoul86/AAA/permit/internal/trust"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z" by
// goreleaser. Defaults to "dev" for local builds.
var version = "dev"

const usage = `permit — makes Claude Code permission rules easy to write and diagnose

Usage:
  permit allow <rule> [--user]   add a permanent allow rule (project by default, --user for ~/.claude/settings.json)
  permit list                    show the merged allow/ask/deny rules across all settings sources
  permit doctor [rule]           check the documented reasons a rule might silently not apply

Rules use Claude Code's own syntax exactly, e.g.:
  permit allow "Bash(npm run *)"
  permit allow "Bash(git commit *)"
  permit allow "Read(./.env)"

permit never makes its own allow/deny decisions — it only writes rules into
the same settings files Claude Code reads, and helps you understand why an
existing rule isn't working.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "allow":
		err = cmdAllow(os.Args[2:])
	case "list":
		err = cmdList(os.Args[2:])
	case "doctor":
		err = cmdDoctor(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	case "-version", "--version", "version":
		fmt.Println("permit", version)
		return
	default:
		fmt.Fprintf(os.Stderr, "permit: unknown command %q\n\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "permit:", err)
		os.Exit(1)
	}
}

func projectRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	cwd, _ := os.Getwd()
	return cwd
}

func cmdAllow(args []string) error {
	user := false
	var rule string
	for _, a := range args {
		if a == "--user" {
			user = true
			continue
		}
		rule = a
	}
	if rule == "" {
		return fmt.Errorf("usage: permit allow <rule> [--user]")
	}
	parsed, err := rules.Parse(rule)
	if err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	var path string
	if user {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".claude", "settings.json")
	} else {
		path = filepath.Join(projectRoot(), ".claude", "settings.json")
	}

	added, err := settings.AddAllow(path, parsed.Raw)
	if err != nil {
		return err
	}
	if added {
		fmt.Printf("permit: added %q to %s\n", parsed.Raw, path)
	} else {
		fmt.Printf("permit: %q is already in %s\n", parsed.Raw, path)
	}
	return nil
}

func cmdList(args []string) error {
	for _, src := range settings.Sources(projectRoot()) {
		rs, err := settings.Read(src.Path)
		if err != nil {
			fmt.Printf("%s: could not read (%v)\n", src.Name, err)
			continue
		}
		total := len(rs.Allow) + len(rs.Ask) + len(rs.Deny)
		if total == 0 {
			continue
		}
		fmt.Printf("%s\n", src.Name)
		for _, r := range rs.Deny {
			fmt.Printf("  deny   %s\n", r)
		}
		for _, r := range rs.Ask {
			fmt.Printf("  ask    %s\n", r)
		}
		for _, r := range rs.Allow {
			fmt.Printf("  allow  %s\n", r)
		}
	}
	return nil
}

func cmdDoctor(args []string) error {
	root := projectRoot()
	fmt.Printf("checking %s\n\n", root)

	status, err := trust.Check(root)
	if err != nil {
		fmt.Printf("⚠️  could not check workspace trust: %v\n", err)
	} else if !status.Checked {
		fmt.Println("⚠️  no trust record found for this project yet — if allow rules aren't applying, open Claude Code here interactively at least once and accept the trust dialog if shown")
	} else if !status.Trusted {
		fmt.Println("🔴 this workspace has NOT accepted the trust dialog — per Claude Code's docs, permissions.allow rules are read but NOT applied until you do. This is very likely why an allow rule isn't working.")
	} else {
		fmt.Println("✅ workspace trust: accepted")
	}
	fmt.Println()

	var allDeny, allAsk []rules.Rule
	anyInvalid := false
	for _, src := range settings.Sources(root) {
		rs, err := settings.Read(src.Path)
		if err != nil {
			fmt.Printf("🔴 %s: invalid JSON (%v)\n", src.Name, err)
			anyInvalid = true
			continue
		}
		for _, raw := range rs.Deny {
			if r, err := rules.Parse(raw); err == nil {
				allDeny = append(allDeny, r)
			}
		}
		for _, raw := range rs.Ask {
			if r, err := rules.Parse(raw); err == nil {
				allAsk = append(allAsk, r)
			}
		}
	}
	if !anyInvalid {
		fmt.Println("✅ all settings files are valid JSON")
	}

	if len(args) == 0 {
		return nil
	}
	candidate, err := rules.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid rule to check: %w", err)
	}
	fmt.Printf("\nchecking whether %q could be shadowed by an existing deny/ask rule:\n", candidate.Raw)
	found := false
	for _, d := range allDeny {
		if shadows(d, candidate) {
			fmt.Printf("🔴 possibly shadowed by deny rule %q — deny always wins regardless of specificity\n", d.Raw)
			found = true
		}
	}
	for _, a := range allAsk {
		if shadows(a, candidate) {
			fmt.Printf("🟡 possibly shadowed by ask rule %q — ask always forces a prompt even if an allow rule also matches\n", a.Raw)
			found = true
		}
	}
	if !found {
		fmt.Println("✅ no obvious shadowing deny/ask rule found (best-effort check — see README for what this can and can't catch)")
	}
	return nil
}

// shadows is a deliberately conservative, best-effort heuristic — not a
// full reimplementation of Claude Code's real matching engine, which
// includes compound-command splitting, wrapper stripping, and more. It
// catches the two simplest, most common real cases: a bare tool-name rule,
// and one specifier being a literal prefix of the other.
func shadows(existing, candidate rules.Rule) bool {
	if existing.Tool != candidate.Tool {
		return false
	}
	if existing.Specifier == "" {
		return true // bare tool name matches everything for that tool
	}
	if existing.Specifier == candidate.Specifier {
		return true
	}
	trimmed := strings.TrimSuffix(strings.TrimSuffix(existing.Specifier, "*"), " ")
	return trimmed != "" && strings.HasPrefix(candidate.Specifier, trimmed)
}
