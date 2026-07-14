package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// os.UserHomeDir() reads $USERPROFILE on Windows, not $HOME — see the
// identical comment in internal/trash's own withTempHome, where this was
// first found via a real CI failure on windows-latest.
func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func TestInstallUnixCreatesRMSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific mechanism")
	}
	withTempHome(t)

	if _, err := Install(); err != nil {
		t.Fatal(err)
	}

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(dir, "rm"))
	if err != nil {
		t.Fatal(err)
	}
	self, _ := os.Executable()
	self, _ = filepath.EvalSymlinks(self)
	if target != self {
		t.Fatalf("rm symlink points to %q, want %q", target, self)
	}
}

func TestInstallUnixTouchesRCFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific mechanism")
	}
	home := withTempHome(t)

	touched, err := Install()
	if err != nil {
		t.Fatal(err)
	}
	if len(touched) == 0 {
		t.Fatal("expected at least one rc file touched")
	}

	zshenv, err := os.ReadFile(filepath.Join(home, ".zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(zshenv), "export PATH=") {
		t.Fatalf(".zshenv doesn't contain a PATH export: %s", zshenv)
	}
	if !strings.Contains(string(zshenv), marker) {
		t.Fatal(".zshenv doesn't contain spare's marker comment")
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific mechanism")
	}
	home := withTempHome(t)

	if _, err := Install(); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(home, ".zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(string(content), marker)
	if count != 1 {
		t.Fatalf("marker appears %d times after running Install twice, want exactly 1 (not duplicated)", count)
	}
}

func TestInstallDoesNotClobberExistingRCContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific mechanism")
	}
	home := withTempHome(t)
	preexisting := "# my own stuff\nexport EDITOR=vim\n"
	if err := os.WriteFile(filepath.Join(home, ".zshenv"), []byte(preexisting), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(home, ".zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "export EDITOR=vim") {
		t.Fatal("Install must not remove the user's own existing rc content")
	}
}

func TestStatusReflectsInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific mechanism")
	}
	withTempHome(t)

	before, err := CurrentStatus()
	if err != nil {
		t.Fatal(err)
	}
	if before.RMLinkExists {
		t.Fatal("RMLinkExists should be false before Install")
	}

	if _, err := Install(); err != nil {
		t.Fatal(err)
	}

	after, err := CurrentStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !after.RMLinkExists {
		t.Fatal("RMLinkExists should be true after Install")
	}
	if len(after.RCFilesTouched) == 0 {
		t.Fatal("RCFilesTouched should be non-empty after Install")
	}
}

func TestUninstallRemovesOnlySpareContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific mechanism")
	}
	home := withTempHome(t)
	preexisting := "# my own stuff\nexport EDITOR=vim\n"
	if err := os.WriteFile(filepath.Join(home, ".zshenv"), []byte(preexisting), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(); err != nil {
		t.Fatal(err)
	}

	dir, _ := Dir()
	if _, err := os.Lstat(filepath.Join(dir, "rm")); !os.IsNotExist(err) {
		t.Fatal("rm symlink should be gone after Uninstall")
	}

	content, err := os.ReadFile(filepath.Join(home, ".zshenv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), marker) {
		t.Fatal("spare's marker/content should be gone after Uninstall")
	}
	if !strings.Contains(string(content), "export EDITOR=vim") {
		t.Fatal("Uninstall must not remove the user's own pre-existing content")
	}

	status, err := CurrentStatus()
	if err != nil {
		t.Fatal(err)
	}
	if status.RMLinkExists || len(status.RCFilesTouched) != 0 {
		t.Fatalf("status after Uninstall should show nothing installed, got %+v", status)
	}
}

func TestPowerShellBlockShadowsBothAliasesAndCmdlet(t *testing.T) {
	block := powerShellBlock("C:\\Users\\test\\spare.exe")
	for _, want := range []string{"Remove-Alias", "function Remove-Item", "function rm", "function del"} {
		if !strings.Contains(block, want) {
			t.Errorf("PowerShell block missing %q:\n%s", want, block)
		}
	}
}
