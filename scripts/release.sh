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

# Get current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo "📍 Current branch: $CURRENT_BRANCH"

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

# Check if we have the latest changes
echo "🔄 Fetching latest changes..."
git fetch origin

# Switch to main branch and ensure it's up to date
echo "🔀 Switching to main branch..."
git checkout main
git pull origin main

# Update version in main.go
echo "📝 Updating version in main.go"
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s/const version = \".*\"/const version = \"${VERSION#v}\"/" main.go
else
    # Linux
    sed -i "s/const version = \".*\"/const version = \"${VERSION#v}\"/" main.go
fi

# Show the version change
echo "📋 Version updated:"
grep "const version" main.go

# Run tests
echo "🧪 Running tests"
if ! make test-all; then
    echo "❌ Tests failed. Aborting release."
    git checkout -- main.go
    exit 1
fi

# Build to verify
echo "🔨 Building to verify"
if ! make build; then
    echo "❌ Build failed. Aborting release."
    git checkout -- main.go
    exit 1
fi

# Test the built binary
echo "🔍 Testing built binary"
if ! ./bin/deadman-go -version; then
    echo "❌ Binary test failed. Aborting release."
    git checkout -- main.go
    exit 1
fi

# Show changes
echo "📋 Changes to be committed:"
git diff --name-only

# Commit version update
echo "💾 Committing version update"
git add main.go
git commit -m "Release $VERSION"

# Create annotated tag
echo "🏷️  Creating tag $VERSION"
if [ "$VERSION" = "v0.0.1" ]; then
    TAG_MESSAGE="Release $VERSION

🎉 Initial release of deadman-go

Features:
- Go implementation of deadman ping monitoring
- Configuration compatibility with original deadman
- Terminal UI with real-time status display
- Concurrent monitoring with configurable limits
- Prometheus metrics support
- Cross-platform binaries (Linux, macOS, Windows)

This project is inspired by and maintains compatibility with the original deadman tool by upa."
else
    TAG_MESSAGE="Release $VERSION

See CHANGELOG.md for detailed changes."
fi

git tag -a "$VERSION" -m "$TAG_MESSAGE"

# Push changes and tag
echo "📤 Pushing changes and tag"
git push origin main
git push origin "$VERSION"

echo ""
echo "✅ Release $VERSION has been created!"
echo "🔗 Check the release at: https://github.com/doridoridoriand/deadman-go/releases/tag/$VERSION"
echo "⏳ GitHub Actions will build and publish the binaries automatically."
echo ""
echo "📊 You can monitor the build progress at:"
echo "   https://github.com/doridoridoriand/deadman-go/actions"
echo ""
echo "🔄 Returning to original branch: $CURRENT_BRANCH"
if [ "$CURRENT_BRANCH" != "main" ]; then
    git checkout "$CURRENT_BRANCH"
fi