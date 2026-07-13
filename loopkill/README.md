# loopkill

Detects when your AI agent is stuck in a loop and interrupts it before it burns your quota.

```
$ loopkill -- claude
loopkill: watching "claude" (threshold=3, window=50)
...
⚠️  loopkill: this line has repeated 3 times in the last 50 lines — looks stuck:
    "Reading file src/utils.go"
```

## Why

Agentic coding tools sometimes get stuck repeating the same failed action —
rereading the same file, retrying the same edit — with no self-detection.
By the time a human notices, it's burned real time and real tokens.

## How it works

`loopkill` wraps any command. Its output is passed straight through to your
terminal **byte-for-byte, with zero added latency** — loopkill never touches
what you see, so the agent's own UI renders exactly as it would without it.

Separately, on a copy of that output, it strips ANSI escape codes and watches
for lines that keep reappearing. Critically: **a line repeated on consecutive
frames (a redrawn spinner, a progress bar) is treated as one event, not
counted as looping.** Only a line that shows up again after other output —
the actual signature of a stuck agent — counts toward the threshold. Verified
in testing: a 15-frame spinner produces zero false positives, while a
genuinely repeated action fires reliably.

Each stuck pattern is only reported once per episode — it won't spam you on
every subsequent repeat of the same line — but can fire again later if that
exact line ages out of the recent-history window and then recurs.

## Usage

```
loopkill [flags] -- <command> [args...]

  -threshold N   how many times a line must repeat within the window to count as a loop (default 3)
  -window N      how many recent distinct lines to consider (default 50)
  -kill          send SIGINT to the wrapped process when a loop is detected (default: warn only)
  -quiet         suppress the startup banner
```

By default loopkill only warns — loudly, with a terminal bell — and lets the
agent keep running, since a false positive that kills a legitimately slow
task is worse than an extra warning. Pass `-kill` once you trust it for your
setup and want it to actually interrupt the process (SIGINT, so the wrapped
CLI gets a chance to clean up rather than being hard-killed).

## Install

```
go install github.com/Soldsoul86/AAA/loopkill@latest
```

(Prebuilt binaries once this has real users — for now, `go build` from source.)

## Known limitations

- **Unix-only for `-kill`.** It sends `SIGINT` via `syscall`; Windows signal
  handling works differently and hasn't been ported.
- **stdout only.** Output on stderr isn't analyzed, only passed through —
  most agent CLIs render their main interactive output on stdout, but this
  is a real gap if a tool does otherwise.
- **Heuristic, not semantic.** It matches normalized *text*, not meaning —
  an agent rephrasing the same failed idea in slightly different words each
  time won't be caught. This trades false negatives for a low false-positive
  rate, deliberately.
- **No native Claude Code integration yet.** Claude Code writes structured
  session transcripts to disk, which would make loop detection far more
  reliable than scraping terminal text — but the exact transcript schema
  wasn't confirmed before building this, so it's not implemented. The
  generic wrapper mode above works today for any CLI, Claude Code included;
  a native mode is a planned, not yet built, improvement.

## License

MIT
