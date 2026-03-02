#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  render_homebrew_formula.sh [options]

Options:
  --manifest <path>       Release manifest path (default: dist/release-manifest.json)
  --template <path>       Formula template path (default: packaging/homebrew/surveiller.rb.tmpl)
  --dist-dir <path>       Dist directory root (default: dist)
  --output <path>         Rendered formula path (default: <dist>/packages/homebrew/surveiller.rb)
  --checksums-file <path> Checksums file (default: <dist>/checksums.txt)
  -h, --help              Show this help
EOF
}

to_abs_path() {
  local repo_root="$1"
  local path="$2"
  case "${path}" in
    /*) printf '%s\n' "${path}" ;;
    *) printf '%s\n' "${repo_root}/${path}" ;;
  esac
}

relative_to_dist() {
  local dist_dir_abs="$1"
  local file_path="$2"
  if [[ "${file_path}" == "${dist_dir_abs}/"* ]]; then
    printf '%s\n' "${file_path#${dist_dir_abs}/}"
  else
    basename "${file_path}"
  fi
}

escape_sed_replacement() {
  printf '%s' "$1" | sed -e 's/[\/&]/\\&/g'
}

manifest_artifact_field() {
  local manifest="$1"
  local artifact_name="$2"
  local field="$3"
  jq -r --arg name "${artifact_name}" --arg field "${field}" '
    [.artifacts[] | select(.name == $name) | .[$field]][0] // empty
  ' "${manifest}"
}

main() {
  local repo_root manifest_path template_path dist_dir output_path checksums_file
  local manifest_abs template_abs dist_dir_abs output_abs checksums_abs
  local brew_version
  local darwin_arm64_url darwin_arm64_sha darwin_amd64_url darwin_amd64_sha
  local linux_arm64_url linux_arm64_sha linux_amd64_url linux_amd64_sha
  local tmp_output artifact_name artifact_sha

  repo_root="$(packaging_repo_root)"
  manifest_path="dist/release-manifest.json"
  template_path="packaging/homebrew/surveiller.rb.tmpl"
  dist_dir="dist"
  output_path=""
  checksums_file=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --manifest)
        manifest_path="${2:-}"
        shift 2
        ;;
      --template)
        template_path="${2:-}"
        shift 2
        ;;
      --dist-dir)
        dist_dir="${2:-}"
        shift 2
        ;;
      --output)
        output_path="${2:-}"
        shift 2
        ;;
      --checksums-file)
        checksums_file="${2:-}"
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

  if [ -z "${output_path}" ]; then
    output_path="${dist_dir}/packages/homebrew/surveiller.rb"
  fi
  if [ -z "${checksums_file}" ]; then
    checksums_file="${dist_dir}/checksums.txt"
  fi

  manifest_abs="$(to_abs_path "${repo_root}" "${manifest_path}")"
  template_abs="$(to_abs_path "${repo_root}" "${template_path}")"
  dist_dir_abs="$(to_abs_path "${repo_root}" "${dist_dir}")"
  output_abs="$(to_abs_path "${repo_root}" "${output_path}")"
  checksums_abs="$(to_abs_path "${repo_root}" "${checksums_file}")"

  packaging_require_file "${manifest_abs}"
  packaging_require_file "${template_abs}"
  packaging_require_command jq

  brew_version="$(jq -r '.package_versions.brew // empty' "${manifest_abs}")"
  [ -n "${brew_version}" ] || packaging_die "manifest missing package_versions.brew"

  darwin_arm64_url="$(manifest_artifact_field "${manifest_abs}" "surveiller-darwin-arm64" "url")"
  darwin_arm64_sha="$(manifest_artifact_field "${manifest_abs}" "surveiller-darwin-arm64" "sha256")"
  darwin_amd64_url="$(manifest_artifact_field "${manifest_abs}" "surveiller-darwin-amd64" "url")"
  darwin_amd64_sha="$(manifest_artifact_field "${manifest_abs}" "surveiller-darwin-amd64" "sha256")"
  linux_arm64_url="$(manifest_artifact_field "${manifest_abs}" "surveiller-linux-arm64" "url")"
  linux_arm64_sha="$(manifest_artifact_field "${manifest_abs}" "surveiller-linux-arm64" "sha256")"
  linux_amd64_url="$(manifest_artifact_field "${manifest_abs}" "surveiller-linux-amd64" "url")"
  linux_amd64_sha="$(manifest_artifact_field "${manifest_abs}" "surveiller-linux-amd64" "sha256")"

  [ -n "${darwin_arm64_url}" ] || packaging_die "manifest missing darwin arm64 artifact URL"
  [ -n "${darwin_arm64_sha}" ] || packaging_die "manifest missing darwin arm64 artifact SHA"
  [ -n "${darwin_amd64_url}" ] || packaging_die "manifest missing darwin amd64 artifact URL"
  [ -n "${darwin_amd64_sha}" ] || packaging_die "manifest missing darwin amd64 artifact SHA"
  [ -n "${linux_arm64_url}" ] || packaging_die "manifest missing linux arm64 artifact URL"
  [ -n "${linux_arm64_sha}" ] || packaging_die "manifest missing linux arm64 artifact SHA"
  [ -n "${linux_amd64_url}" ] || packaging_die "manifest missing linux amd64 artifact URL"
  [ -n "${linux_amd64_sha}" ] || packaging_die "manifest missing linux amd64 artifact SHA"

  if ! packaging_validate_sha256 "${darwin_arm64_sha}"; then
    packaging_die "invalid darwin arm64 sha256: ${darwin_arm64_sha}"
  fi
  if ! packaging_validate_sha256 "${darwin_amd64_sha}"; then
    packaging_die "invalid darwin amd64 sha256: ${darwin_amd64_sha}"
  fi
  if ! packaging_validate_sha256 "${linux_arm64_sha}"; then
    packaging_die "invalid linux arm64 sha256: ${linux_arm64_sha}"
  fi
  if ! packaging_validate_sha256 "${linux_amd64_sha}"; then
    packaging_die "invalid linux amd64 sha256: ${linux_amd64_sha}"
  fi

  mkdir -p "$(dirname "${output_abs}")"
  if [ ! -f "${checksums_abs}" ]; then
    : > "${checksums_abs}"
  fi

  tmp_output="$(mktemp)"
  sed \
    -e "s/{{ \\.Version }}/$(escape_sed_replacement "${brew_version}")/g" \
    -e "s/{{ \\.DarwinArm64URL }}/$(escape_sed_replacement "${darwin_arm64_url}")/g" \
    -e "s/{{ \\.DarwinArm64SHA256 }}/$(escape_sed_replacement "${darwin_arm64_sha}")/g" \
    -e "s/{{ \\.DarwinAmd64URL }}/$(escape_sed_replacement "${darwin_amd64_url}")/g" \
    -e "s/{{ \\.DarwinAmd64SHA256 }}/$(escape_sed_replacement "${darwin_amd64_sha}")/g" \
    -e "s/{{ \\.LinuxArm64URL }}/$(escape_sed_replacement "${linux_arm64_url}")/g" \
    -e "s/{{ \\.LinuxArm64SHA256 }}/$(escape_sed_replacement "${linux_arm64_sha}")/g" \
    -e "s/{{ \\.LinuxAmd64URL }}/$(escape_sed_replacement "${linux_amd64_url}")/g" \
    -e "s/{{ \\.LinuxAmd64SHA256 }}/$(escape_sed_replacement "${linux_amd64_sha}")/g" \
    "${template_abs}" > "${tmp_output}"

  if grep -q '{{ \.' "${tmp_output}"; then
    rm -f "${tmp_output}"
    packaging_die "formula template contains unresolved placeholders"
  fi

  mv "${tmp_output}" "${output_abs}"

  artifact_name="$(relative_to_dist "${dist_dir_abs}" "${output_abs}")"
  artifact_sha="$(packaging_sha256_file "${output_abs}")"
  packaging_upsert_checksum "${checksums_abs}" "${artifact_name}" "${artifact_sha}"
  packaging_sort_checksums_file "${checksums_abs}"

  packaging_log_info "homebrew formula rendered at ${output_abs}"
}

main "$@"
