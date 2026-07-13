# Releasing loopkill

loopkill lives in the [Soldsoul86/AAA](https://github.com/Soldsoul86/AAA)
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

## Every release of loopkill after that

Tags are prefixed per tool, since one repo holds four release pipelines:

```
git tag loopkill-v0.1.0
git push origin loopkill-v0.1.0
```

This triggers `.github/workflows/release-loopkill.yml` only — the other three
tools' workflows won't run. Within a few minutes:
- A GitHub Release tagged `loopkill-v0.1.0` exists with binaries for macOS
  (Intel + Apple Silicon), Linux (amd64 + arm64), and Windows
- `checksums.txt` is published alongside them
- `brew install Soldsoul86/tap/loopkill` works
- `install.sh` and `go install github.com/Soldsoul86/AAA/loopkill@latest` both resolve to this version

## Testing the build locally without publishing anything

```
cd loopkill
goreleaser release --snapshot --clean
```

This runs the full cross-compilation step and writes everything to
`loopkill/dist/`, without needing a git tag or pushing anywhere.

## Honest caveat

The monorepo-specific wiring here (tag-prefix triggers, the
`GORELEASER_CURRENT_TAG` override so goreleaser doesn't get confused by
other tools' tags) was written based on goreleaser's documented behavior,
but has **not** been run end to end — network restrictions in the
environment this was built in prevented a live test. Run the local
`--snapshot` command above before your first real tag, and treat the first
real release as the actual first test of this pipeline.
