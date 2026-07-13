# Releasing permit

permit lives in the [Soldsoul86/AAA](https://github.com/Soldsoul86/AAA)
monorepo. Its release pipeline is scoped to this subfolder and uses the
same proven pattern as checkpoint (verified end to end across two real
releases before permit was built) — tagging permit never triggers a
release for the other tools.

## One-time setup

Already done once, shared across every tool in this repo — nothing to
repeat here. See checkpoint's `docs/RELEASING.md` if you're setting this
up for the very first time.

## Every release of permit

```
git tag permit-v0.1.0
git push origin permit-v0.1.0
```

This triggers `.github/workflows/release-permit.yml` only. Internally, the
workflow converts this into `v0.1.0+permit` (semver build metadata) so it
can't collide with another tool's version number in this shared repo.
Within a few minutes:
- A GitHub Release tagged `v0.1.0+permit` exists with binaries for macOS,
  Linux, and Windows
- `brew install Soldsoul86/tap/permit` works
- `install.sh` and `go install github.com/Soldsoul86/AAA/permit@latest` resolve to this version

Also push the separate Go-convention tag (not automated — different naming
scheme than the release trigger, per Go's own module versioning rules for
subdirectory modules):

```
git tag permit/v0.1.0
git push origin permit/v0.1.0
```

## Testing the build locally without publishing anything

```
cd permit
goreleaser release --snapshot --clean
```

## Status

Built directly on the proven, twice-verified checkpoint release pattern —
no new release-pipeline risk introduced by this tool specifically.
