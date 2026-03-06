# Package Repository Operations

This runbook covers rollback and recovery for package publication failures across:

- `apt`
- `dnf`
- `brew`
- `choco`

## When To Use This Runbook

Use this runbook when `.github/workflows/publish-packages.yml` completes with one or more failed managers, or when installs fail after publication.

Primary signal:

- `publication-status-summary` artifact shows one or more entries with `state=failed`.

## Inputs Required

- Failed workflow run URL (or run ID)
- Target version that failed (for example, `v0.0.10`)
- Last known-good version (for example, `v0.0.9`)
- Failed manager list (`apt`, `dnf`, `brew`, `choco`)

## Rollback Procedure

### 1. Contain

1. Stop additional release/publish runs until rollback is complete.
2. Download and archive:
   - `publication-status-summary/summary.json`
   - `publication-status-summary/summary.md`
3. Record failed managers and error messages.

### 2. Choose Rollback Strategy

1. If failure is transient (network/auth timeout), re-run same version for failed managers.
2. If published content is bad, republish the last known-good version for failed managers.

### 3. Execute Manager-Scoped Rollback

Run `Publish Packages` manually (`workflow_dispatch`) with:

- `version`: rollback target (`<failed-version>` or `<last-known-good-version>`)
- `managers`: only failed managers (comma-separated)
- `dry_run`: `false`
- `retry_failed_once`: `true`

Example with GitHub CLI:

```bash
gh workflow run publish-packages.yml --ref main \
  -f version=v0.0.9 \
  -f managers=apt,dnf \
  -f dry_run=false \
  -f retry_failed_once=true
```

### 4. Manager-Specific Notes

- `apt`:
  - Verify `dists/stable/Release`, `InRelease`/`Release.gpg`, and `binary-*/Packages*` are regenerated.
  - If signing fails, update `APT_GPG_PRIVATE_KEY` / `APT_GPG_PASSPHRASE` and re-run.
- `dnf`:
  - Verify `packages/repodata/` is regenerated and signature file is present when signing is enabled.
  - If signing fails, update `RPM_GPG_PRIVATE_KEY` / `RPM_GPG_PASSPHRASE` and re-run.
- `brew`:
  - Re-run publish for `brew` to rewrite the formula in the tap to the selected rollback version.
  - Confirm tap commit includes the expected version/checksum.
- `choco`:
  - If push failed, fix credentials/source and re-run `choco`.
  - If an immutable feed already accepted a bad package version, publish a newer fixed patch version and communicate the upgrade path.

### 5. Verify Rollback

1. Confirm the rollback publish run has no failed managers in `summary.json`.
2. Run install smoke for the rollback version:

```bash
gh workflow run package-install-smoke.yml --ref main -f version=v0.0.9
```

3. Confirm all install jobs are green:
   - `install-smoke-ubuntu`
   - `install-smoke-fedora`
   - `install-smoke-macos`
   - `install-smoke-windows`

## Post-Rollback Follow-Up

1. Create an incident note with:
   - failure cause
   - impacted managers
   - rollback run URL
   - verification run URL
2. Add/adjust preventive checks in scripts or workflow.
3. Resume normal release flow only after smoke validation succeeds.
