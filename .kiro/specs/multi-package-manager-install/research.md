# Phase 0 Research: Multi Package Manager Installation

## 1) `choro` interpretation

- Decision: Treat `choro` as `choco` (Chocolatey) alias/typo.
- Rationale: In package manager sets listed with `apt/dnf/brew`, the canonical Windows counterpart is Chocolatey (`choco`); no mainstream manager named `choro` exists in this context.
- Alternatives considered:
  - Add a separate unknown manager named `choro` (rejected: no ecosystem fit).
  - Use `winget` instead (rejected: not requested).

## 2) Linux package generation approach

- Decision: Use `nfpm` to generate `.deb` and `.rpm` from the same release metadata.
- Rationale: `nfpm` is lightweight, Go-friendly, and supports both DEB/RPM with shared config, reducing duplicated packaging logic.
- Alternatives considered:
  - Native `dpkg-deb` + `rpmbuild` scripts (rejected: higher maintenance).
  - Full custom shell packaging (rejected: brittle and harder to validate).

## 3) APT/DNF repository publication

- Decision: Publish static repository layouts from CI artifacts to a versioned object storage/static host, with signed repository metadata.
- Rationale: Static hosting is simple to operate, cache-friendly, and easy to consume from `apt`/`dnf`.
- Alternatives considered:
  - Dedicated package hosting SaaS only (rejected: external lock-in risk).
  - GitHub Releases-only without repo metadata (rejected: does not support native `apt install`/`dnf install` UX).

## 4) Release workflow migration strategy

- Decision: Incremental rollout; keep current binary release workflow and add packaging jobs.
- Rationale: Minimizes release risk and preserves existing consumers while package-manager paths stabilize.
- Alternatives considered:
  - Big-bang migration to a fully new release pipeline (rejected: high blast radius).

## 5) Homebrew distribution model

- Decision: Publish via dedicated tap repository (`<org>/homebrew-tap`) with automated formula update PR/commit.
- Rationale: Tap provides explicit ownership and version control for formula updates.
- Alternatives considered:
  - Homebrew core submission (rejected for initial phase due to review cadence and policy constraints).

## 6) Chocolatey distribution model

- Decision: Build `.nupkg` with nuspec templates and publish using CI secret-managed API key.
- Rationale: Standard Chocolatey workflow; aligns with automated release tagging.
- Alternatives considered:
  - Manual local `choco push` operations (rejected: not reproducible).

## 7) Integrity and trust policy

- Decision: Keep SHA256 checksums for all binaries and packages; add package signing where ecosystem supports it (`apt` repo metadata signing, RPM signing path).
- Rationale: Consistent provenance and safer consumption across managers.
- Alternatives considered:
  - Checksums only with no signing (rejected: weaker chain of trust for repo metadata).

## 8) CI validation pattern

- Decision: Add install smoke-test matrix:
  - Ubuntu container for `apt`
  - Fedora container for `dnf`
  - macOS runner for `brew`
  - Windows runner for `choco`
- Rationale: Validates real install path per manager without over-scoping to deep functional tests.
- Alternatives considered:
  - Unit tests only for package files (rejected: misses repository/install integration issues).

