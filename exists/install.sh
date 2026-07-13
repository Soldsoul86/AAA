#!/bin/sh
# Installs the latest release of exists from GitHub Releases.
# Usage: curl -fsSL https://raw.githubusercontent.com/Soldsoul86/AAA/main/exists/install.sh | sh
#
# This repo holds several tools. Each tool's release is uniquely tagged
# "vX.Y.Z+exists" (semver build metadata, so it can never collide with
# another tool's version number in this shared repo) — this script finds
# that tag, then builds the plain "<tool>_X.Y.Z_<os>_<arch>.tar.gz"
# filename goreleaser actually produces (build metadata is deliberately
# left out of filenames, only used to keep the release tag unique).
set -e

REPO="Soldsoul86/AAA"
BINARY="exists"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "error: unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux|darwin) ;;
  *) echo "error: unsupported OS: $os — Windows users, download the .zip from the Releases page instead" >&2; exit 1 ;;
esac

# Full release tag, e.g. "v0.1.0+exists"
tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' \
  | grep -- "+${BINARY}$" \
  | head -1)
[ -n "$tag" ] || { echo "error: no release found for $BINARY (looked for a tag ending in +${BINARY})" >&2; exit 1; }

# Strip the leading "v" and the "+tool" suffix to get the plain version
# used in the archive filename: "v0.1.0+exists" -> "0.1.0"
plain_version=$(echo "$tag" | sed -E "s/^v//; s/\+${BINARY}$//")

archive="${BINARY}_${plain_version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPO/releases/download/${tag}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "downloading ${archive} (${tag})..."
curl -fsSL "${base_url}/${archive}" -o "$tmp/${archive}"
curl -fsSL "${base_url}/checksums.txt" -o "$tmp/checksums.txt"

expected=$(grep " ${archive}$" "$tmp/checksums.txt" | awk '{print $1}')
if command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmp/${archive}" | awk '{print $1}')
else
  actual=$(sha256sum "$tmp/${archive}" | awk '{print $1}')
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
  echo "error: checksum verification failed — refusing to install" >&2
  exit 1
fi
echo "checksum verified"

tar -xzf "$tmp/${archive}" -C "$tmp"

if [ -w "$INSTALL_DIR" ]; then
  mv "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
else
  echo "installing to $INSTALL_DIR requires sudo"
  sudo mv "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "$BINARY installed to $INSTALL_DIR/$BINARY"
