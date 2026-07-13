// permit: diagnoses the documented reasons a Claude Code permission rule
// silently doesn't apply.
//
// It does not reimplement or compete with Claude Code's own permission
// engine, and it does not write anything — it only reads the real settings
// files and the real ~/.claude.json Claude Code itself reads, and reports
// what it finds against the documented failure modes.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/Soldsoul86/AAA/permit/internal/rules"
	"github.com/Soldsoul86/AAA/permit/internal/settings"
	"github.com/Soldsoul86/AAA/permit/internal/trust"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z" by
// goreleaser. Defaults to "dev" for local builds and is never set at all
// for "go install" builds — that path doesn't run our ldflags, only
// goreleaser's cross-compile step does. effectiveVersion() falls back to
// Go's own module version (embedded automatically by "go install") so
// --version is accurate either way, not just for goreleaser-built binaries.
var version = "dev"

func effectiveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}

const usage = `permit — diagnoses why a Claude Code permission rule silently doesn't apply

Usage:
  permit doctor [rule]   check the documented reasons a rule might not be working

permit checks two documented, common causes:
  - workspace trust not accepted (permissions.allow is read but not
    applied until you accept Claude Code's one-time trust dialog)
  - an existing deny/ask rule shadowing the one you're asking about
    (deny and ask always win regardless of specificity)

Example:
  permit doctor "Bash(git push origin main)"

Everything else here — writing rules, skipping the trust dialog — is
already a native Claude Code feature (the interactive "don't ask again"
flow, and hand-editing .claude/settings.json). permit doesn't duplicate
that; it only diagnoses.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "doctor":
		err = cmdDoctor(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	case "-version", "--version", "version":
		fmt.Println("permit", effectiveVersion())
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
