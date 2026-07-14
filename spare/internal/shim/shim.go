// Package shim installs spare so that plain "rm" resolves to it. The
// mechanism was chosen after directly inspecting how Claude Code actually
// invokes shell commands in this project's own session: it re-sources a
// "shell snapshot" before every single Bash tool call, and that snapshot
// captures and replays both shell functions and PATH — confirmed by finding
// Claude Code's own internal `function grep { ... }` override sitting in
// that snapshot, used for every Bash call. A PATH-based shim is chosen over
// a shell function specifically because PATH is a plain environment
// variable, inherited by any child process including a non-interactive
// `sh -c "..."` invocation, whereas a shell function only exists within the
// shell that defined it — the more universal, least agent-specific choice.
//
// The shim itself is the spare binary, symlinked to a file named "rm" (or
// "rm.exe" on Windows) in a directory added to the front of PATH. main.go
// checks its own invoked name (argv[0]) and behaves as `spare rm` when
// invoked as "rm" — the same multi-call-binary pattern busybox uses.
package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Dir returns the shim directory, ~/.spare/shims, creating it if needed.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".spare", "shims")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

const marker = "# added by `spare init` — https://github.com/Soldsoul86/AAA/tree/main/spare"

// rcFile describes one shell startup file spare needs to touch, and why.
type rcFile struct {
	path   string
	reason string
}

// targetRCFiles returns every shell startup file worth adding the PATH
// export to, maximizing the chance a non-interactive agent-spawned shell
// picks it up rather than only covering a human's own interactive shell:
//   - .zshenv is sourced for every zsh invocation — interactive, login,
//     non-interactive, all of them — confirmed by zsh's own documented
//     startup file semantics, not assumed.
//   - .bashrc and .bash_profile cover the common interactive/login cases;
//     a $BASH_ENV pointer is also written so non-interactive, non-login
//     bash shells (the ones many agents actually spawn) pick it up too,
//     since bash reads $BASH_ENV specifically for that case and nothing
//     else by default.
func targetRCFiles(home string) []rcFile {
	return []rcFile{
		{filepath.Join(home, ".zshenv"), "zsh: sourced for every invocation"},
		{filepath.Join(home, ".bashrc"), "bash: interactive shells"},
		{filepath.Join(home, ".bash_profile"), "bash: login shells"},
	}
}

const bashEnvFileName = ".spare-bash-env"

// Install symlinks the running binary as "rm" inside the shim directory and
// adds that directory to PATH via every shell startup file it can find.
// Idempotent: running it twice doesn't duplicate the PATH line.
func Install() (installed []string, err error) {
	if runtime.GOOS == "windows" {
		return installWindows()
	}
	return installUnix()
}

