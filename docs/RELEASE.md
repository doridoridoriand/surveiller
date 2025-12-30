# Release Process

This document describes the release process for surveiller.

## Overview

Releases are automated using GitHub Actions. When a version tag is pushed, the CI system automatically:

1. Runs all tests
2. Builds binaries for multiple platforms
3. Creates checksums
4. Publishes a GitHub release with binaries

## Release Steps

### Method 1: Using the Release Script (Recommended)

```bash
# Create and push a new release
./scripts/release.sh v0.0.1
```

The script will:
- Validate the version format
- Check that you're on a clean working directory
- Update the version in `main.go`
- Run tests to ensure everything works
- Commit the version change
- Create and push the git tag
- Trigger the automated release process

### Method 2: Manual Process

1. **Update version in main.go**
   ```go
   const version = "0.0.1"  // Remove the 'v' prefix
   ```

2. **Run tests**
   ```bash
   make test-all
   ```

3. **Commit version change**
   ```bash
   git add main.go
   git commit -m "Release v0.0.1"
   git push origin main
   ```

4. **Create and push tag**
   ```bash
   git tag -a v0.0.1 -m "Release v0.0.1"
   git push origin v0.0.1
   ```

### Method 3: Using Make Commands

```bash
# Check current status
make release-check

# Create a specific release
make release-tag TAG=v0.0.1

# Quick development release (creates v0.0.1)
make release-dev
```

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR.MINOR.PATCH** (e.g., `v1.0.0`)
- **Pre-release**: `v1.0.0-alpha1`, `v1.0.0-beta1`, `v1.0.0-rc1`

### Version Guidelines

- **v0.x.x**: Development/pre-1.0 releases
- **v1.0.0**: First stable release
- **Patch** (v1.0.1): Bug fixes, no new features
- **Minor** (v1.1.0): New features, backward compatible
- **Major** (v2.0.0): Breaking changes

## Release Artifacts

Each release includes:

### Binaries
- `surveiller-linux-amd64` - Linux x86_64
- `surveiller-linux-arm64` - Linux ARM64
- `surveiller-darwin-amd64` - macOS Intel
- `surveiller-darwin-arm64` - macOS Apple Silicon
- `surveiller-windows-amd64.exe` - Windows x86_64

### Checksums
- `checksums.txt` - SHA256 checksums for all binaries

## Automated Release Process

The GitHub Actions workflow (`.github/workflows/release.yml`) handles:

1. **Testing**: Runs unit tests and property-based tests
2. **Building**: Cross-compiles for all supported platforms
3. **Checksums**: Generates SHA256 checksums
4. **Release Notes**: Creates installation instructions
5. **Publishing**: Creates GitHub release with all artifacts

## Pre-release Testing

Before creating a release:

```bash
# Run all tests
make test-all

# Build locally to verify
make build

# Test the binary
./bin/surveiller -version

# Cross-compile test
make release
```

## Release Checklist

- [ ] All tests pass (`make test-all`)
- [ ] Version updated in `main.go`
- [ ] CHANGELOG.md updated (if applicable)
- [ ] Working directory is clean
- [ ] On main branch (recommended)
- [ ] Tag follows semantic versioning

## Troubleshooting

### Failed Release

If a release fails:

1. Check the GitHub Actions logs
2. Fix any issues
3. Delete the tag if necessary:
   ```bash
   git tag -d v0.0.1
   git push origin :refs/tags/v0.0.1
   ```
4. Create a new release

### Version Conflicts

If you need to update a version:

1. Never modify existing tags
2. Create a new patch version instead
3. Document the issue in release notes

## Post-Release

After a successful release:

1. Verify binaries work on target platforms
2. Update documentation if needed
3. Announce the release (if significant)
4. Monitor for issues

## Development Releases

For testing purposes, you can create development releases:

```bash
# Create a development release
./scripts/release.sh v0.0.1-dev1

# Or use the quick command
make release-dev
```

Development releases are automatically marked as pre-releases on GitHub.