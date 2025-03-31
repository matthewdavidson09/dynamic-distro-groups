#!/bin/bash
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: ./release.sh v1.0.0"
  exit 1
fi

APP_NAME="dynamic-distro-groups"
DIST_DIR="dist"
REPO="matthewdavidson09/dynamic-distro-groups" # Update this to your GitHub repo

# Clean and make dist dir
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "🧱 Building binaries for $VERSION..."

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$DIST_DIR/${APP_NAME}-linux-amd64"

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "$DIST_DIR/${APP_NAME}-windows-amd64.exe"

# Zip up
cd "$DIST_DIR"
zip "${APP_NAME}-${VERSION}-linux-amd64.zip" "${APP_NAME}-linux-amd64"
zip "${APP_NAME}-${VERSION}-windows-amd64.zip" "${APP_NAME}-windows-amd64.exe"
cd ..

echo "📦 Binaries built and zipped in $DIST_DIR"

echo "🚀 Creating GitHub release and uploading assets..."

gh release create "$VERSION" \
  --repo "$REPO" \
  --title "$VERSION" \
  --notes "Automated release of $VERSION" \
  "$DIST_DIR"/*.zip

echo "✅ Release $VERSION published!"
