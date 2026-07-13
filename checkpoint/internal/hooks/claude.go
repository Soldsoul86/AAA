// Package hooks installs checkpoint into Claude Code's PreToolUse hook so a
// snapshot is taken automatically before any file-editing tool call.
//
// NOTE: Claude Code's hooks schema has changed before and may change again.
// This writes the structure documented at the time this was built. If it
// stops working, `checkpoint init` still prints the JSON it would have
// written so you can add it by hand.
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const matcher = "Edit|Write|MultiEdit|NotebookEdit"
const command = "checkpoint save --quiet --label auto --source hook"

type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type matcherGroup struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

// InstallClaudeCode adds the checkpoint PreToolUse hook to a Claude Code
// settings.json file, creating the file if needed and leaving every other
// key untouched. Returns true if it changed anything.
func InstallClaudeCode(settingsPath string) (bool, error) {
	raw := map[string]interface{}{}

	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &raw); err != nil {
			return false, fmt.Errorf("%s is not valid JSON: %w", settingsPath, err)
		}
	} else if !os.IsNotExist(err) {
		return false, err
	}

	hooksRaw, _ := raw["hooks"].(map[string]interface{})
	if hooksRaw == nil {
		hooksRaw = map[string]interface{}{}
	}

	preToolUse, _ := toMatcherGroups(hooksRaw["PreToolUse"])

	for _, g := range preToolUse {
		if g.Matcher == matcher {
			for _, h := range g.Hooks {
				if h.Command == command {
					return false, nil // already installed
				}
			}
		}
	}

	preToolUse = append(preToolUse, matcherGroup{
		Matcher: matcher,
		Hooks:   []hookEntry{{Type: "command", Command: command}},
	})

	hooksRaw["PreToolUse"] = preToolUse
	raw["hooks"] = hooksRaw

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return false, err
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return false, err
	}
	out = append(out, '\n')
	if err := os.WriteFile(settingsPath, out, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func toMatcherGroups(v interface{}) ([]matcherGroup, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var groups []matcherGroup
	if err := json.Unmarshal(b, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}
