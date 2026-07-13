package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadMissingFileIsNotAnError(t *testing.T) {
	rs, err := Read(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("a missing settings file should not be an error, got %v", err)
	}
	if len(rs.Allow) != 0 || len(rs.Ask) != 0 || len(rs.Deny) != 0 {
		t.Fatalf("expected an empty RuleSet, got %+v", rs)
	}
}

func TestReadParsesAllThreeRuleTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	content := `{
  "permissions": {
    "allow": ["Bash(npm test)"],
    "ask": ["Bash(git push *)"],
    "deny": ["Bash(rm -rf *)"]
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	rs, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Allow) != 1 || rs.Allow[0] != "Bash(npm test)" {
		t.Fatalf("allow: got %+v", rs.Allow)
	}
	if len(rs.Ask) != 1 || rs.Ask[0] != "Bash(git push *)" {
		t.Fatalf("ask: got %+v", rs.Ask)
	}
	if len(rs.Deny) != 1 || rs.Deny[0] != "Bash(rm -rf *)" {
		t.Fatalf("deny: got %+v", rs.Deny)
	}
}

func TestReadInvalidJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{not valid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}
