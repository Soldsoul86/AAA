// Package settings safely reads and merges permission rules into Claude
// Code's settings.json files, preserving every other key untouched.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Source identifies which of Claude Code's layered settings files a rule
// should be read from or written to.
type Source struct {
	Name string // human-readable, e.g. "project"
	Path string
}

// Sources returns the settings files in the precedence order documented at
// code.claude.com/docs/en/permissions: local project settings, shared
// project settings, then user settings. Managed settings are deliberately
// excluded — permit should never write to a file meant to be centrally
// controlled by an organization.
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

// AddAllow appends a rule to permissions.allow in the given settings file,
// creating the file and the permissions.allow array if needed, and leaving
// every other key untouched. Returns false if the rule is already present
// (idempotent, matching checkpoint's hook installer).
func AddAllow(path, rule string) (bool, error) {
	raw := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return false, err
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}

	permissions, _ := raw["permissions"].(map[string]interface{})
	if permissions == nil {
		permissions = map[string]interface{}{}
	}

	var allow []interface{}
	if existing, ok := permissions["allow"].([]interface{}); ok {
		allow = existing
	}
	for _, a := range allow {
		if s, ok := a.(string); ok && s == rule {
			return false, nil // already present
		}
	}
	allow = append(allow, rule)
	permissions["allow"] = allow
	raw["permissions"] = permissions

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return false, err
	}
	out = append(out, '\n')
	return true, os.WriteFile(path, out, 0o644)
}
