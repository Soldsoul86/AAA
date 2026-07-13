// Package trust checks Claude Code's workspace trust state — whether a
// project's permissions.allow rules are actually being applied at all.
//
// Per code.claude.com/docs/en/permissions: "permissions.allow rules ...
// grant capability, so Claude Code applies them only after you accept the
// workspace trust dialog for that workspace. Until then, Claude Code reads
// the rules but doesn't apply them." This is a documented, common reason
// an allow rule silently does nothing.
//
// The storage location (~/.claude.json, keyed by absolute project path,
// under projects[path].hasTrustDialogAccepted) is not documented publicly
// as a stable API — it was found by directly inspecting a real Claude Code
// config file. It may change without notice; if this stops finding
// anything, that's the most likely reason, not a sign the project is
// actually untrusted.
package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Status struct {
	Checked bool // whether we found an entry for this project at all
	Trusted bool
}

// Check looks up workspace trust status for a project directory (normally
// its git root) against the real ~/.claude.json.
func Check(projectDir string) (Status, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Status{}, err
	}
	return CheckAt(projectDir, filepath.Join(home, ".claude.json"))
}

// CheckAt is Check with an explicit config file path, so it can be tested
// against a fixture instead of the real ~/.claude.json.
func CheckAt(projectDir, claudeJSONPath string) (Status, error) {
	data, err := os.ReadFile(claudeJSONPath)
	if os.IsNotExist(err) {
		return Status{}, fmt.Errorf("no ~/.claude.json found — has Claude Code been run on this machine?")
	}
	if err != nil {
		return Status{}, err
	}

	var raw struct {
		Projects map[string]struct {
			HasTrustDialogAccepted bool `json:"hasTrustDialogAccepted"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Status{}, err
	}

	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return Status{}, err
	}
	entry, ok := raw.Projects[abs]
	if !ok {
		return Status{Checked: false}, nil
	}
	return Status{Checked: true, Trusted: entry.HasTrustDialogAccepted}, nil
}
