# Codex Agent Context Update (Manual Fallback)

## Why this file exists

The requested script `.specify/scripts/bash/update-agent-context.sh codex` is not present in this repository.  
This file records the equivalent context update manually for the current plan.

## Newly introduced technologies (from current plan)

- `nfpm` for `.deb` and `.rpm` package generation
- APT repository indexing/signing workflow
- DNF/YUM repository metadata generation (`repodata`)
- Homebrew tap automation
- Chocolatey package automation (`.nupkg`)

## Constraints to preserve

- Keep existing binary release artifacts and checksums.
- Keep tag-driven release flow as source of truth.
- Add manager-specific smoke tests in CI.

