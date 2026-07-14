# spare

Makes `rm` (and `Remove-Item`/`del` on Windows) recoverable — the safety net
underneath every AI coding agent's delete command, regardless of which
agent, or which OS, issued it.

![spare catching a real rm and restoring the file — the actual binary, no staged output](./demo.gif)

```
$ rm important.txt
spare: moved 1 item(s) to trash — `spare list` to view, `spare restore <id>` to undo
$ spare restore
spare: restored /Users/you/project/important.txt
```

## Why

Four real, severe, publicly filed incidents motivated this, not a
hypothetical: a custom operating system with 78+ verified build iterations
destroyed by one misinterpreted "clean up the scaffolding" request
([#23913](https://github.com/anthropics/claude-code/issues/23913)); 10
days and 160+ hours of work silently deleted by an internal worktree
cleanup ([#46444](https://github.com/anthropics/claude-code/issues/46444));
18 files mistaken for stray output and removed without confirmation
([#43887](https://github.com/anthropics/claude-code/issues/43887)); an
entire `~/Desktop`, including installed applications, gone in one command
([#30700](https://github.com/anthropics/claude-code/issues/30700)). Every
one of these shares the same root cause: `rm` (or its Windows equivalent)
is permanent, and an agent — like a distracted human — occasionally gets
the scope of a delete wrong.

`checkpoint`, this project's other undo tool, would not have caught any of
these — three were untracked files, and `git stash` structurally cannot
capture what git never tracked. That gap is exactly what `spare` closes,
without needing git at all.

## How it works

`spare` diverts destructive delete commands to a local trash instead of
letting them run for real. The interception mechanism is different on
Unix and Windows, deliberately — a single approach doesn't work on both.

### Unix (macOS, Linux)

`spare init` symlinks the `spare` binary as `rm` inside `~/.spare/shims/`,
then adds that directory to the front of `PATH`. From then on, `rm`
resolves to `spare rm`, which moves the target into the trash instead of
deleting it — for any process, any shell, any agent, or a human typing at
the terminal.

This was verified against real, observed behavior, not assumed: Claude
Code re-sources a "shell snapshot" before every single Bash tool call, and
that snapshot was found to already contain Claude Code's *own* internal
`function grep { ... }` override — proof the vendor itself relies on
exactly this class of shell-shimming technique, on every Bash call, in
every session. A PATH-based shim was chosen over a shell function
specifically because PATH is a plain environment variable, inherited by
any child process including a non-interactive `sh -c "..."` invocation,
whereas a shell function only exists within the shell that defined it —
the more universal, least agent-specific mechanism available. `spare init`
writes the PATH export to `.zshenv` (confirmed sourced for *every* zsh
invocation, not just interactive ones) and to `.bashrc`/`.bash_profile`/
`$BASH_ENV` for bash, to maximize the chance a non-interactive,
agent-spawned shell picks it up too.

### Windows

Windows needed a genuinely different design, not a port of the Unix one.
`rm` and `del` are built-in PowerShell *aliases* for the `Remove-Item`
cmdlet, and PowerShell resolves aliases before it ever consults PATH — a
PATH-shimmed `rm.exe` would simply never be reached. `spare init` instead
removes those built-in aliases and defines functions in their place, plus
a function named `Remove-Item` itself (PowerShell explicitly allows a
function to shadow a cmdlet by name), added to the PowerShell profile.
This catches both `rm`/`del` usage and direct `Remove-Item` calls.

## Verified, not just cross-compiled

Every other tool in this collection has only ever had its Windows and
Linux binaries cross-compiled — never actually run. spare closes that
gap: its own CI workflow
([`spare-ci.yml`](https://github.com/Soldsoul86/AAA/blob/main/.github/workflows/spare-ci.yml))
runs a genuine end-to-end `rm` diversion-and-restore test on real
`ubuntu-latest`, `macos-latest`, and `windows-latest` GitHub Actions
runners, on every push. The first real run of that workflow found and
fixed a real bug — `os.UserHomeDir()` reads `%USERPROFILE%` on Windows,
not `$HOME`, which had silently broken test isolation — the kind of
platform-specific gotcha that cross-compiling alone would never surface.
Both the `rm`/`del` alias interception *and* the harder `Remove-Item`
cmdlet shadowing are confirmed working with real printed output from an
actual Windows runner, not asserted from documentation.

## Usage

```
spare init                install the shim — takes effect in new sessions
spare status               show whether the shim is installed
spare disable              remove everything spare init added
spare rm [-rf] paths...    the actual interception target — you normally
                           never type this yourself, "rm" resolves to it
spare list [--json]        show what's currently in the trash
spare restore [id]         restore the most recently deleted item, or a
                           specific one by id (a unique prefix also works,
                           git-style — you rarely need the full id)
spare purge [DAYS] --yes   permanently delete trash older than DAYS
                           (default: 30) — always shows what would be
                           purged first; nothing is destroyed for good
                           without an explicit --yes
```

`spare init` doesn't apply retroactively to the shell you ran it in — open
a new terminal, or start a new agent session, for it to take effect.

## Known limitations

- **Takes effect on the next new session, not the current one.** Same
  caveat `checkpoint` documents for its own hook: shell startup files and
  Claude Code's shell snapshot are both read once, at the start of a
  session — installing mid-session doesn't retroactively wire it in.
- **A determined `rm -f /path/to/real/binary/rm` (an absolute path
  bypassing PATH resolution entirely) is not intercepted.** Neither is a
  statically-linked or otherwise self-contained deletion syscall made
  without going through a shell at all. spare protects the overwhelmingly
  common case — an agent or a person typing `rm`/`Remove-Item` — not
  every conceivable way to delete a file at the OS level.
- **Restore refuses to overwrite an existing file at the original path
  unless `--force` is passed.** Silently clobbering whatever now occupies
  that path would just be a different way of losing data.
- **No automatic size cap on the trash.** Files accumulate at
  `~/.spare/trash` until `spare purge` is run (or a scheduled purge is set
  up separately — not built into spare itself). A 30-day-old, multi-GB
  trash directory is possible if `purge` is never run.
- **PowerShell mechanism verified live on GitHub's `windows-latest`
  runner, not yet on every real-world Windows configuration** (Windows 10
  vs. 11, PowerShell 5.1 vs. 7+, restricted execution policies in
  corporate environments). The core mechanism is confirmed working; edge
  cases in less common configurations haven't all been exercised.

## Install

```
go install github.com/Soldsoul86/AAA/spare@latest
```

Then run `spare init` and open a new terminal.

Prebuilt binaries, Homebrew, and `.deb`/`.rpm` packages are available once
this has a tagged release — see
[`docs/RELEASING.md`](./docs/RELEASING.md).

## License

MIT
