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
