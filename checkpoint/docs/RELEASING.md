# Releasing checkpoint

checkpoint lives in the [Soldsoul86/AAA](https://github.com/Soldsoul86/AAA)
monorepo, alongside three other tools. Its release pipeline is scoped to
this subfolder — tagging one tool never triggers a release for the others.

## One-time setup (shared across all four tools, do this once)

1. **Create the Homebrew tap repo** — one tap holds formulas for all four
   tools, so this is a single repo, not one per tool:
   ```
   gh repo create Soldsoul86/homebrew-tap --public
   ```

2. **Create a Personal Access Token** with `repo` scope (classic token, or a
   fine-grained token scoped to just the `homebrew-tap` repo). Add it as a
   repository secret named `HOMEBREW_TAP_GITHUB_TOKEN` in the **AAA** repo's
   settings (Settings → Secrets and variables → Actions) — one secret,
   shared by all four release workflows since they all live in this repo.

## Every release of checkpoint after that

You push a tag prefixed with the tool name — that's what triggers the
right workflow and nothing else:

```
git tag checkpoint-v0.1.2
git push origin checkpoint-v0.1.2
```

This triggers `.github/workflows/release-checkpoint.yml` only — the other three
tools' workflows won't run. Internally, the workflow converts this into a real
semver release tag with the tool name as build metadata —
`v0.1.2+checkpoint` — so it can never collide with another tool's version
number in this shared repo, while still resolving correctly for
`go install ...@latest`. Within a few minutes:
- A GitHub Release tagged `v0.1.2+checkpoint` exists with binaries for macOS
  (Intel + Apple Silicon), Linux (amd64 + arm64), and Windows
- `checksums.txt` is published alongside them
- `brew install Soldsoul86/tap/checkpoint` works
- `install.sh` and `go install github.com/Soldsoul86/AAA/checkpoint@latest` both resolve to this version

Also push the Go-convention tag separately (this one is not automated by
the workflow, since it uses a different naming scheme Go's module system
expects):

```
git tag checkpoint/v0.1.2
git push origin checkpoint/v0.1.2
```

## Testing the build locally without publishing anything

```
cd checkpoint
goreleaser release --snapshot --clean
```

This runs the full cross-compilation step and writes everything to
`checkpoint/dist/`, without needing a git tag or pushing anywhere.

## Verified, not just designed

This pipeline has been run end to end successfully twice (v0.1.0 and
v0.1.1) — including a real upgrade test (`brew upgrade`), three full
uninstall/reinstall cycles with no leftover state, and `go install@latest`
correctly resolving to the newest version. It got there by fixing two real
bugs found live: an invented goreleaser config field that doesn't actually
exist, and a cross-tool version-collision bug in the original tag scheme.
Both are fixed in what's described above — this is no longer a
"should work" pipeline, it's a "has worked, twice" one.
