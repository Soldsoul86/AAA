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

// SetTrusted marks a project as trusted in the real ~/.claude.json,
// creating its project entry if none exists. It touches only that one
// project's hasTrustDialogAccepted field — every other key in the file
// (including sensitive ones like OAuth account data) is read, preserved
// byte-for-byte in structure, and written back untouched. The file's
// permission mode is preserved exactly, not loosened.
//
// This bypasses a real security gate — Claude Code's workspace trust
// dialog exists so a project can't silently gain capability the moment
// you open it. Only call this for projects you already trust yourself.
func SetTrusted(projectDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return SetTrustedAt(projectDir, filepath.Join(home, ".claude.json"))
}

// SetTrustedAt is SetTrusted with an explicit config file path, so it can
// be tested against a fixture instead of the real ~/.claude.json.
func SetTrustedAt(projectDir, claudeJSONPath string) error {
	var mode os.FileMode = 0o600 // matches the real file's own permissions if it exists
	raw := map[string]interface{}{}

	if data, err := os.ReadFile(claudeJSONPath); err == nil {
		if info, statErr := os.Stat(claudeJSONPath); statErr == nil {
			mode = info.Mode()
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &raw); err != nil {
				return fmt.Errorf("%s is not valid JSON, refusing to touch it: %w", claudeJSONPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	projects, _ := raw["projects"].(map[string]interface{})
	if projects == nil {
		projects = map[string]interface{}{}
	}

	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	entry, _ := projects[abs].(map[string]interface{})
	if entry == nil {
		entry = map[string]interface{}{}
	}
	entry["hasTrustDialogAccepted"] = true
	projects[abs] = entry
	raw["projects"] = projects

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(claudeJSONPath, out, mode)
}
