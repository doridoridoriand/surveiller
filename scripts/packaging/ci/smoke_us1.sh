#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  smoke_us1.sh [options]

Options:
  --version <tag>      Release tag used for smoke packaging (default: v0.0.1-smoke)
  --dist-dir <path>    Dist directory (default: dist)
  -h, --help           Show this help
EOF
}

VERSION="v0.0.1-smoke"
DIST_DIR="dist"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --dist-dir)
      DIST_DIR="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "[packaging] ERROR: unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

cd "${REPO_ROOT}"

echo "[packaging] building release binaries for smoke test"
make release

echo "[packaging] generating release manifest"
./scripts/packaging/generate_release_manifest.sh \
  --version "${VERSION}" \
  --commit-sha "$(git rev-parse HEAD)" \
  --release-date "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --repository "${GITHUB_REPOSITORY:-doridoridoriand/surveiller}" \
  --dist-dir "${DIST_DIR}" \
  --checksums-file "${DIST_DIR}/checksums.txt" \
  --schema packaging/release-manifest.schema.json \
  --output "${DIST_DIR}/release-manifest.json"

echo "[packaging] building linux packages"
./scripts/packaging/build_linux_packages.sh \
  --manifest "${DIST_DIR}/release-manifest.json" \
  --dist-dir "${DIST_DIR}" \
  --checksums-file "${DIST_DIR}/checksums.txt"

echo "[packaging] rendering homebrew formula"
./scripts/packaging/render_homebrew_formula.sh \
  --manifest "${DIST_DIR}/release-manifest.json" \
  --dist-dir "${DIST_DIR}" \
  --checksums-file "${DIST_DIR}/checksums.txt"

if command -v pwsh >/dev/null 2>&1; then
  echo "[packaging] building chocolatey payload"
  pwsh -File ./scripts/packaging/build_choco_package.ps1 \
    -ManifestPath "${DIST_DIR}/release-manifest.json" \
    -DistDir "${DIST_DIR}" \
    -ChecksumsFile "${DIST_DIR}/checksums.txt"
else
  echo "[packaging] ERROR: pwsh is required for smoke_us1.sh" >&2
  exit 1
fi

echo "[packaging] verifying package artifacts"
./scripts/packaging/verify_package_artifacts.sh \
  --manifest "${DIST_DIR}/release-manifest.json" \
  --dist-dir "${DIST_DIR}" \
  --checksums-file "${DIST_DIR}/checksums.txt"

echo "[packaging] US1 smoke test passed"
