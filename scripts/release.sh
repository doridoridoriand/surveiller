#!/bin/bash

# Simple release script for deadman-go
# Usage: ./scripts/release.sh v0.0.1

set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.0.1"
    exit 1
fi

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    echo "Error: Version must be in format vX.Y.Z or vX.Y.Z-suffix"
    echo "Examples: v0.0.1, v1.0.0, v1.0.0-beta1"
    exit 1
fi

echo "🚀 Preparing release $VERSION"

# Check if we're on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "⚠️  Warning: You're not on the main branch (current: $CURRENT_BRANCH)"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo "❌ Working directory is not clean. Please commit or stash changes."
    git status --short
    exit 1
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "❌ Tag $VERSION already exists"
    exit 1
fi

# Update version in main.go
echo "📝 Updating version in main.go"
sed -i.bak "s/const version = \".*\"/const version = \"${VERSION#v}\"/" main.go
rm main.go.bak

# Run tests
echo "🧪 Running tests"
make test-all

# Build to verify
echo "🔨 Building to verify"
make build

# Show changes
echo "📋 Changes to be committed:"
git diff --name-only

# Commit version update
echo "💾 Committing version update"
git add main.go
git commit -m "Release $VERSION"

# Create and push tag
echo "🏷️  Creating tag $VERSION"
git tag -a "$VERSION" -m "Release $VERSION"

echo "📤 Pushing changes and tag"
git push origin main
git push origin "$VERSION"

echo "✅ Release $VERSION has been created!"
echo "🔗 Check the release at: https://github.com/doridoridoriand/deadman-go/releases/tag/$VERSION"
echo "⏳ GitHub Actions will build and publish the binaries automatically."