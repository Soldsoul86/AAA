package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddAllowCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "settings.json")

	added, err := AddAllow(path, "Bash(npm run *)")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected added=true for a new rule in a new file")
	}

	rs, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Allow) != 1 || rs.Allow[0] != "Bash(npm run *)" {
		t.Fatalf("got %+v", rs)
	}
}

func TestAddAllowIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	added1, err := AddAllow(path, "Bash(git commit *)")
	if err != nil || !added1 {
		t.Fatalf("first add: added=%v err=%v", added1, err)
	}
	added2, err := AddAllow(path, "Bash(git commit *)")
	if err != nil {
		t.Fatal(err)
	}
	if added2 {
		t.Fatal("expected added=false on duplicate rule")
	}

	rs, _ := Read(path)
	if len(rs.Allow) != 1 {
		t.Fatalf("expected exactly 1 rule after adding the same one twice, got %d", len(rs.Allow))
	}
}

func TestAddAllowPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := `{
  "hooks": {"PreToolUse": [{"matcher": "Edit", "hooks": [{"type": "command", "command": "checkpoint save"}]}]},
  "permissions": {"deny": ["Bash(rm -rf *)"]}
}`
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := AddAllow(path, "Bash(npm test)"); err != nil {
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
	if _, ok := raw["hooks"]; !ok {
		t.Fatal("expected the pre-existing hooks key to survive AddAllow untouched")
	}

	rs, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Deny) != 1 || rs.Deny[0] != "Bash(rm -rf *)" {
		t.Fatalf("expected the pre-existing deny rule to survive, got %+v", rs.Deny)
	}
	if len(rs.Allow) != 1 || rs.Allow[0] != "Bash(npm test)" {
		t.Fatalf("expected the new allow rule to be added, got %+v", rs.Allow)
	}
}

func TestReadMissingFileIsNotAnError(t *testing.T) {
	rs, err := Read(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("a missing settings file should not be an error, got %v", err)
	}
	if len(rs.Allow) != 0 || len(rs.Ask) != 0 || len(rs.Deny) != 0 {
		t.Fatalf("expected an empty RuleSet, got %+v", rs)
	}
}
