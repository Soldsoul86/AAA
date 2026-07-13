# checkpoint

Automatic, zero-config git checkpoints before your AI agent touches anything.

```
$ checkpoint init
checkpoint: installed a PreToolUse hook in .claude/settings.json
```

That's it. From now on, every time Claude Code is about to edit, write, or run a
notebook cell, checkpoint silently snapshots your working tree first. If the
agent breaks something, you're one command away from the version before it happened.

```
$ checkpoint list
#    AGE       HASH     SOURCE         SUMMARY
1    2m ago    a7ebbba9 hook           3 files changed, 40 insertions(+), 12 deletions(-)
2    9m ago    7a20cf0a hook           1 file changed, 2 insertions(+)

$ checkpoint diff 2      # what changed since that point
$ checkpoint restore 2   # go back to it
```

## Why

AI coding agents occasionally delete, revert, or silently corrupt files they were
asked to edit. The standard advice that's emerged for this is "commit before every
agent session" — which works, but only if you remember to do it, every single time,
under pressure, forever. checkpoint automates the thing you already know you should
be doing.

## Install

```
go install github.com/Soldsoul86/AAA/checkpoint@latest
```

(Prebuilt binaries coming once this has real users — for now, `go build` from source.)

## How it works

There is no custom storage format and no daemon. A checkpoint is a plain git commit
object created with `git stash create` — it doesn't touch HEAD, the index, your
working tree, or the stash ref. The resulting hash is appended, with a timestamp
and label, to `.git/checkpoint/log.jsonl`. That's the entire mechanism.

This means:
- Nothing outside `.git` is ever modified.
- Your commit history and branches are never touched.
- If checkpoint disappears tomorrow, every checkpoint is still a real git object
  you can inspect and restore with plain `git diff <hash>` / `git checkout <hash> -- .`
- Deleting `.git/checkpoint/log.jsonl` forgets everything checkpoint has ever recorded,
  with zero side effects on the repo itself.

## Commands

| Command | What it does |
|---|---|
| `checkpoint init [--user]` | Installs the Claude Code `PreToolUse` hook (project-level by default; `--user` installs to `~/.claude/settings.json` instead) |
| `checkpoint save [--label X]` | Takes a manual snapshot right now |
| `checkpoint list [-n N]` | Shows recent checkpoints and what's changed since each one |
| `checkpoint diff [N]` | Diffs checkpoint `#N` against the current working tree (default: 1, the most recent) |
| `checkpoint restore [N] [--yes]` | Restores tracked files to checkpoint `#N` — always saves your current state as a new checkpoint first, so restore is itself undoable |
| `checkpoint run -- <command>` | Wraps any agent CLI without native hook support: snapshots before it starts and after it exits |

## Using it with tools other than Claude Code

Claude Code is the only tool checkpoint hooks into natively right now, because it's
the only one with a documented `PreToolUse` hook system at the time this was built.
For everything else — Cursor, Codex, Aider, Gemini CLI — wrap the session instead:

```
checkpoint run -- cursor-agent
checkpoint run -- codex
```

This only checkpoints at session start/end rather than before every individual
edit, which is a real gap. Per-tool-call hooks for other agents are the natural
next step once they expose the same kind of hook system Claude Code does.

## Known limitations

- **Untracked files aren't captured.** `git stash create` only snapshots tracked
  files. If your agent creates a brand-new file, it won't be in the checkpoint
  unless you've `git add`ed it first. Handling untracked files safely (without
  risking `git clean`-style data loss) is a planned improvement, not yet built.
- **`restore` never deletes files.** It overwrites tracked files that exist in the
  target checkpoint, but if a file was created *after* that checkpoint, restoring
  won't remove it. This is deliberate — checkpoint would rather leave an extra
  file behind than silently delete something you wanted — but it does mean
  `restore` is not a perfect time machine. Check `checkpoint diff` before you
  trust a restore blindly.
- **The Claude Code hooks schema may drift.** `checkpoint init` writes the hook
  format documented at the time this was built. If Claude Code changes it,
  `init` may need an update — please open an issue rather than assume it's silently broken.

## License

MIT
