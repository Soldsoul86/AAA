#!/bin/sh
# Installs the latest release of loopkill from GitHub Releases.
# Usage: curl -fsSL https://raw.githubusercontent.com/Soldsoul86/AAA/main/loopkill/install.sh | sh
#
# This repo holds four tools, so releases are tag-prefixed per tool
# (e.g. "loopkill-v0.1.0"), not a single repo-wide "latest" release —
# this script filters for that prefix rather than trusting /releases/latest.
set -e

REPO="Soldsoul86/AAA"
BINARY="loopkill"
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

version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' \
  | grep "^${BINARY}-v" \
  | head -1)
[ -n "$version" ] || { echo "error: no release found for $BINARY (looked for a tag starting with ${BINARY}-v)" >&2; exit 1; }

archive="${BINARY}_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPO/releases/download/${version}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "downloading ${archive} (${version})..."
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
