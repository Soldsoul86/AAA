# permit

Makes Claude Code permission rules easy to write correctly, and easy to
diagnose when they silently don't apply.

```
$ permit allow "Bash(npm run *)"
permit: added "Bash(npm run *)" to .claude/settings.json

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
permanently per-project, and when they don't, the most common documented
cause is that **`permissions.allow` rules are silently ignored until you
accept the workspace trust dialog for that project** — Claude Code reads
the rules but doesn't apply them until then, with no obvious signal that
this is what's happening.

permit doesn't try to fix Claude Code's permission engine or compete with
it — that would mean reimplementing its real matching logic (compound
command splitting, wrapper stripping, PowerShell AST parsing, and more),
which is a large, security-relevant surface no small tool should casually
duplicate. Instead, permit helps you **write correct, permanent rules using
Claude Code's own real syntax**, and **checks the documented reasons** a
rule might not be working.

## Commands

| Command | What it does |
|---|---|
| `permit allow <rule> [--user]` | Adds a permanent rule to `permissions.allow` — project-level by default, `--user` for `~/.claude/settings.json` |
| `permit list` | Shows the merged allow/ask/deny rules across every settings source (local, project, user) — precedence spans multiple files, so seeing only one is how a shadowing rule gets missed |
| `permit doctor [rule]` | Checks workspace trust status, settings JSON validity, and (if a rule is given) whether an existing deny/ask rule could be shadowing it |

Rules use Claude Code's real syntax exactly — see
[the official permissions docs](https://code.claude.com/docs/en/permissions)
for the full specification. Examples:

```
permit allow "Bash(npm run *)"
permit allow "Bash(git commit *)"
permit allow "Read(./.env)"
permit allow "WebFetch(domain:example.com)"
```

## How the trust check works

`permit doctor` reads `~/.claude.json` and looks up
`projects["<absolute-path>"].hasTrustDialogAccepted` for your project's git
root. This location is **not a documented, stable public API** — it was
found by directly inspecting a real Claude Code config file on a real
machine, and verified against real trusted/untrusted project entries. If
Claude Code changes where it stores this, the check will stop finding
anything (it fails safely — you'll see "no trust record found," not a
false claim either way).

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
- **No hook-based auto-approval.** An earlier design for this tool
  attempted to build a PreToolUse hook that would make its own allow/deny
  decisions on every tool call. That was deliberately abandoned: it would
  have meant re-implementing Claude Code's real permission-matching logic
  independently, with real risk of subtly disagreeing with it — too risky
  for a tool making security-relevant decisions, and genuinely out of
  scope for "one tiny annoyance, done well."

## License

MIT
