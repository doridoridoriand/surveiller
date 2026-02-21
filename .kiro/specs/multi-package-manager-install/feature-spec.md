# Feature Specification: Multi Package Manager Installation

## Summary

Enable `surveiller` installation via major package managers so users can install and update without manual binary download.

Target package managers:
- `apt` (Debian/Ubuntu)
- `dnf` (Fedora/RHEL-family)
- `brew` (Homebrew)
- `choco` (Chocolatey; user input included `choro`, treated as alias/typo)

## Problem Statement

Current releases publish standalone binaries only. This creates friction for adoption and upgrades in operational environments where package managers are standard.

## Goals

- Provide package-manager-native install paths for Linux, macOS, and Windows.
- Keep release flow mostly automated from Git tags.
- Preserve checksum/signature verification in package metadata.
- Keep current direct binary release flow working during rollout.

## Non-Goals

- Building custom GUI installers.
- Supporting every ecosystem (`snap`, `winget`, `pacman`, etc.) in this phase.
- Changing runtime behavior of `surveiller` itself.

## Functional Requirements

### FR-1: Package Build Artifacts
- On tagged release, the pipeline MUST build `.deb`, `.rpm`, Homebrew formula update payload, and Chocolatey package payload.
- Package metadata MUST include version, checksum, and license.

### FR-2: Repository/Tap Publication
- `apt` packages MUST be publishable to an APT repository layout.
- `dnf` packages MUST be publishable to a YUM/DNF repository layout.
- `brew` formula MUST be published to a tap repository.
- `choco` package MUST be publishable to Chocolatey community/internal feed.

### FR-3: Install Command Documentation
- README/release notes MUST expose copy-paste install commands for each manager.
- Commands MUST reference stable channels for latest version installation.

### FR-4: Validation
- CI MUST validate package install for each manager in isolated environments.
- Validation MUST include `surveiller -version` and basic command invocation.

### FR-5: Backward Compatibility
- Existing binary artifacts and checksums MUST continue to be released.
- Existing release tag trigger workflow MUST remain the source of truth.

## Success Criteria

- Users can install via:
  - `apt install surveiller`
  - `dnf install surveiller`
  - `brew install <tap>/surveiller`
  - `choco install surveiller`
- End-to-end release pipeline completes with package publish status visible.
- Rollback plan exists for failed package publication.

