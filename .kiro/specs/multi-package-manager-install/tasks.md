# Tasks: Multi Package Manager Installation

## Metadata

- Feature directory: `.kiro/specs/multi-package-manager-install`
- Plan source (mapped): `.kiro/specs/multi-package-manager-install/implementation-plan.md`
- Spec source (mapped): `.kiro/specs/multi-package-manager-install/feature-spec.md`
- Optional docs loaded:
  - `.kiro/specs/multi-package-manager-install/data-model.md`
  - `.kiro/specs/multi-package-manager-install/contracts/package-distribution.openapi.yaml`
  - `.kiro/specs/multi-package-manager-install/research.md`
  - `.kiro/specs/multi-package-manager-install/quickstart.md`

## Phase 1: Setup

- [x] T001 Create packaging directory scaffold and usage note in `packaging/README.md`
- [x] T002 Add base nfpm configuration skeleton in `packaging/nfpm/nfpm.yaml`
- [x] T003 Add Homebrew formula template in `packaging/homebrew/surveiller.rb.tmpl`
- [x] T004 Add Chocolatey nuspec template in `packaging/choco/surveiller.nuspec.tmpl`
- [x] T005 Add package build helper targets in `Makefile`

## Phase 2: Foundational (Blocking Prerequisites)

- [ ] T006 Implement release manifest generator in `scripts/packaging/generate_release_manifest.sh`
- [ ] T007 [P] Add release manifest schema in `packaging/release-manifest.schema.json`
- [ ] T008 [P] Implement semantic version normalization helper in `scripts/packaging/normalize_version.sh`
- [ ] T009 Implement shared packaging environment loader in `scripts/packaging/lib/common.sh`
- [ ] T010 Update release artifact baseline stage in `.github/workflows/release.yml`

## Phase 3: User Story 1 (P1) - Build Package Artifacts

Story goal:
Tag releaseで `.deb` / `.rpm` / Homebrew formula payload / Chocolatey payload を生成し、既存バイナリ配布を維持する。

Independent test criteria:
`.github/workflows/test-release.yml` 実行結果に、既存バイナリ + パッケージ成果物 + checksum/license/versionメタデータが含まれること。

- [ ] T011 [US1] Define DEB/RPM package metadata and contents in `packaging/nfpm/nfpm.yaml`
- [ ] T012 [P] [US1] Implement Linux package build script in `scripts/packaging/build_linux_packages.sh`
- [ ] T013 [P] [US1] Implement Homebrew formula rendering script in `scripts/packaging/render_homebrew_formula.sh`
- [ ] T014 [P] [US1] Implement Chocolatey package assembly script in `scripts/packaging/build_choco_package.ps1`
- [ ] T015 [US1] Wire package build jobs into `.github/workflows/release.yml`
- [ ] T016 [US1] Add package artifact assertions in `.github/workflows/test-release.yml`
- [ ] T017 [US1] Add package artifact integrity verification script in `scripts/packaging/verify_package_artifacts.sh`

## Phase 4: User Story 2 (P2) - Publish Repository/Tap/Feed

Story goal:
`apt` / `dnf` / `brew` / `choco` 向けの公開処理を自動化し、公開状態を追跡可能にする。

Independent test criteria:
公開ワークフローの dry-run で4マネージャすべての publish ステップが成功し、publication status 出力が `published` または `queued` の期待状態になること。

- [ ] T018 [US2] Create package publication workflow in `.github/workflows/publish-packages.yml`
- [ ] T019 [P] [US2] Implement APT repository publish and sign script in `scripts/packaging/publish_apt_repo.sh`
- [ ] T020 [P] [US2] Implement DNF repository publish and sign script in `scripts/packaging/publish_dnf_repo.sh`
- [ ] T021 [P] [US2] Implement Homebrew tap publication script in `scripts/packaging/publish_homebrew_tap.sh`
- [ ] T022 [P] [US2] Implement Chocolatey publication script in `scripts/packaging/publish_choco_package.ps1`
- [ ] T023 [US2] Implement publication status collector in `scripts/packaging/collect_publication_status.sh`
- [ ] T024 [US2] Wire status summary and retry handling in `.github/workflows/publish-packages.yml`

## Phase 5: User Story 3 (P3) - Install Validation and User Documentation

Story goal:
各パッケージマネージャの install 動作をCIで検証し、利用者向け手順を README/Release Notes に反映する。

Independent test criteria:
install smoke test workflow が Ubuntu/Fedora/macOS/Windows の各ジョブで `surveiller -version` を成功させること。README と Release notes に4マネージャの install コマンドが記載されること。

- [ ] T025 [US3] Create install smoke-test matrix workflow in `.github/workflows/package-install-smoke.yml`
- [ ] T026 [P] [US3] Add apt smoke test script in `scripts/packaging/ci/smoke_apt.sh`
- [ ] T027 [P] [US3] Add dnf smoke test script in `scripts/packaging/ci/smoke_dnf.sh`
- [ ] T028 [P] [US3] Add brew smoke test script in `scripts/packaging/ci/smoke_brew.sh`
- [ ] T029 [P] [US3] Add choco smoke test script in `scripts/packaging/ci/smoke_choco.ps1`
- [ ] T030 [US3] Add package-manager installation section in `README.md`
- [ ] T031 [US3] Add package publication operation guide in `docs/RELEASE.md`
- [ ] T032 [US3] Update release notes generation block in `.github/workflows/release.yml`

## Final Phase: Polish & Cross-Cutting

- [ ] T033 Add rollback runbook for failed publication in `docs/package-repository-operations.md`
- [ ] T034 [P] Add signing key rotation checklist in `docs/RELEASE.md`
- [ ] T035 [P] Add package-manager troubleshooting notes in `README.md`
- [ ] T036 Add final verification checklist in `.kiro/specs/multi-package-manager-install/quickstart.md`

## Dependencies (User Story Order)

1. Phase 1 Setup -> Phase 2 Foundational
2. US1 depends on Phase 2
3. US2 depends on US1 artifacts
4. US3 depends on US2 publication endpoints and US1 artifacts
5. Final Phase depends on US1 + US2 + US3

## Dependency Graph

`Setup -> Foundational -> US1 -> US2 -> US3 -> Polish`

## Parallel Execution Examples

US1 parallel set (after T011):
- T012, T013, T014

US2 parallel set (after T018):
- T019, T020, T021, T022

US3 parallel set (after T025):
- T026, T027, T028, T029

## Implementation Strategy

1. MVP first: complete US1 to ensure package artifacts are generated without breaking current binary release.
2. Increment 2: implement US2 to automate publication to each ecosystem.
3. Increment 3: implement US3 to add install validation and user-facing installation docs.
4. Finalize with Polish tasks for rollback, key management, and operational readiness.
