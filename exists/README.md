# exists

Checks whether a package your AI agent just installed actually exists in
the real registry — before the build breaks, or worse, before something
with that exact hallucinated name (registered on purpose, after the fact)
gets pulled in and trusted.

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
copying the agent's output — will install it without checking. `exists`
checks the moment a package is installed, not after.

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

## Install

```
go install github.com/Soldsoul86/AAA/exists@latest
```

(Prebuilt binaries via `install.sh` and Homebrew once this has a tagged release.)

## License

MIT
