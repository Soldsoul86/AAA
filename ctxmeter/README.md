# ctxmeter

A live context-window gauge for any AI coding CLI.

```
$ claude | ctxmeter watch -max 200000
[████████░░░░░░░░░░░░]  41% (~82,431 / 200,000 est. tokens)
```

Nobody knows how close they are to compaction until it happens and wipes
their working memory. ctxmeter is a small, always-visible answer to "how
full am I right now."

## Usage

```
ctxmeter watch -file PATH [-max N] [-interval MS]
ctxmeter watch -claude-code [-max N] [-interval MS]
<some-cli> | ctxmeter watch [-max N]
```

- **`-file PATH`** tails a log or transcript file and estimates cumulative content size.
- **`-claude-code`** is a best-effort auto-detector: it looks for the most
  recently modified `.jsonl` file under `~/.claude/projects/<this project>/`.
  This path is based on an *observed* pattern, not a documented, guaranteed
  API — if it doesn't find anything, it tells you so and falls back to
  reading stdin. Use `-file` directly if you know the right path.
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
- **`-claude-code` auto-detection is best-effort.** The project-directory
  path pattern was observed, not confirmed against official documentation.
  If Claude Code changes its storage layout, this will silently stop
  finding anything and fall back to stdin — it won't crash, but it also
  won't warn you loudly beyond the one startup message.
- **No native support yet for Cursor, Codex, Aider, or Gemini CLI** beyond
  the generic stdin-piping mode, which works with any of them but only
  estimates what actually gets printed to the terminal, not necessarily
  everything in the model's real context.

## License

MIT
