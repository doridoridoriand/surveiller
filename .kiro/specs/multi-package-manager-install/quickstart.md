# Quickstart: Multi Package Manager Release Flow

## Prerequisites

- GitHub Actions secrets:
  - `APT_GPG_PRIVATE_KEY`
  - `RPM_GPG_PRIVATE_KEY`
  - `HOMEBREW_TAP_PUSH_TOKEN`
  - `CHOCO_API_KEY`
- Packaging tools available in CI jobs:
  - `nfpm`
  - `createrepo_c` (for DNF metadata generation)
  - `choco` (Windows publish job)

## 1. Create a release tag

```bash
./scripts/release.sh v0.1.0
```

## 2. Build binary and package artifacts

Expected outputs per tag:
- Binaries: `surveiller-{os}-{arch}`
- Packages: `.deb`, `.rpm`, `.nupkg`
- Metadata: checksums + signed repository metadata

## 3. Publish repositories/tap/feed

- APT repo update and index/sign
- DNF repo update and `repodata` refresh
- Homebrew tap formula bump commit/PR
- Chocolatey push

## 4. Validate installation matrix

Smoke tests must pass:
- Ubuntu: `apt install surveiller`
- Fedora: `dnf install surveiller`
- macOS: `brew install <tap>/surveiller`
- Windows: `choco install surveiller`

Each test verifies:
```bash
surveiller -version
```

## 5. Rollback strategy

- Keep binary-only release assets as fallback.
- If one manager publish fails, mark manager status `failed` and do not block already-successful managers.
- Re-run publish job for failed manager after fix.

