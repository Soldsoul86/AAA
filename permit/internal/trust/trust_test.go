package trust

import (
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
