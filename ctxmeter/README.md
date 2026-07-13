# ctxmeter

A live context-window gauge for AI coding CLIs that don't already have one.

```
$ claude | ctxmeter watch -max 200000
[████████░░░░░░░░░░░░]  41% (~82,431 / 200,000 est. tokens)
```

Nobody knows how close they are to compaction until it happens and wipes
their working memory. ctxmeter is a small, always-visible answer to "how
full am I right now."

## If you're using Claude Code, use `/statusline` instead

Claude Code has a native `/statusline` that can show context usage directly
in its own UI, with no piping, no separate process, and access to real
internal state rather than an outside estimate. If you're only ever running
Claude Code, set that up and skip this tool — it will do a more accurate job
than ctxmeter's character-count approximation ever can.

ctxmeter earns its place for everything `/statusline` doesn't reach: Cursor,
Codex, Aider, Gemini CLI, or any other CLI agent with no built-in context
gauge. The `-claude-code` flag below exists mainly so the same binary works
uniformly across tools if you're switching between several — not as the
primary reason to install it.

## Usage

```
ctxmeter watch -file PATH [-max N] [-interval MS]
ctxmeter watch -claude-code [-max N] [-interval MS]
<some-cli> | ctxmeter watch [-max N]
```

- **`-file PATH`** tails a log or transcript file and estimates cumulative content size.
- **`-claude-code`** scans every project directory under
  `~/.claude/projects/` and watches whichever `.jsonl` file was modified
  most recently. Earlier versions instead encoded the current working
  directory and looked up a single matching folder — that was wrong,
  confirmed against a real session where the process's cwd didn't match
  the directory Claude Code actually keyed the session under. Scanning for
  "whichever session is active right now" sidesteps that. If it doesn't
  find anything, it tells you so and falls back to reading stdin. Use
  `-file` directly if you know the right path.
- **Piped mode** (no flags) reads stdin and estimates as it goes — and
  passes everything through to stdout completely unmodified, so wrapping a
  CLI in a pipeline never swallows its output. Verified: the gauge renders
  on stderr, the original output still reaches stdout untouched.

## How the estimate works

There's no real tokenizer here — reimplementing every model's exact
tokenizer would require vendoring a vocabulary file per model and would go
stale immediately. Instead ctxmeter uses the standard rough approximation
of **~4 characters of text ≈ 1 token**, applied to:

- the raw text of a line, if it isn't JSON, or
- the sum of every *string value* in a line, if it parses as JSON —
  recursively, regardless of field names. This means ctxmeter doesn't need
  to know a tool's exact log schema to produce a reasonable estimate; it
  just adds up how much text content is in each entry.

Treat the number as "roughly how full", not an exact count.

## Install

```
go install github.com/Soldsoul86/AAA/ctxmeter@latest
```

## Known limitations

- **Not an exact token count.** See above — this is a proportional estimate, not the real number your provider bills or limits against.
- **`-claude-code` auto-detection is best-effort.** It picks the most
  recently modified session file across every project directory, which is
  usually right but isn't a documented, guaranteed API. If Claude Code
  changes its storage layout, this will silently stop finding anything and
  fall back to stdin — it won't crash, but it also won't warn you loudly
  beyond the one startup message. If you have multiple Claude Code windows
  open at once, it watches whichever one was most recently active, not
  necessarily the one you care about — use `-file` to pin a specific
  session.
- **No native support yet for Cursor, Codex, Aider, or Gemini CLI** beyond
  the generic stdin-piping mode, which works with any of them but only
  estimates what actually gets printed to the terminal, not necessarily
  everything in the model's real context.

## License

MIT
