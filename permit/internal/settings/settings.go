// Package settings reads permission rules from Claude Code's settings.json
// files. It is read-only — permit doesn't write rules, only diagnoses them.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Source identifies one of Claude Code's layered settings files.
type Source struct {
	Name string // human-readable, e.g. "project"
	Path string
}

// Sources returns the settings files in the precedence order documented at
// code.claude.com/docs/en/permissions: local project settings, shared
// project settings, then user settings. Managed settings are deliberately
// excluded — they're centrally controlled and not something permit needs
// to read for its diagnosis.
func Sources(projectRoot string) []Source {
	home, _ := os.UserHomeDir()
	return []Source{
		{"local project (.claude/settings.local.json)", filepath.Join(projectRoot, ".claude", "settings.local.json")},
		{"shared project (.claude/settings.json)", filepath.Join(projectRoot, ".claude", "settings.json")},
		{"user (~/.claude/settings.json)", filepath.Join(home, ".claude", "settings.json")},
	}
}

// RuleSet is the merged view of one settings file's permission rules.
type RuleSet struct {
	Allow []string
	Ask   []string
	Deny  []string
}

// Read loads the permissions block from a settings file. A missing file is
// not an error — it just has no rules yet.
func Read(path string) (RuleSet, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return RuleSet{}, nil
	}
	if err != nil {
		return RuleSet{}, err
	}
	var raw struct {
		Permissions struct {
			Allow []string `json:"allow"`
			Ask   []string `json:"ask"`
			Deny  []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return RuleSet{}, err
	}
	return RuleSet{Allow: raw.Permissions.Allow, Ask: raw.Permissions.Ask, Deny: raw.Permissions.Deny}, nil
}
