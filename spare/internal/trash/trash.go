// Package trash implements the actual store: move-not-delete, with enough
// metadata to restore a file to exactly where it came from. No custom binary
// format — file contents are moved byte-for-byte into a plain directory, and
// metadata is a JSONL log, the same append-only pattern checkpoint uses for
// its own log. If spare disappeared tomorrow, every trashed file is still
// sitting there as a normal file, findable by hand.
package trash

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Entry struct {
	ID           string    `json:"id"`
	OriginalPath string    `json:"original_path"` // absolute
	TrashPath    string    `json:"trash_path"`    // absolute, where it actually lives now
	DeletedAt    time.Time `json:"deleted_at"`
	IsDir        bool      `json:"is_dir"`
}

// Root returns spare's data directory, ~/.spare, creating it if needed.
func Root() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".spare")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return root, nil
}

func objectsDir(root string) string { return filepath.Join(root, "trash") }
func logPath(root string) string    { return filepath.Join(root, "log.jsonl") }

// Move relocates path into the trash and records an Entry. The original
// basename is preserved under a unique ID directory, so two different files
// named "config.json" trashed at different times don't collide and both
// restore under their real name.
func Move(path string) (Entry, error) {
	root, err := Root()
	if err != nil {
		return Entry{}, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return Entry{}, err
	}
	info, err := os.Lstat(absPath) // Lstat: don't follow symlinks, match rm's own semantics
	if err != nil {
		return Entry{}, err
	}

	id, err := newID()
	if err != nil {
		return Entry{}, err
	}
	destDir := filepath.Join(objectsDir(root), id)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return Entry{}, err
	}
	dest := filepath.Join(destDir, filepath.Base(absPath))

	if err := moveAny(absPath, dest); err != nil {
		return Entry{}, err
	}

	e := Entry{
		ID:           id,
		OriginalPath: absPath,
		TrashPath:    dest,
		DeletedAt:    time.Now(),
		IsDir:        info.IsDir(),
	}
	if err := appendLog(logPath(root), e); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// moveAny renames path to dest, falling back to copy+remove on any rename
// failure — most commonly because src and dest are on different filesystems
// (os.Rename returns EXDEV in that case: a real, common situation with an
// external drive, a differently-mounted container volume, or a home
// directory on a separate partition from /tmp). Not trying to distinguish
// EXDEV from other rename failures deliberately — if the real cause is
// something else (a permissions problem, say), the copy fallback fails too,
// with its own clear error, so there's no correctness cost to always trying
// it rather than pattern-matching the specific error first.
func moveAny(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	return copyAndRemove(src, dest)
}

func copyAndRemove(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.Symlink(target, dest); err != nil {
			return err
		}
		return os.Remove(src)
	}
	if info.IsDir() {
		if err := copyDir(src, dest); err != nil {
			return err
		}
		return os.RemoveAll(src)
	}
	if err := copyFile(src, dest, info.Mode()); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return copyFile(p, target, info.Mode())
	})
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(b)), nil
}

func appendLog(path string, e Entry) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// List returns every trash entry, most recently deleted first. A corrupted
// or truncated log line is skipped, not fatal — same posture checkpoint
// takes on its own log for the same reason: a torn write from an interrupted
// process shouldn't make every earlier entry unreadable.
func List() ([]Entry, error) {
	root, err := Root()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(logPath(root))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].DeletedAt.After(entries[j].DeletedAt) })
	return entries, scanner.Err()
}

var ErrNotFound = errors.New("no trash entry with that id")
var ErrTargetExists = errors.New("restore target already exists")

// Restore moves a trashed entry back to its original path. Refuses to
// overwrite an existing file unless force is set — silently overwriting
// whatever now occupies the original path would be its own way of losing
// data, exactly the failure mode this tool exists to prevent.
func Restore(id string, force bool) (Entry, error) {
	entries, err := List()
	if err != nil {
		return Entry{}, err
	}
	var target *Entry
	for i := range entries {
		if entries[i].ID == id {
			target = &entries[i]
			break
		}
	}
	if target == nil {
		return Entry{}, ErrNotFound
	}

	if _, err := os.Lstat(target.OriginalPath); err == nil && !force {
		return Entry{}, ErrTargetExists
	}

	if err := os.MkdirAll(filepath.Dir(target.OriginalPath), 0o755); err != nil {
		return Entry{}, err
	}
	if err := moveAny(target.TrashPath, target.OriginalPath); err != nil {
		return Entry{}, err
	}

	root, err := Root()
	if err != nil {
		return *target, nil // restore itself succeeded; log cleanup is best-effort
	}
	_ = removeFromLog(logPath(root), id)
	return *target, nil
}

// removeFromLog rewrites the log without the given id. Best-effort: a
// failure here doesn't undo a successful restore, it just means `spare list`
// might show a stale entry pointing at a path that no longer has anything
// in the trash — cosmetic, not data-losing.
func removeFromLog(path, id string) error {
	entries, err := List()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	// List() sorts newest-first; write back oldest-first to match the
	// original append order.
	sort.Slice(entries, func(i, j int) bool { return entries[i].DeletedAt.Before(entries[j].DeletedAt) })
	for _, e := range entries {
		if e.ID == id {
			continue
		}
		line, err := json.Marshal(e)
		if err != nil {
			f.Close()
			return err
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Purge permanently deletes trash entries older than olderThan, returning
// how many were removed.
func Purge(olderThan time.Duration) (int, error) {
	entries, err := List()
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-olderThan)
	root, err := Root()
	if err != nil {
		return 0, err
	}

	purged := 0
	var kept []Entry
	for _, e := range entries {
		if e.DeletedAt.Before(cutoff) {
			if err := os.RemoveAll(filepath.Dir(e.TrashPath)); err != nil {
				kept = append(kept, e) // couldn't remove it, keep the record
				continue
			}
			purged++
			continue
		}
		kept = append(kept, e)
	}

	// Rewrite the log with only the kept entries.
	tmp := logPath(root) + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return purged, err
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].DeletedAt.Before(kept[j].DeletedAt) })
	for _, e := range kept {
		line, err := json.Marshal(e)
		if err != nil {
			f.Close()
			return purged, err
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			f.Close()
			return purged, err
		}
	}
	if err := f.Close(); err != nil {
		return purged, err
	}
	return purged, os.Rename(tmp, logPath(root))
}