func installUnix() ([]string, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return nil, err
	}

	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	rmPath := filepath.Join(dir, "rm")
	if existing, err := os.Readlink(rmPath); err != nil || existing != self {
		os.Remove(rmPath) // ignore error: fine if it didn't exist
		if err := os.Symlink(self, rmPath); err != nil {
			return nil, fmt.Errorf("linking %s -> %s: %w", rmPath, self, err)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// $BASH_ENV target: a tiny standalone file, not one of the rc files
	// themselves, so it stays correct even if the user's own .bashrc has
	// an early "non-interactive? bail out" guard (a common pattern that
	// would otherwise skip anything appended after it).
	bashEnvFile := filepath.Join(home, bashEnvFileName)
	if err := writeIfMissing(bashEnvFile, exportLine(dir)+"\n"); err != nil {
		return nil, err
	}

	var touched []string
	for _, rc := range targetRCFiles(home) {
		if err := appendPathExport(rc.path, dir); err != nil {
			return nil, fmt.Errorf("updating %s: %w", rc.path, err)
		}
		touched = append(touched, rc.path)
	}
	// Point BASH_ENV at the file above from .bash_profile /.bashrc too, so
	// a login shell that then spawns non-interactive children propagates it.
	for _, rc := range []string{filepath.Join(home, ".bashrc"), filepath.Join(home, ".bash_profile")} {
		line := fmt.Sprintf(`[ -z "${BASH_ENV:-}" ] && export BASH_ENV=%q`, bashEnvFile)
		if err := appendIfMissing(rc, line); err != nil {
			return nil, err
		}
	}

	return touched, nil
}

func exportLine(shimDir string) string {
	return fmt.Sprintf(`export PATH=%q:"$PATH"`, shimDir)
}

// installWindows takes a fundamentally different approach than Unix,
// because a PATH-based executable shim doesn't work for PowerShell: "rm"
// and "del" are built-in *aliases* for the Remove-Item cmdlet, and
// PowerShell's command resolution order is alias > function > cmdlet >
// external executable — an rm.exe sitting in PATH is never reached, because
// the alias wins first. The only way to actually intercept this is to
// remove the built-in aliases and define functions in their place, plus
// define a function named Remove-Item itself (PowerShell explicitly allows
// a function to shadow a cmdlet name) to catch direct cmdlet calls too.
//
// This has NOT been verified against a real Windows machine — written
// against documented PowerShell alias/function/cmdlet resolution order,
// not confirmed live. Real verification is tracked separately (real
// Windows CI, not just cross-compilation) before this should be trusted
// the way the Unix mechanism now can be.
func installWindows() ([]string, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Both PowerShell 7+ and Windows PowerShell 5.1 profile locations are
	// written to, since which one is actually loaded depends on which
	// powershell.exe / pwsh.exe an agent invokes, and that can't be
	// detected from here.
	profiles := []string{
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	}

	block := powerShellBlock(self)
	var touched []string
	for _, p := range profiles {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		if err := appendIfMissing(p, block); err != nil {
			return nil, fmt.Errorf("updating %s: %w", p, err)
		}
		touched = append(touched, p)
	}
	return touched, nil
}

func powerShellBlock(spareExe string) string {
	return fmt.Sprintf(`if (Get-Alias rm -ErrorAction SilentlyContinue) { Remove-Alias rm -Force }
if (Get-Alias del -ErrorAction SilentlyContinue) { Remove-Alias del -Force }
function Remove-Item {
    param([Parameter(ValueFromRemainingArguments=$true)]$RestArgs)
    & %q rm @RestArgs
}
function rm { Remove-Item @args }
function del { Remove-Item @args }`, spareExe)
}

func appendPathExport(rc, shimDir string) error {
	return appendIfMissing(rc, exportLine(shimDir))
}

func appendIfMissing(path, line string) error {
	existing, _ := os.ReadFile(path) // missing file is fine, starts empty
	if strings.Contains(string(existing), line) {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n" + marker + "\n" + line + "\n")
	return err
}

func writeIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // don't clobber a file that already exists for another reason
	}
	return os.WriteFile(path, []byte(marker+"\n"+content), 0o644)
}

// Status reports whether the shim is installed and where.
type Status struct {
	ShimDir        string
	RMLinkExists   bool
	RCFilesTouched []string
}

func CurrentStatus() (Status, error) {
	dir, err := Dir()
	if err != nil {
		return Status{}, err
	}
	s := Status{ShimDir: dir}
	if _, err := os.Lstat(filepath.Join(dir, "rm")); err == nil {
		s.RMLinkExists = true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return s, err
	}
	for _, rc := range targetRCFiles(home) {
		content, err := os.ReadFile(rc.path)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), marker) {
			s.RCFilesTouched = append(s.RCFilesTouched, rc.path)
		}
	}
	return s, nil
}

// Uninstall removes everything spare init added: the rm symlink and the
// marked block from every rc file it touched. Anything not added by spare
// (a user's own PATH exports, functions, etc.) is left untouched.
func Uninstall() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	os.Remove(filepath.Join(dir, "rm"))
	os.Remove(filepath.Join(dir, "rm.exe"))

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	files := targetRCFiles(home)
	files = append(files,
		rcFile{path: filepath.Join(home, ".bashrc")},
		rcFile{path: filepath.Join(home, ".bash_profile")},
	)
	seen := map[string]bool{}
	for _, rc := range files {
		if seen[rc.path] {
			continue
		}
		seen[rc.path] = true
		if err := removeMarkedLines(rc.path); err != nil {
			return err
		}
	}
	os.Remove(filepath.Join(home, bashEnvFileName))
	return nil
}

// removeMarkedLines strips every "marker line + the line after it" block
// spare's own appends added, leaving everything else in the file exactly
// as it was.
func removeMarkedLines(path string) error {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var kept []string
	skipNext := 0
	for _, line := range lines {
		if skipNext > 0 {
			skipNext--
			continue
		}
		if strings.TrimSpace(line) == marker {
			skipNext = 1 // the line spare wrote right after its own marker
			continue
		}
		kept = append(kept, line)
	}
	return os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o644)
}
