# exists

Checks whether a package your AI agent just installed is a name that
actually exists in the real registry, right now — catching a hallucinated
package name before you find out from a build failure, or worse, before
someone squats that exact name later and it silently starts resolving.

```
$ exists watch -claude-code
exists: watching /Users/you/.claude/projects/.../session.jsonl
exists: "super-fake-hallucinated-package-that-does-not-exist-42" was just
installed from pypi, but no package by that name exists in the real
registry — check for a typo or a hallucinated name before trusting this
```

## Why

LLMs occasionally invent a plausible-sounding package name — close enough
to something real that nobody double-checks. It used to just mean a build
failure and a wasted few minutes. It doesn't stop there anymore: attackers
watch for exactly this pattern and register the invented name with real,
malicious code (slopsquatting), betting that someone's agent — or someone
copying the agent's output — will install it without checking.

Be precise about what that means `exists` can and can't catch: it checks
whether a name exists *right now*, at the moment the agent installs it. If
the hallucinated name is genuinely unclaimed, `exists` catches it — that's
the common case, and the main value here. If an attacker has *already*
registered that exact name with malicious code before your agent tries it,
the package now legitimately exists, and `exists` will correctly say so —
it cannot tell a real package from a successfully squatted one, because
from the registry's point of view there's no difference. This tool closes
the "hallucinated name that isn't claimed by anyone" gap. It is not a
malware scanner, and shouldn't be described or relied on as one.

## How it works

`exists` watches a Claude Code session transcript for shell commands that
add a new dependency — `npm install`/`yarn add`/`pnpm add`, `pip install`/
`pip3 install`/`poetry add`/`uv add` — extracts the package name(s), and
makes a real HTTP call to the actual registry (`registry.npmjs.org` or
`pypi.org`) to check it exists. A 404 means it doesn't. There's no local
cache or guesswork involved: this is the same question you'd ask by hand.

This is the only tool in this collection that needs network access — every
other tool here is a local, offline text processor. Worth knowing before
you install it.

## Usage

```
exists watch -file PATH [-interval MS]
exists watch -claude-code [-interval MS]
```

- **`-file PATH`** watches a JSONL transcript directly.
- **`-claude-code`** scans every project directory under
  `~/.claude/projects/` and watches whichever `.jsonl` file was modified
  most recently — same approach used by
  [actually](https://github.com/Soldsoul86/AAA/tree/main/actually), chosen
  after confirming that guessing a single folder from the current working
  directory is unreliable.

If the path given to `-file` doesn't exist yet, exists keeps polling
rather than exiting — useful if you start watching before the file is
created — but says so explicitly after 5 consecutive misses rather than
running forever with no output. A third-party test of a typo'd path
confirmed the old behavior looked identical to "working, nothing to
report" with zero feedback; this warning is the fix.

## Known limitations

- **v1 covers npm and PyPI only.** These are where the best-evidenced
  hallucination and slopsquatting harm concentrates — most published
  research and incident reports on this problem are npm/PyPI-specific.
  Go modules were deliberately left out: a Go import is a real VCS path
  that `go build` already refuses to resolve if it doesn't exist, which is
  a structurally different (and already somewhat self-correcting)
  situation from a registry where anyone can publish any name. crates.io,
  Maven, and RubyGems are real gaps, not built yet.
- **A fixed list of install commands.** `npm install`/`i`, `yarn add`,
  `pnpm add`, `pip`/`pip3 install`, `poetry add`, `uv add`. A custom
  install wrapper script won't be recognized — precise list over loose
  guessing, same reasoning as every other command-detection choice in this
  collection.
- **Checks that the base package exists, not the specific version pinned.**
  `requests==99.99.99` (a real package, a version that doesn't exist)
  won't be flagged in v1 — only a package name that doesn't exist at all.
- **A handful of value-taking flags are recognized** (`--index-url`,
  `--group`, `--registry`, etc.) so their argument isn't misparsed as a
  second package name — a real bug caught during testing
  (`poetry add --group dev fastapi` was initially treating `dev` as a
  package to check). The list isn't exhaustive; an uncommon flag outside
  it could still misfire this way.
- **A failed check is reported distinctly from a missing package.** If the
  registry can't be reached or times out, you get "couldn't verify," not a
  false "doesn't exist" — same posture as
  [actually](https://github.com/Soldsoul86/AAA/tree/main/actually)'s
  `unknown` verdict for unrecognized test output.
- **Registry existence checks are real network calls in this project's own
  test suite**, not mocked — deliberately, since the entire point is
  whether the real registry agrees. If you're offline, `go test ./...`
  will fail on `internal/registry` and `internal/watch`.
- **Always checks the public registry, never a private one.** If your
  project installs from a private npm scope or an internal PyPI mirror
  (common at companies with their own package feeds), `exists` will report
  a real, legitimate private package as "doesn't exist," because it only
  ever asks `registry.npmjs.org`/`pypi.org` — it has no way to know about,
  or check, whatever registry your `.npmrc`/`pip.conf` actually points at.
  If you rely on a private registry, expect false "missing" reports for
  every package that lives only there. Not yet built: reading the actual
  configured registry instead of assuming the public one.
- **Can only see a package that doesn't exist yet — not a malicious one
  that does.** See Why above: a successfully slopsquatted package (already
  registered under the hallucinated name) will correctly report as
  existing. `exists` narrows the window where a hallucinated name is
  silently trusted; it does not replace scanning what a package actually
  contains.

## Verified against real data

Confirmed npm and PyPI's actual existence-check response shapes via curl
before writing any code, then wrote unit tests that make live calls to
both real registries rather than mocking them. After shipping v0.1.0, a
separate black-box pass installed the tool fresh via `go install` and
drove it through every documented command style end to end via the real
CLI — `npm install`, `yarn add`, `pnpm add`, `pip install`, `poetry add`
(including the `--group dev` flag-value edge case), and `uv add`, each
with a mix of real and hallucinated package names in the same transcript —
plus true live-append watching and a typo'd `-file` path. That pass is
what found and fixed the silent-hang-on-missing-file bug described above.

A follow-up pre-release audit tested a transcript line larger than the
scanner's buffer cap — the same class of bug found and fixed in
[actually](https://github.com/Soldsoul86/AAA/tree/main/actually), which
shares this watch-loop shape. Fixed the same way: an unbounded line
reader that tracks bytes consumed instead of a fixed-size buffer, so one
oversized line can no longer silently take down detection for the rest of
the session.

## Install

```
go install github.com/Soldsoul86/AAA/exists@latest
```

(Prebuilt binaries via `install.sh` and Homebrew once this has a tagged release.)

## License

MIT
