#!/usr/bin/env bash
# publish-aur.sh — Publish release to Arch User Repository.
#
# Run from CI after GoReleaser completes. Updates the AUR PKGBUILD
# with the new version and checksums, then pushes to the AUR git repo.
#
# Required env vars:
#   AUR_SSH_KEY    — SSH private key with push access to AUR
#   RELEASE_TAG    — The release tag (e.g., v1.2.3)
#
# TODO: Replace CLI_NAME and AUR_PACKAGE with your values.

set -euo pipefail

CLI_NAME="${CLI_NAME:-mycli}"
AUR_PACKAGE="${AUR_PACKAGE:-${CLI_NAME}-bin}"
VERSION="${RELEASE_TAG#v}"

: "${AUR_SSH_KEY:?AUR_SSH_KEY is required}"
: "${RELEASE_TAG:?RELEASE_TAG is required}"

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

# Set up SSH for AUR
mkdir -p ~/.ssh
echo "$AUR_SSH_KEY" > ~/.ssh/aur
chmod 600 ~/.ssh/aur
cat >> ~/.ssh/config <<SSH
Host aur.archlinux.org
  IdentityFile ~/.ssh/aur
  StrictHostKeyChecking accept-new
SSH

# Clone AUR package
echo "Cloning AUR package ${AUR_PACKAGE}..."
git clone "ssh://aur@aur.archlinux.org/${AUR_PACKAGE}.git" "$WORK_DIR/aur"
cd "$WORK_DIR/aur"

# Download checksums from GitHub release
CHECKSUMS_URL="https://github.com/basecamp/${CLI_NAME}-cli/releases/download/${RELEASE_TAG}/checksums.txt"
echo "Fetching checksums from ${CHECKSUMS_URL}..."
curl -fsSL "$CHECKSUMS_URL" -o checksums.txt

# Extract checksums for Linux archives
AMD64_SHA=$(grep "linux_amd64" checksums.txt | awk '{print $1}')
ARM64_SHA=$(grep "linux_arm64" checksums.txt | awk '{print $1}')

if [[ -z "$AMD64_SHA" || -z "$ARM64_SHA" ]]; then
  echo "Error: could not extract checksums from release"
  exit 1
fi

# Update PKGBUILD
cat > PKGBUILD <<PKGBUILD
# Maintainer: 37signals <support@37signals.com>
pkgname=${AUR_PACKAGE}
pkgver=${VERSION}
pkgrel=1
pkgdesc="${CLI_NAME} CLI"
arch=('x86_64' 'aarch64')
url="https://github.com/basecamp/${CLI_NAME}-cli"
license=('MIT')
provides=('${CLI_NAME}')
conflicts=('${CLI_NAME}')

source_x86_64=("\${url}/releases/download/v\${pkgver}/${CLI_NAME}_\${pkgver}_linux_amd64.tar.gz")
source_aarch64=("\${url}/releases/download/v\${pkgver}/${CLI_NAME}_\${pkgver}_linux_arm64.tar.gz")
sha256sums_x86_64=('${AMD64_SHA}')
sha256sums_aarch64=('${ARM64_SHA}')

package() {
  install -Dm755 ${CLI_NAME} "\${pkgdir}/usr/bin/${CLI_NAME}"
}
PKGBUILD

# Generate .SRCINFO
makepkg --printsrcinfo > .SRCINFO

# Commit and push
git add PKGBUILD .SRCINFO
if git diff --cached --quiet; then
  echo "No changes to AUR package"
  exit 0
fi

git config user.name "${CLI_NAME}-cli[bot]"
git config user.email "${CLI_NAME}-cli[bot]@users.noreply.github.com"
git commit -m "Update to ${VERSION}"
git push origin master

echo "AUR package updated to ${VERSION}"
