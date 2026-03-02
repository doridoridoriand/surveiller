#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  verify_package_artifacts.sh [options]

Options:
  --manifest <path>       Release manifest path (default: dist/release-manifest.json)
  --dist-dir <path>       Dist directory root (default: dist)
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

assert_checksum_entry_matches() {
  local checksums_file="$1"
  local dist_dir_abs="$2"
  local artifact_path="$3"
  local artifact_name expected_sha actual_sha

  artifact_name="$(relative_to_dist "${dist_dir_abs}" "${artifact_path}")"
  expected_sha="$(packaging_lookup_checksum "${checksums_file}" "${artifact_name}")"
  [ -n "${expected_sha}" ] || packaging_die "checksums.txt missing entry: ${artifact_name}"

  actual_sha="$(packaging_sha256_file "${artifact_path}")"
  if [ "${actual_sha}" != "${expected_sha}" ]; then
    packaging_die "checksum mismatch for ${artifact_name}: expected ${expected_sha}, got ${actual_sha}"
  fi
}

main() {
  local repo_root manifest_path dist_dir checksums_file
  local manifest_abs dist_dir_abs checksums_abs
  local linux_dir formula_path choco_dir nuspec_path
  local release_version release_version_no_v apt_version dnf_version brew_version choco_version
  local linux_amd64_sha linux_arm64_sha darwin_amd64_sha darwin_arm64_sha
  local deb_files rpm_files nupkg_files

  repo_root="$(packaging_repo_root)"
  manifest_path="dist/release-manifest.json"
  dist_dir="dist"
  checksums_file=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --manifest)
        manifest_path="${2:-}"
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
      -h|--help)
        usage
        exit 0
        ;;
      *)
        packaging_die "unknown argument: $1"
        ;;
    esac
  done

  if [ -z "${checksums_file}" ]; then
    checksums_file="${dist_dir}/checksums.txt"
  fi

  manifest_abs="$(to_abs_path "${repo_root}" "${manifest_path}")"
  dist_dir_abs="$(to_abs_path "${repo_root}" "${dist_dir}")"
  checksums_abs="$(to_abs_path "${repo_root}" "${checksums_file}")"

  packaging_require_file "${manifest_abs}"
  packaging_require_file "${checksums_abs}"
  packaging_require_command jq

  linux_dir="${dist_dir_abs}/packages/linux"
  formula_path="${dist_dir_abs}/packages/homebrew/surveiller.rb"
  choco_dir="${dist_dir_abs}/packages/choco"
  nuspec_path="${choco_dir}/surveiller.nuspec"

  release_version="$(jq -r '.version // empty' "${manifest_abs}")"
  release_version_no_v="${release_version#v}"
  apt_version="$(jq -r '.package_versions.apt // empty' "${manifest_abs}")"
  dnf_version="$(jq -r '.package_versions.dnf // empty' "${manifest_abs}")"
  brew_version="$(jq -r '.package_versions.brew // empty' "${manifest_abs}")"
  choco_version="$(jq -r '.package_versions.choco // empty' "${manifest_abs}")"

  [ -n "${release_version}" ] || packaging_die "manifest missing version"
  [ -n "${apt_version}" ] || packaging_die "manifest missing package_versions.apt"
  [ -n "${dnf_version}" ] || packaging_die "manifest missing package_versions.dnf"
  [ -n "${brew_version}" ] || packaging_die "manifest missing package_versions.brew"
  [ -n "${choco_version}" ] || packaging_die "manifest missing package_versions.choco"

  shopt -s nullglob
  deb_files=("${linux_dir}"/*_"${apt_version}"_*.deb)
  rpm_files=("${linux_dir}"/*-"${dnf_version}"-*.rpm)
  nupkg_files=("${choco_dir}"/*."${choco_version}".nupkg)
  shopt -u nullglob

  [ "${#deb_files[@]}" -gt 0 ] || packaging_die "no deb packages found for version ${apt_version} in ${linux_dir}"
  [ "${#rpm_files[@]}" -gt 0 ] || packaging_die "no rpm packages found for version ${dnf_version} in ${linux_dir}"
  [ "${#nupkg_files[@]}" -gt 0 ] || packaging_die "no nupkg files found for version ${choco_version} in ${choco_dir}"
  packaging_require_file "${formula_path}"
  packaging_require_file "${nuspec_path}"

  linux_amd64_sha="$(jq -r '.artifacts[] | select(.name=="surveiller-linux-amd64") | .sha256' "${manifest_abs}")"
  linux_arm64_sha="$(jq -r '.artifacts[] | select(.name=="surveiller-linux-arm64") | .sha256' "${manifest_abs}")"
  darwin_amd64_sha="$(jq -r '.artifacts[] | select(.name=="surveiller-darwin-amd64") | .sha256' "${manifest_abs}")"
  darwin_arm64_sha="$(jq -r '.artifacts[] | select(.name=="surveiller-darwin-arm64") | .sha256' "${manifest_abs}")"

  for pkg in "${deb_files[@]}"; do
    assert_checksum_entry_matches "${checksums_abs}" "${dist_dir_abs}" "${pkg}"

    if command -v dpkg-deb >/dev/null 2>&1; then
      local deb_name deb_version
      deb_name="$(dpkg-deb -f "${pkg}" Package)"
      deb_version="$(dpkg-deb -f "${pkg}" Version)"
      [ "${deb_name}" = "surveiller" ] || packaging_die "deb metadata package name mismatch in ${pkg}: ${deb_name}"
      [ "${deb_version}" = "${apt_version}" ] || packaging_die "deb metadata version mismatch in ${pkg}: ${deb_version}"
    fi
  done

  for pkg in "${rpm_files[@]}"; do
    assert_checksum_entry_matches "${checksums_abs}" "${dist_dir_abs}" "${pkg}"

    if command -v rpm >/dev/null 2>&1; then
      local rpm_name rpm_version rpm_license
      rpm_name="$(rpm -qp --queryformat '%{NAME}' "${pkg}")"
      rpm_version="$(rpm -qp --queryformat '%{VERSION}' "${pkg}")"
      rpm_license="$(rpm -qp --queryformat '%{LICENSE}' "${pkg}")"
      [ "${rpm_name}" = "surveiller" ] || packaging_die "rpm metadata package name mismatch in ${pkg}: ${rpm_name}"
      [ "${rpm_version}" = "${dnf_version}" ] || packaging_die "rpm metadata version mismatch in ${pkg}: ${rpm_version}"
      [ "${rpm_license}" = "MIT" ] || packaging_die "rpm metadata license mismatch in ${pkg}: ${rpm_license}"
    fi
  done

  assert_checksum_entry_matches "${checksums_abs}" "${dist_dir_abs}" "${formula_path}"
  for sha in "${linux_amd64_sha}" "${linux_arm64_sha}" "${darwin_amd64_sha}" "${darwin_arm64_sha}"; do
    [ -n "${sha}" ] || packaging_die "manifest missing required binary sha for formula rendering"
    grep -q "${sha}" "${formula_path}" || packaging_die "formula missing expected sha256: ${sha}"
  done
  grep -q "version \"${brew_version}\"" "${formula_path}" || packaging_die "formula missing version ${brew_version}"
  grep -q "license \"MIT\"" "${formula_path}" || packaging_die "formula missing MIT license metadata"

  for pkg in "${nupkg_files[@]}"; do
    assert_checksum_entry_matches "${checksums_abs}" "${dist_dir_abs}" "${pkg}"
  done
  grep -q "<version>${choco_version}</version>" "${nuspec_path}" || packaging_die "nuspec missing version ${choco_version}"
  grep -q "<licenseUrl>" "${nuspec_path}" || packaging_die "nuspec missing license metadata"
  grep -q "releases/tag/v${release_version_no_v}" "${nuspec_path}" || packaging_die "nuspec release notes URL does not match version ${release_version}"

  packaging_log_info "package artifact verification passed"
}

main "$@"
