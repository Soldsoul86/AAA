# again

Counts how many times you've had to repeat yourself to your AI agent this session.

```
$ again watch -claude-code
again: this looks similar (56%) to something you said 2 prompts ago — you've repeated yourself 1 time(s) this session
again: that's a few repeats now — might be worth starting a fresh session instead of re-explaining again
```

## Why

The quiet, cumulative frustration of re-explaining the same constraint to an
agent, over and over, is invisible while it's happening — you just feel
tired. `again` turns it into a number.

## How it works

`again` polls a session transcript file for new user-authored prompts, and
compares each new one against your last 20 prompts using **Jaccard
similarity**: lowercase both, strip punctuation, and measure word-set
overlap. That's it — no ML, no API calls, nothing that can change behavior
silently between versions. `intersection / union`, fully deterministic,
fully unit-tested.

This is deliberately crude, and it has a real, demonstrated limitation
worth knowing about honestly rather than glossing over. In testing:

- Prompt 1: *"please fix the login bug"*
- Prompt 3: *"no that still has the login bug, please fix the login bug"* — correctly flagged, 56% similar to prompt 1
- Prompt 4: *"the login bug is still not fixed, can you fix the login bug"* — **not flagged**. Its similarity to prompt 1 had drifted to 36%, below the default 50% threshold, even though a human would clearly read it as the same complaint a third time.

Each successive rephrasing drifts a little further from the original
wording, and this heuristic compares against the literal words used, not
the underlying ask. If you find it's under-catching real repeats, lower
`-threshold` (0.35–0.4 catches more, at the cost of more false positives on
genuinely different-but-topically-related prompts).

## Usage

```
again watch -file PATH [-threshold F] [-nudge N] [-interval MS]
again watch -claude-code [-threshold F] [-nudge N] [-interval MS]
again report
```

- **`-file PATH`** watches a JSONL transcript directly.
- **`-claude-code`** is the same best-effort auto-detection used by
  [ctxmeter](https://github.com/Soldsoul86/AAA/tree/main/ctxmeter) — an *observed*,
  not officially documented, path pattern
  (`~/.claude/projects/<this-project>/*.jsonl`). If your Claude Code
  version stores things differently, this will find nothing and tell you
  so — use `-file` directly if you know the right path.
- **`-threshold`** (default 0.5) — similarity score above which two prompts count as "the same ask."
- **`-nudge`** (default 3) — print a suggestion to start fresh after this many repeats.

## Measuring what it actually costs — not what it "saves"

```
$ again report
again: 3 repeated prompt(s) detected across all tracked sessions
again: ~412 tokens measured in those repeated prompts alone
```

Every detected repeat is logged to `~/.again/savings.jsonl` with the
**real, measured token count of that specific repeated prompt** — using
the same rough ~4-characters-per-token approximation as
[ctxmeter](https://github.com/Soldsoul86/AAA/tree/main/ctxmeter), not a
real tokenizer. `again report` sums this across every session you've ever
watched, cumulatively, stored locally, never uploaded anywhere.

Deliberately **not** called "tokens saved." Saved implies a counterfactual
— what would have happened if again weren't installed — and that's not
something this tool (or any tool without a time machine) can actually
know. What it reports is real and verifiable: tokens spent on prompts
that were, by the same measure the live warning uses, substantially
repeating something already said. What you do with that number — restart
sooner, write a project rule instead, ignore it — is up to you.

## Extracting user turns

Session log schemas vary and can change without notice, so `again` tries a
couple of commonly-seen shapes defensively:

- `{"role":"user","content":"..."}` (plain string content)
- `{"role":"user","content":[{"type":"text","text":"..."}]}` (block-array
  content — matches the Anthropic Messages API shape; `tool_result` and
  other non-text blocks are ignored on purpose)
- `{"type":"user","message":{"role":"user","content":...}}` (a nested
  wrapper some session-log formats use)

All three shapes are unit tested against synthetic fixtures. **None of
these have been verified against a real Claude Code transcript file** — if
`again` never detects anything, that's the most likely reason. Please open
an issue with one redacted sample line rather than assume it's silently
working.

## Install

```
go install github.com/Soldsoul86/AAA/again@latest
```

## Known limitations

- Compares word overlap, not meaning — see the worked example above.
- Read-only, polling-based — it does not sit inside your live keystrokes
  (a truly real-time version would need a pseudo-terminal wrapper, which
  risks breaking a TUI agent's raw-mode input handling if done carelessly;
  that's a real next step, not yet built).
- Transcript schema detection is best-effort, unverified against a real file.

## License

MIT
