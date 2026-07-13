# Releasing exists

exists lives in the [Soldsoul86/AAA](https://github.com/Soldsoul86/AAA)
monorepo, alongside the other tools. Its release pipeline is scoped to this
subfolder — tagging one tool never triggers a release for the others.

## One-time setup (shared across all tools, already done for this repo)

1. **Homebrew tap repo** — one tap holds formulas for every tool in this
   monorepo: `Soldsoul86/homebrew-tap`. Already created; nothing to do here
   for a new tool.

2. **`HOMEBREW_TAP_GITHUB_TOKEN`** — a repository secret on the **AAA** repo
   with write access to the tap. Already configured; shared by every
   release workflow in this repo.

## Every release of exists

You push a tag prefixed with the tool name — that's what triggers the
right workflow and nothing else:

```
git tag exists-v0.1.0
git push origin exists-v0.1.0
```

This triggers `.github/workflows/release-exists.yml` only — no other
tool's workflow runs. Internally, the workflow converts this into a real
semver release tag with the tool name as build metadata — `v0.1.0+exists`
— so it can never collide with another tool's version number in this
shared repo, while still resolving correctly for `go install ...@latest`.
Within a few minutes:
- A GitHub Release tagged `v0.1.0+exists` exists with binaries for macOS
  (Intel + Apple Silicon), Linux (amd64 + arm64), and Windows
- `checksums.txt` is published alongside them
- `brew install Soldsoul86/tap/exists` works
- `install.sh` and `go install github.com/Soldsoul86/AAA/exists@latest` both resolve to this version

Also push the Go-convention tag separately (this one is not automated by
the workflow, since it uses a different naming scheme Go's module system
expects):

```
git tag exists/v0.1.0
git push origin exists/v0.1.0
```

## Testing the build locally without publishing anything

This repo's shared tags across tools confuse goreleaser's own tag-detection
during local snapshot testing — confirmed directly: it picked up another
tool's tag and failed to parse it as semver. Override it explicitly:

```
cd exists
GORELEASER_CURRENT_TAG=v0.1.0+exists goreleaser release --snapshot --clean
```

This runs the full cross-compilation step and writes everything to
`exists/dist/`, without needing a git tag or pushing anywhere.

## Verified, not just designed

Same pipeline template that's been run end to end successfully for
checkpoint, loopkill, ctxmeter, again, permit, and actually — real GitHub
Releases, real `brew install`, real `install.sh` checksum-verified
downloads. exists's own code has an extra layer of real-world verification
beyond that: its unit tests make live calls to the real npm and PyPI
registries rather than mocking them, and the finished binary was run
end-to-end against a synthetic transcript containing both a real and a
hallucinated package before shipping — see the README.
