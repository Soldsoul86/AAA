# permit

Diagnoses why a Claude Code permission rule silently doesn't apply.

```
$ permit doctor "Bash(git push origin main)"
✅ workspace trust: accepted
✅ all settings files are valid JSON
🔴 possibly shadowed by deny rule "Bash(git push *)" — deny always wins regardless of specificity
```

## Why

Claude Code's own docs are precise about this, and it's worth reading
directly before assuming a bug: **file-edit "always allow" is documented to
last only until session end, not permanently** — that's by design, not a
bug most people expect. Bash command approvals *are* meant to persist
permanently per-project — Claude Code already does this natively, no tool
needed, the first time you click "yes, don't ask again." When a rule still
doesn't seem to work, the most common documented cause is that
**`permissions.allow` rules are silently ignored until you accept the
workspace trust dialog for that project** — Claude Code reads the rules
but doesn't apply them until then, with no obvious signal that this is
what's happening. That's the specific thing `permit doctor` checks for.

## What this deliberately does *not* do, and why

Earlier versions of this tool also included `permit allow` (write a rule
via CLI) and `permit trust` (skip the trust dialog). Both were removed
after a direct, honest look at what they actually added: **Claude Code
already provides both natively** — clicking "don't ask again" once writes
the same rule, and accepting the trust dialog is one click. Wrapping a
CLI around functionality that already exists as a single click isn't a
fix, it's rework. Keeping them in would have made permit look like it
solves something it doesn't.

What Claude Code does *not* provide natively is a way to find out *why* a
rule you already have isn't working — that diagnostic gap is real, and
that's the entire scope of what's left.

permit also doesn't try to reimplement Claude Code's own permission
engine as a competing decision-maker. That would mean duplicating its real
matching logic (compound command splitting, wrapper stripping, PowerShell
AST parsing, and more) — a large, security-relevant surface no small tool
should casually take on.

## Usage

```
permit doctor            # check trust status and settings JSON validity
permit doctor "<rule>"   # also check whether that rule could be shadowed
```

Rules use Claude Code's real syntax exactly — see
[the official permissions docs](https://code.claude.com/docs/en/permissions)
for the full specification, e.g. `Bash(npm run *)`, `Read(./.env)`,
`WebFetch(domain:example.com)`.

## How the trust check works

`permit doctor` reads `~/.claude.json` and looks up
`projects["<absolute-path>"].hasTrustDialogAccepted` for your project's git
root. This location is **not a documented, stable public API** — it was
found by directly inspecting a real Claude Code config file on a real
machine, and verified against real trusted/untrusted project entries. If
Claude Code changes where it stores this, the check will stop finding
anything (it fails safely — you'll see "no trust record found," not a
false claim either way). permit only ever reads this file, never writes it.

## How the shadow-rule check works — and its real limits

`permit doctor <rule>` checks whether an existing deny/ask rule could
shadow the rule you're asking about, using a **deliberately conservative,
best-effort heuristic**: an exact tool-name match, plus either a bare
tool-name rule (which shadows everything for that tool) or one specifier
being a literal prefix of the other.

This catches the single most common real case — a broad deny like
`Bash(git push *)` shadowing a narrower allow like
`Bash(git push origin main)` — but it is **not** a full reimplementation of
Claude Code's actual matching engine. It will miss real shadowing in cases
involving mid-pattern wildcards (`Bash(git * main)`), compound commands,
or process-wrapper stripping (`timeout`, `nice`, `xargs`, etc.). Treat a
clean `permit doctor` result as "no obvious issue found," not a guarantee.

## Install

```
go install github.com/Soldsoul86/AAA/permit@latest
```

## Known limitations

- **Shadow-rule detection is best-effort**, not exhaustive — see above.
- **The trust-check storage location is unofficial** and may change without notice.
- **Read-only, on purpose.** permit never writes to any Claude Code config file.

## License

MIT
