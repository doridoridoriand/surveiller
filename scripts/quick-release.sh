#!/bin/bash

# Quick release script - just create and push tag
# Usage: ./scripts/quick-release.sh v0.0.1

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

echo "🚀 Creating quick release $VERSION"

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo "❌ Tag $VERSION already exists"
    exit 1
fi

# Get current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo "📍 Current branch: $CURRENT_BRANCH"

# Ensure we're on main branch
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "🔀 Switching to main branch..."
    git checkout main
    git pull origin main
fi

# Create tag
echo "🏷️  Creating tag $VERSION"
if [ "$VERSION" = "v0.0.1" ]; then
    TAG_MESSAGE="Release $VERSION

🎉 Initial release of deadman-go

A Go implementation of the deadman ping monitoring tool with terminal UI and Prometheus metrics support.

Features:
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

# Push tag
echo "📤 Pushing tag $VERSION"
git push origin "$VERSION"

echo ""
echo "✅ Release tag $VERSION has been created and pushed!"
echo "🔗 Check the release at: https://github.com/doridoridoriand/deadman-go/releases/tag/$VERSION"
echo "⏳ GitHub Actions will build and publish the binaries automatically."
echo ""
echo "📊 Monitor the build progress at:"
echo "   https://github.com/doridoridoriand/deadman-go/actions"

# Return to original branch
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "🔄 Returning to original branch: $CURRENT_BRANCH"
    git checkout "$CURRENT_BRANCH"
fi