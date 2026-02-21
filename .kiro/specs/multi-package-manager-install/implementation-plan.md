# Implementation Plan: Multi Package Manager Installation

## Setup (Fallback Execution Notes)

- Requested script: `.specify/scripts/bash/setup-plan.sh --json`
- Result: `no such file or directory`
- Repository has no `.specify` directory; this plan is generated with equivalent structure under:
  - `SPECS_DIR=.kiro/specs/multi-package-manager-install`
  - `FEATURE_SPEC=.kiro/specs/multi-package-manager-install/feature-spec.md`
  - `IMPL_PLAN=.kiro/specs/multi-package-manager-install/implementation-plan.md`
  - `BRANCH=main`

## Technical Context

| Item | Value |
|---|---|
| Product | `surveiller` CLI (Go) |
| Current release model | GitHub tag -> binaries + checksums |
| Target managers | `apt`, `dnf`, `brew`, `choco` |
| Build environment | GitHub Actions (`ubuntu-latest`) |
| Existing artifacts | linux/macOS/windows binaries |
| Packaging tool candidates | `goreleaser` + `nfpm`, repo publish scripts |
| Security baseline | SHA256 checksums already produced |

### Initial Clarification Items (Resolved in Phase 0)

1. User input `choro` exact meaning.
2. How to publish APT and DNF repositories (self-hosted vs external service).
3. Whether to migrate whole release flow to `goreleaser` or keep existing workflow plus additive packaging.
4. Package verification requirements (GPG signing scope per manager).

All unknowns are resolved in `research.md`.

## Constitution Check (Pre-Design)

Requested source: `.specify/memory/constitution.md`  
Result: not found in repository.  
Fallback constitution source: `CONTRIBUTING.md` + current release/CI conventions.

Derived principles:
1. Backward compatibility with existing CLI and release artifacts.
2. Automated verification required for new functionality.
3. Documentation updates required for user-facing changes.
4. Keep changes maintainable and reproducible.

Pre-design status:
- Principle 1: PASS (plan keeps binary releases)
- Principle 2: PASS (install validation added per manager)
- Principle 3: PASS (quickstart + README updates planned)
- Principle 4: PASS (single source release metadata model planned)

## Gate Evaluation (Pre-Phase 0)

- Gate A: No unresolved clarification items before final plan -> PASS (resolved in Phase 0 research).
- Gate B: No constitution violations without justification -> PASS.
- Gate C: Existing release compatibility preserved -> PASS.

## Phase 0: Research Plan

Research tasks dispatched:
1. Resolve `choro` interpretation in package-manager context.
2. Compare publication strategies for APT/DNF repositories.
3. Compare incremental vs full release-workflow migration.
4. Define signing/checksum policy by package manager.
5. Identify best-practice structure for Homebrew tap and Chocolatey package updates.

Output:
- `research.md`

## Phase 1: Design & Contracts Plan

Design outputs:
1. `data-model.md`
2. `contracts/package-distribution.openapi.yaml`
3. `quickstart.md`
4. Agent context update output for Codex

### Planned Implementation Slices

1. Build system foundation
- Introduce unified release metadata (version, checksums, URLs).
- Add package recipe generation (`.deb`, `.rpm`, Homebrew formula template, Chocolatey nuspec/template).

2. Publication automation
- APT: publish to repository structure with signed metadata.
- DNF: publish RPMs + `repodata`.
- Brew: update formula in tap repo via PR/commit automation.
- Choco: package and push via API key from secrets.

3. Validation and rollback
- Add install smoke tests per manager.
- Add publication dry-run mode on non-tag builds.
- Keep binary release as fallback on package publish failure.

## Phase 2: Execution Backlog

1. Introduce release manifest generator.
2. Integrate `nfpm` packaging for DEB/RPM.
3. Add APT repository publish workflow/job.
4. Add DNF repository publish workflow/job.
5. Add Homebrew tap update automation.
6. Add Chocolatey package build/publish automation.
7. Add CI matrix install tests.
8. Update README/release notes install sections.
9. Add operational runbook for key rotation and rollback.

## Agent Context Update

Requested script: `.specify/scripts/bash/update-agent-context.sh codex`  
Result: `no such file or directory`  
Fallback action: manually updated agent context at:
- `.kiro/agent-context/codex.md`

## Constitution Check (Post-Design)

1. Compatibility preserved? -> PASS
2. Automated testing defined? -> PASS
3. Documentation deliverables included? -> PASS
4. Reproducible release flow? -> PASS (through manifest + scripted publication)
5. Security integrity addressed? -> PASS (checksums + signing policy in research/design)

No gate violations remain.
