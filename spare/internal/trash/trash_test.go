package trash

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withTempHome points os.UserHomeDir at a fresh temp dir so tests never
// touch the real ~/.spare. Real bug, found via CI on windows-latest:
// os.UserHomeDir() reads $USERPROFILE on Windows, not $HOME — setting HOME
// alone left every Windows test operating against the actual runner's real
// home directory, silently, which is exactly what let entries from
// unrelated tests bleed into each other.
func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func TestMoveAndList(t *testing.T) {
	home := withTempHome(t)
	src := filepath.Join(home, "project", "important.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("real content"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := Move(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("original path should no longer exist after Move")
	}
	content, err := os.ReadFile(e.TrashPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "real content" {
		t.Fatalf("trashed content = %q, want %q", content, "real content")
	}

	entries, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != e.ID {
		t.Fatalf("List() = %+v, want one entry matching %q", entries, e.ID)
	}
}

func TestMoveDirectory(t *testing.T) {
	home := withTempHome(t)
	src := filepath.Join(home, "project", "subdir")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := Move(src)
	if err != nil {
		t.Fatal(err)
	}
	if !e.IsDir {
		t.Error("IsDir = false, want true")
	}
	if _, err := os.Stat(filepath.Join(e.TrashPath, "nested", "file.txt")); err != nil {
		t.Fatalf("nested file not preserved: %v", err)
	}
}

func TestMoveSymlinkDoesNotFollowTarget(t *testing.T) {
	home := withTempHome(t)
	target := filepath.Join(home, "real.txt")
	if err := os.WriteFile(target, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if _, err := Move(link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("symlink target should survive trashing the link itself: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatal("the symlink itself should be gone")
	}
}

func TestCopyAndRemoveFallback(t *testing.T) {
	// Exercises the same code path moveAny falls back to on a cross-device
	// rename, without needing an actual second filesystem: call the
	// fallback function directly, matching what happens when os.Rename
	// returns EXDEV in the real world.
	home := withTempHome(t)
	src := filepath.Join(home, "src.txt")
	dest := filepath.Join(home, "dest", "src.txt")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("cross-device content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyAndRemove(src, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("source should be removed after copyAndRemove")
	}
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "cross-device content" {
		t.Fatalf("got %q", content)
	}
}

func TestRestore(t *testing.T) {
	home := withTempHome(t)
	src := filepath.Join(home, "restore-me.txt")
	if err := os.WriteFile(src, []byte("restore content"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := Move(src)
	if err != nil {
		t.Fatal(err)
	}

	restored, err := Restore(e.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(restored.OriginalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "restore content" {
		t.Fatalf("got %q", content)
	}

	entries, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("restored entry should be removed from the log, got %+v", entries)
	}
}

func TestRestoreRefusesToOverwriteWithoutForce(t *testing.T) {
	home := withTempHome(t)
	src := filepath.Join(home, "conflict.txt")
	if err := os.WriteFile(src, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := Move(src)
	if err != nil {
		t.Fatal(err)
	}
	// Something new now occupies the original path.
	if err := os.WriteFile(src, []byte("new file, unrelated"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Restore(e.ID, false); err != ErrTargetExists {
		t.Fatalf("Restore without force = %v, want ErrTargetExists", err)
	}
	content, _ := os.ReadFile(src)
	if string(content) != "new file, unrelated" {
		t.Fatal("the new file at the original path must not be touched when restore is refused")
	}
}

func TestRestoreForceOverwrites(t *testing.T) {
	home := withTempHome(t)
	src := filepath.Join(home, "conflict2.txt")
	if err := os.WriteFile(src, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := Move(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("will be overwritten"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Restore(e.ID, true); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("got %q, want the restored original content", content)
	}
}

func TestRestoreUnknownID(t *testing.T) {
	withTempHome(t)
	if _, err := Restore("does-not-exist", false); err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestListSkipsCorruptedLines(t *testing.T) {
	home := withTempHome(t)
	root, err := Root()
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(home, "good.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Move(src); err != nil {
		t.Fatal(err)
	}

	// Append a corrupted line by hand, simulating a torn write.
	f, err := os.OpenFile(logPath(root), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not valid json\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	entries, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected the 1 good entry to survive a corrupted line, got %d", len(entries))
	}
}

func TestPurgeRemovesOldEntriesOnly(t *testing.T) {
	home := withTempHome(t)
	old := filepath.Join(home, "old.txt")
	recent := filepath.Join(home, "recent.txt")
	if err := os.WriteFile(old, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recent, []byte("recent"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldEntry, err := Move(old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Move(recent); err != nil {
		t.Fatal(err)
	}

	// Backdate the old entry's DeletedAt by rewriting the log directly,
	// simulating time passing without needing to actually sleep in the test.
	root, _ := Root()
	entries, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for i := range entries {
		if entries[i].ID == oldEntry.ID {
			entries[i].DeletedAt = time.Now().Add(-48 * time.Hour)
		}
	}
	rewriteLogForTest(t, logPath(root), entries)

	purged, err := Purge(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if purged != 1 {
		t.Fatalf("purged = %d, want 1", purged)
	}
	remaining, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].OriginalPath != recent {
		t.Fatalf("remaining = %+v, want only the recent entry", remaining)
	}
}

func rewriteLogForTest(t *testing.T, path string, entries []Entry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			t.Fatal(err)
		}
	}
}
