#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  generate_release_manifest.sh [options]

Options:
  --version <tag>              Release version tag (e.g. v1.2.3)
  --commit-sha <sha>           Commit SHA for this release
  --release-date <rfc3339>     Release timestamp (UTC RFC3339)
  --repository <owner/repo>    GitHub repository slug
  --release-base-url <url>     Base URL for release artifacts
  --dist-dir <path>            Distribution directory (default: dist)
  --checksums-file <path>      Checksum file (default: <dist>/checksums.txt)
  --schema <path>              Manifest schema path (existence check only)
  --output <path>              Output manifest path (default: <dist>/release-manifest.json)
  -h, --help                   Show this help
EOF
}

artifact_os_arch() {
  local artifact_name="$1"
  case "${artifact_name}" in
    surveiller-linux-amd64)
      printf 'linux amd64'
      ;;
    surveiller-linux-arm64)
      printf 'linux arm64'
      ;;
    surveiller-darwin-amd64)
      printf 'darwin amd64'
      ;;
    surveiller-darwin-arm64)
      printf 'darwin arm64'
      ;;
    surveiller-windows-amd64.exe)
      printf 'windows amd64'
      ;;
    *)
      return 1
      ;;
  esac
}

main() {
  local version commit_sha release_date repository dist_dir checksums_file
  local schema_path output_path release_base_url
  local normalize_script

  version="${VERSION:-}"
  if [[ "${version}" != v* ]]; then
    version="v${version}"
  fi
  commit_sha="${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || echo "unknown")}"
  release_date="$(packaging_utc_now)"
  repository="$(packaging_default_repository)"
  dist_dir="dist"
  checksums_file=""
  schema_path="packaging/release-manifest.schema.json"
  output_path=""
  release_base_url=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version)
        version="${2:-}"
        shift 2
        ;;
      --commit-sha)
        commit_sha="${2:-}"
        shift 2
        ;;
      --release-date)
        release_date="${2:-}"
        shift 2
        ;;
      --repository)
        repository="${2:-}"
        shift 2
        ;;
      --release-base-url)
        release_base_url="${2:-}"
        shift 2
        ;;
      --dist-dir)
        dist_dir="${2:-}"
        shift 2
        ;;
      --checksums-file)
        checksums_file="${2:-}"
        shift 2
        ;;
      --schema)
        schema_path="${2:-}"
        shift 2
        ;;
      --output)
        output_path="${2:-}"
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        packaging_die "unknown argument: $1"
        ;;
    esac
  done

  if [ -z "${version}" ]; then
    version="$(packaging_default_version)"
  fi

  if [[ "${version}" != v* ]]; then
    version="v${version}"
  fi

  if [[ ! "${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
    packaging_die "version must be semantic tag format (vX.Y.Z): ${version}"
  fi

  if [ -z "${checksums_file}" ]; then
    checksums_file="${dist_dir}/checksums.txt"
  fi

  if [ -z "${output_path}" ]; then
    output_path="${dist_dir}/release-manifest.json"
  fi

  if [ -z "${release_base_url}" ]; then
    release_base_url="$(packaging_release_base_url "${repository}" "${version}")"
  fi

  normalize_script="${SCRIPT_DIR}/normalize_version.sh"

  packaging_require_file "${checksums_file}"
  packaging_require_file "${schema_path}"
  packaging_require_executable "${normalize_script}"
  mkdir -p "$(dirname "${output_path}")"

  local version_normalized apt_version dnf_version brew_version choco_version
  version_normalized="$("${normalize_script}" generic "${version}")"
  apt_version="$("${normalize_script}" apt "${version}")"
  dnf_version="$("${normalize_script}" dnf "${version}")"
  brew_version="$("${normalize_script}" brew "${version}")"
  choco_version="$("${normalize_script}" choco "${version}")"

  local checksum_entries
  checksum_entries="$(packaging_sorted_checksums "${checksums_file}")"
  if [ -z "${checksum_entries}" ]; then
    packaging_die "no checksum entries found in ${checksums_file}"
  fi

  local artifact_names artifact_count
  artifact_names=(
    "surveiller-linux-amd64"
    "surveiller-linux-arm64"
    "surveiller-darwin-amd64"
    "surveiller-darwin-arm64"
    "surveiller-windows-amd64.exe"
  )
  artifact_count=0

  local tmp_manifest
  tmp_manifest="$(mktemp)"
  trap 'rm -f "${tmp_manifest}"' EXIT

  {
    printf '{\n'
    printf '  "version": "%s",\n' "$(packaging_json_escape "${version}")"
    printf '  "version_normalized": "%s",\n' "$(packaging_json_escape "${version_normalized}")"
    printf '  "commit_sha": "%s",\n' "$(packaging_json_escape "${commit_sha}")"
    printf '  "release_date": "%s",\n' "$(packaging_json_escape "${release_date}")"
    printf '  "repository": "%s",\n' "$(packaging_json_escape "${repository}")"
    printf '  "release_base_url": "%s",\n' "$(packaging_json_escape "${release_base_url}")"
    printf '  "artifacts": [\n'

    local artifact_name artifact_sha os_arch artifact_os artifact_arch artifact_index
    artifact_index=0
    for artifact_name in "${artifact_names[@]}"; do
      artifact_sha="$(packaging_lookup_checksum "${checksums_file}" "${artifact_name}")"
      if [ -z "${artifact_sha}" ]; then
        continue
      fi

      if ! packaging_validate_sha256 "${artifact_sha}"; then
        packaging_die "invalid sha256 for artifact ${artifact_name}: ${artifact_sha}"
      fi

      os_arch="$(artifact_os_arch "${artifact_name}")"
      artifact_os="${os_arch%% *}"
      artifact_arch="${os_arch##* }"

      if [ "${artifact_index}" -gt 0 ]; then
        printf ',\n'
      fi
      printf '    {\n'
      printf '      "name": "%s",\n' "$(packaging_json_escape "${artifact_name}")"
      printf '      "type": "binary",\n'
      printf '      "os": "%s",\n' "$(packaging_json_escape "${artifact_os}")"
      printf '      "arch": "%s",\n' "$(packaging_json_escape "${artifact_arch}")"
      printf '      "url": "%s",\n' "$(packaging_json_escape "${release_base_url}/${artifact_name}")"
      printf '      "sha256": "%s"\n' "$(packaging_json_escape "${artifact_sha}")"
      printf '    }'

      artifact_index=$((artifact_index + 1))
      artifact_count=$((artifact_count + 1))
    done

    if [ "${artifact_count}" -eq 0 ]; then
      printf '\n'
    else
      printf '\n'
    fi

    printf '  ],\n'
    printf '  "checksums": {\n'

    local checksum_file checksum_sha checksum_index
    checksum_index=0
    while IFS=$'\t' read -r checksum_file checksum_sha; do
      [ -n "${checksum_file}" ] || continue
      if ! packaging_validate_sha256 "${checksum_sha}"; then
        packaging_die "invalid sha256 in checksums map for ${checksum_file}: ${checksum_sha}"
      fi
      if [ "${checksum_index}" -gt 0 ]; then
        printf ',\n'
      fi
      printf '    "%s": "%s"' \
        "$(packaging_json_escape "${checksum_file}")" \
        "$(packaging_json_escape "${checksum_sha}")"
      checksum_index=$((checksum_index + 1))
    done <<< "${checksum_entries}"
    printf '\n'
    printf '  },\n'
    printf '  "package_versions": {\n'
    printf '    "apt": "%s",\n' "$(packaging_json_escape "${apt_version}")"
    printf '    "dnf": "%s",\n' "$(packaging_json_escape "${dnf_version}")"
    printf '    "brew": "%s",\n' "$(packaging_json_escape "${brew_version}")"
    printf '    "choco": "%s"\n' "$(packaging_json_escape "${choco_version}")"
    printf '  }\n'
    printf '}\n'
  } > "${tmp_manifest}"

  if [ "${artifact_count}" -eq 0 ]; then
    packaging_die "no known release artifacts found in ${checksums_file}"
  fi

  if [ "${PACKAGING_SKIP_JQ:-0}" != "1" ] && command -v jq >/dev/null 2>&1; then
    jq -e . "${tmp_manifest}" >/dev/null
  fi

  mv "${tmp_manifest}" "${output_path}"
  trap - EXIT
  packaging_log_info "release manifest generated at ${output_path}"
}

main "$@"
