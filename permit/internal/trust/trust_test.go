package trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckTrustedProject(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, `{"projects": {"/some/project": {"hasTrustDialogAccepted": true}}}`)

	status, err := CheckAt("/some/project", path)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Checked || !status.Trusted {
		t.Fatalf("expected Checked=true Trusted=true, got %+v", status)
	}
}

func TestCheckUntrustedProject(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, `{"projects": {"/some/project": {"hasTrustDialogAccepted": false}}}`)

	status, err := CheckAt("/some/project", path)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Checked || status.Trusted {
		t.Fatalf("expected Checked=true Trusted=false, got %+v", status)
	}
}

func TestCheckUnknownProject(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, `{"projects": {"/some/other/project": {"hasTrustDialogAccepted": true}}}`)

	status, err := CheckAt("/some/project", path)
	if err != nil {
		t.Fatal(err)
	}
	if status.Checked {
		t.Fatalf("expected Checked=false for a project with no entry, got %+v", status)
	}
}

func TestCheckMissingFile(t *testing.T) {
	_, err := CheckAt("/some/project", filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err == nil {
		t.Fatal("expected an error when ~/.claude.json doesn't exist at all")
	}
}

func TestSetTrustedCreatesNewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	if err := SetTrustedAt("/some/project", path); err != nil {
		t.Fatal(err)
	}
	status, err := CheckAt("/some/project", path)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Checked || !status.Trusted {
		t.Fatalf("expected the newly created entry to read back as trusted, got %+v", status)
	}
}

func TestSetTrustedAddsNewProjectWithoutTouchingOthers(t *testing.T) {
	path := writeFixture(t, t.TempDir(), `{"projects": {"/existing/project": {"hasTrustDialogAccepted": true, "allowedTools": ["Bash(npm test)"]}}}`)

	if err := SetTrustedAt("/new/project", path); err != nil {
		t.Fatal(err)
	}

	newStatus, err := CheckAt("/new/project", path)
	if err != nil || !newStatus.Trusted {
		t.Fatalf("new project should be trusted, got %+v err=%v", newStatus, err)
	}

	// the pre-existing project's own data, including a field SetTrusted
	// never touches, must survive completely intact
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Projects map[string]struct {
			HasTrustDialogAccepted bool     `json:"hasTrustDialogAccepted"`
			AllowedTools           []string `json:"allowedTools"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	existing, ok := raw.Projects["/existing/project"]
	if !ok {
		t.Fatal("the pre-existing project entry was lost entirely")
	}
	if !existing.HasTrustDialogAccepted {
		t.Fatal("the pre-existing project's trust status was clobbered")
	}
	if len(existing.AllowedTools) != 1 || existing.AllowedTools[0] != "Bash(npm test)" {
		t.Fatalf("the pre-existing project's allowedTools field was lost, got %+v", existing.AllowedTools)
	}
}

func TestSetTrustedPreservesUnrelatedTopLevelKeys(t *testing.T) {
	// Simulates the shape of a real ~/.claude.json, which has sensitive
	// top-level keys (oauthAccount, userID, etc.) alongside "projects".
	// SetTrusted must never touch these.
	path := writeFixture(t, t.TempDir(), `{
		"userID": "abc123",
		"oauthAccount": {"token": "should-never-be-touched"},
		"projects": {}
	}`)

	if err := SetTrustedAt("/some/project", path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["userID"] != "abc123" {
		t.Fatalf("userID was not preserved, got %v", raw["userID"])
	}
	oauth, ok := raw["oauthAccount"].(map[string]interface{})
	if !ok || oauth["token"] != "should-never-be-touched" {
		t.Fatalf("oauthAccount was not preserved untouched, got %v", raw["oauthAccount"])
	}
}

func TestSetTrustedIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	if err := SetTrustedAt("/some/project", path); err != nil {
		t.Fatal(err)
	}
	if err := SetTrustedAt("/some/project", path); err != nil {
		t.Fatal(err)
	}
	status, err := CheckAt("/some/project", path)
	if err != nil || !status.Trusted {
		t.Fatalf("expected still trusted after calling twice, got %+v err=%v", status, err)
	}
}

func TestSetTrustedPreservesFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(`{"projects": {}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SetTrustedAt("/some/project", path); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected file permissions to stay 0600, got %o", info.Mode().Perm())
	}
}

func TestSetTrustedRejectsInvalidJSONRatherThanClobberIt(t *testing.T) {
	path := writeFixture(t, t.TempDir(), `{not valid json`)
	if err := SetTrustedAt("/some/project", path); err == nil {
		t.Fatal("expected an error rather than silently overwriting invalid JSON")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{not valid json` {
		t.Fatal("the invalid file should have been left completely untouched after a failed parse")
	}
}

// This is the real, high-value test: check the actual ~/.claude.json on
// this machine against the real home directory, to confirm CheckAt's
// parsing logic works against genuine data, not just a hand-built fixture.
func TestCheckAgainstRealClaudeJSON(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}
	realPath := filepath.Join(home, ".claude.json")
	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		t.Skip("no real ~/.claude.json on this machine")
	}

	status, err := CheckAt(home, realPath)
	if err != nil {
		t.Fatalf("CheckAt against the real config file failed to parse: %v", err)
	}
	t.Logf("real trust status for %s: checked=%v trusted=%v", home, status.Checked, status.Trusted)
}
