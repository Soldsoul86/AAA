# Releasing spare

spare lives in the [Soldsoul86/AAA](https://github.com/Soldsoul86/AAA)
monorepo, alongside the other tools. Its release pipeline is scoped to this
subfolder — tagging one tool never triggers a release for the others.

## One-time setup (shared across all tools, already done for this repo)

1. **Homebrew tap repo** — one tap holds formulas for every tool in this
   monorepo: `Soldsoul86/homebrew-tap`. Already created; nothing to do here
   for a new tool.
2. **`HOMEBREW_TAP_GITHUB_TOKEN`** — a repository secret on the **AAA** repo
   with write access to the tap. Already configured; shared by every
   release workflow in this repo.

## Every release of spare

You push a tag prefixed with the tool name:

```
git tag spare-v0.1.0
git push origin spare-v0.1.0
```

This triggers `.github/workflows/release-spare.yml` only — no other tool's
workflow runs. Internally, the workflow converts this into a real semver
release tag with the tool name as build metadata — `v0.1.0+spare` — so it
can never collide with another tool's version number in this shared repo,
while still resolving correctly for `go install ...@latest`. Within a few
minutes:

- A GitHub Release tagged `v0.1.0+spare` exists with binaries for macOS
  (Intel + Apple Silicon), Linux (amd64 + arm64), and Windows
- `.deb` and `.rpm` packages are attached directly to the release —
  `apt install ./spare_..._amd64.deb` / `dnf install ./spare-...-1.x86_64.rpm`
  work without needing to get into official distro repos
- `checksums.txt` is published alongside everything
- `brew install Soldsoul86/tap/spare` works
- `install.sh` and `go install github.com/Soldsoul86/AAA/spare@latest` both
  resolve to this version

Also push the Go-convention tag separately (not automated by the workflow,
since it uses a different naming scheme Go's module system expects):

```
git tag spare/v0.1.0
git push origin spare/v0.1.0
```

## Testing the build locally without publishing anything

This repo's shared tags across tools confuse goreleaser's own tag-detection
during local snapshot testing — confirmed directly: it picked up another
tool's tag and failed to parse it as semver. Override it explicitly:

```
cd spare
GORELEASER_CURRENT_TAG=v0.1.0+spare goreleaser release --snapshot --clean
```

This runs the full cross-compilation step — including the `.deb`/`.rpm`
packaging — and writes everything to `spare/dist/`, without needing a git
tag or pushing anywhere.

## Verified, not just designed — including the part that's usually skipped

Same release pipeline template proven for the other 6 tools. What's
different for spare specifically: it has its own CI workflow
(`.github/workflows/spare-ci.yml`) that runs on every push, separate from
the release workflow, and actually **executes** the built binary on real
`ubuntu-latest`, `macos-latest`, and `windows-latest` runners — a genuine
end-to-end `rm` diversion and restore, not just a cross-compile check. This
closes a gap every other tool in this repo still has: their Windows and
Linux binaries have only ever been cross-compiled by goreleaser, never
actually run. The first real run of that workflow found and fixed a real
bug (Windows reads `%USERPROFILE%`, not `$HOME` — see the commit history)
that cross-compiling alone would never have surfaced.
