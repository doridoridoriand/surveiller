# Data Model: Multi Package Manager Installation

## Entity: ReleaseManifest

Represents a tagged release and shared metadata consumed by all packaging targets.

Fields:
- `version` (string, required, pattern `^v\d+\.\d+\.\d+([-.].+)?$`)
- `commit_sha` (string, required)
- `release_date` (RFC3339 timestamp, required)
- `artifacts` (array of `BuildArtifact`, required, min 1)
- `checksums` (map filename -> sha256, required)

Validation:
- `version` must match git tag.
- All artifacts must have checksum entries.

## Entity: BuildArtifact

Represents base binaries produced by build matrix.

Fields:
- `name` (string, required)
- `os` (enum: `linux`, `darwin`, `windows`)
- `arch` (enum: `amd64`, `arm64`)
- `url` (string, required)
- `sha256` (string, required, 64 hex chars)

Validation:
- `name` uniqueness within release.
- URL must be HTTPS.

## Entity: PackageRecipe

Defines manager-specific package metadata generated from `ReleaseManifest`.

Fields:
- `manager` (enum: `apt`, `dnf`, `brew`, `choco`)
- `alias` (optional string; `choro` maps to `choco`)
- `package_name` (string, default `surveiller`)
- `version` (string, required, normalized per manager rules)
- `license` (string, required, `MIT`)
- `homepage` (string, required)
- `description` (string, required)
- `dependencies` (array of strings)
- `source_artifact` (reference to `BuildArtifact`)

Validation:
- `source_artifact.os/arch` must match manager target support.
- `package_name` must be stable across versions.

## Entity: RepositoryPublication

Tracks publication status for each package manager.

Fields:
- `manager` (enum)
- `repository_target` (string, required)
- `state` (enum: `queued`, `building`, `published`, `failed`, `rolled_back`)
- `published_at` (timestamp, nullable)
- `error_message` (string, nullable)
- `retry_count` (int, default 0)

State transitions:
1. `queued -> building`
2. `building -> published | failed`
3. `failed -> queued` (retry)
4. `published -> rolled_back` (manual rollback only)

## Entity: InstallValidationResult

Records smoke-test result of package installation.

Fields:
- `manager` (enum)
- `platform` (string)
- `runner_image` (string)
- `install_command` (string)
- `version_output` (string)
- `state` (enum: `pass`, `fail`)
- `logs_url` (string)

Validation:
- `state=pass` requires non-empty `version_output`.
- Release marked successful only if all required managers have `pass`.

## Relationships

1. `ReleaseManifest` 1:N `BuildArtifact`
2. `ReleaseManifest` 1:N `PackageRecipe`
3. `PackageRecipe` 1:1 `RepositoryPublication`
4. `RepositoryPublication` 1:N `InstallValidationResult` (retries/variants)

