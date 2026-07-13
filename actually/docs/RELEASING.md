# Releasing actually

actually lives in the [Soldsoul86/AAA](https://github.com/Soldsoul86/AAA)
monorepo, alongside the other tools. Its release pipeline is scoped to this
subfolder — tagging one tool never triggers a release for the others.

## One-time setup (shared across all tools, already done for this repo)

1. **Homebrew tap repo** — one tap holds formulas for every tool in this
   monorepo: `Soldsoul86/homebrew-tap`. Already created; nothing to do here
   for a new tool.

2. **`HOMEBREW_TAP_GITHUB_TOKEN`** — a repository secret on the **AAA** repo
   with write access to the tap. Already configured; shared by every
   release workflow in this repo.

## Every release of actually

You push a tag prefixed with the tool name — that's what triggers the
right workflow and nothing else:

```
git tag actually-v0.1.0
git push origin actually-v0.1.0
```

This triggers `.github/workflows/release-actually.yml` only — no other
tool's workflow runs. Internally, the workflow converts this into a real
semver release tag with the tool name as build metadata — `v0.1.0+actually`
— so it can never collide with another tool's version number in this
shared repo, while still resolving correctly for `go install ...@latest`.
Within a few minutes:
- A GitHub Release tagged `v0.1.0+actually` exists with binaries for macOS
  (Intel + Apple Silicon), Linux (amd64 + arm64), and Windows
- `checksums.txt` is published alongside them
- `brew install Soldsoul86/tap/actually` works
- `install.sh` and `go install github.com/Soldsoul86/AAA/actually@latest` both resolve to this version

Also push the Go-convention tag separately (this one is not automated by
the workflow, since it uses a different naming scheme Go's module system
expects):

```
git tag actually/v0.1.0
git push origin actually/v0.1.0
```

## Testing the build locally without publishing anything

```
cd actually
goreleaser release --snapshot --clean
```

This runs the full cross-compilation step and writes everything to
`actually/dist/`, without needing a git tag or pushing anywhere.

## Verified, not just designed

This is the same pipeline template that's been run end to end successfully
for checkpoint, loopkill, ctxmeter, again, and permit — real GitHub Releases,
real `brew install`, real `install.sh` checksum-verified downloads. No
pipeline-specific risk expected here; the only new thing about this release
is actually's own code, which has been checked against real data separately
(see the README).
