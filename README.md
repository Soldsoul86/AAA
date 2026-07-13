# AAA

Small, single-purpose, open-source tools that sit alongside AI coding
agents (Claude Code, Cursor, Codex, Gemini CLI) and catch the specific,
recurring failure modes they're prone to. None of them are AI tools
themselves — no accounts, no backend, no billing. Each installs in one
command and does exactly one thing.

| Tool | Category | What it does |
|---|---|---|
| [checkpoint](./checkpoint) | Safety | Automatic git checkpoints before your AI agent runs a bash command — the gap Claude Code's native `/rewind` doesn't cover |
| [loopkill](./loopkill) | Awareness | Detects when your AI agent is stuck in a loop and interrupts it before it burns your quota |
| [ctxmeter](./ctxmeter) | Awareness | A live context-window gauge for AI coding CLIs that don't already have one (Claude Code users: use `/statusline` instead) |
| [again](./again) | Awareness | Counts how many times you've had to repeat yourself to your AI agent this session |
| [permit](./permit) | Safety | Diagnoses why a Claude Code permission rule silently doesn't apply |
| [actually](./actually) | Trust | Cross-checks a "tests pass" claim against the real last test run in the same session |
| [exists](./exists) | Trust | Checks whether a package your agent just installed (npm/PyPI) actually exists in the real registry |

## Verified, not vibes

Every tool here has a "Known limitations" and "Verified against real data"
section in its own README — the point of this section is to put the
concrete evidence in one place instead of six, so it's not just a claim.

- **checkpoint**: hardened against 8 real scenarios beyond the happy path
  (no commits yet, detached HEAD, 30 truly concurrent saves, corrupted log
  lines, symlinks/binaries) — 2 real bugs found and fixed along the way.
  Verified via a real `brew`-installed binary: `brew upgrade`, three full
  uninstall/reinstall cycles, zero leftover state.
- **loopkill**: verified against a real simulated stuck process — zero
  false positives on a spinner, correct detection and kill on a genuine
  loop, not a synthetic pass/fail assertion.
- **ctxmeter**: verified that piping through it never swallows the wrapped
  CLI's real output — the gauge renders on stderr, stdout reaches the
  terminal byte-for-byte unmodified.
- **again**: the real Claude Code transcript shape was confirmed correct
  against a live session, not assumed — and its token counts are the
  actual measured length of a repeated prompt, deliberately never called
  "tokens saved," because that would require guessing a counterfactual no
  tool can actually know.
- **permit**: cut from three commands down to one (`doctor`) after
  discovering Claude Code already had native features doing the same job.
  Shipping less, on purpose, once redundant work was found, rather than
  keeping code around because it existed.
- **actually**: swept against ~500 real assistant messages from this
  project's own development session — 0 false positives after fixing 3
  real bugs found along the way, including a silent-data-loss bug where
  one oversized transcript line could drop detection for the rest of a
  session with zero error output.
- **exists**: its test suite makes live calls to the real npm and PyPI
  registries rather than mocking them, because the entire point is whether
  the real registry agrees. A pre-release audit found its own pitch had
  quietly oversold what it protects against (it can't catch a package name
  that's *already* been squatted, only one that's still unclaimed) — the
  README was rewritten to say so plainly rather than leave the stronger,
  less accurate claim standing.

## Why separate tools instead of one platform

Each one is a narrow fix for a single, specific, well-evidenced annoyance —
in the spirit of tools like `fzf`, `ripgrep`, and `atuin`, not a bundled
suite you have to buy into all at once. Use one, use all seven, doesn't
matter — none of them depend on each other.

## Install

Each tool has its own README with full details. The short version, once
releases exist (see each tool's `docs/RELEASING.md`):

```
go install github.com/Soldsoul86/AAA/checkpoint@latest
go install github.com/Soldsoul86/AAA/loopkill@latest
go install github.com/Soldsoul86/AAA/ctxmeter@latest
go install github.com/Soldsoul86/AAA/again@latest
go install github.com/Soldsoul86/AAA/permit@latest
go install github.com/Soldsoul86/AAA/actually@latest
go install github.com/Soldsoul86/AAA/exists@latest
```

Or build from source today, before any release exists:

```
git clone https://github.com/Soldsoul86/AAA
cd AAA/checkpoint && go build -o checkpoint .
```

## Structure

This is a monorepo — one repo, independent Go modules, one folder each.
Every tool has its own `go.mod`, its own tests, its own
`.goreleaser.yaml`, and its own release workflow
(`.github/workflows/release-<tool>.yml`), triggered only by that tool's own
tag prefix (e.g. `checkpoint-v0.1.0`) so releasing one tool never touches
the others.

## License

MIT, per tool (see each folder's `LICENSE`).
