#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  build_linux_packages.sh [options]

Options:
  --manifest <path>       Release manifest path (default: dist/release-manifest.json)
  --config <path>         nfpm config path (default: packaging/nfpm/nfpm.yaml)
  --dist-dir <path>       Dist directory root (default: dist)
  --output-dir <path>     Linux package output dir (default: <dist>/packages/linux)
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

build_package() {
  local packager="$1"
  local arch="$2"
  local version="$3"
  local repo_root="$4"
  local dist_dir_abs="$5"
  local output_dir_abs="$6"
  local nfpm_config_abs="$7"
  local checksums_file_abs="$8"
  local binary_path target_path artifact_name artifact_sha rpm_arch rendered_config

  binary_path="${dist_dir_abs}/surveiller-linux-${arch}"
  packaging_require_file "${binary_path}"

  case "${packager}" in
    deb)
      target_path="${output_dir_abs}/surveiller_${version}_${arch}.deb"
      ;;
    rpm)
      rpm_arch="${arch}"
      if [ "${arch}" = "amd64" ]; then
        rpm_arch="x86_64"
      elif [ "${arch}" = "arm64" ]; then
        rpm_arch="aarch64"
      fi
      target_path="${output_dir_abs}/surveiller-${version}-1.${rpm_arch}.rpm"
      ;;
    *)
      packaging_die "unsupported packager: ${packager}"
      ;;
  esac

  rendered_config="$(mktemp)"
  sed \
    -e "s/__NFPM_ARCH__/${arch}/g" \
    -e "s/__NFPM_VERSION__/${version}/g" \
    -e "s/__SURVEILLER_BINARY_ARCH__/${arch}/g" \
    "${nfpm_config_abs}" > "${rendered_config}"

  (
    cd "${repo_root}"
    nfpm package --packager "${packager}" --config "${rendered_config}" --target "${target_path}"
  )
  rm -f "${rendered_config}"

  artifact_name="$(relative_to_dist "${dist_dir_abs}" "${target_path}")"
  artifact_sha="$(packaging_sha256_file "${target_path}")"
  packaging_upsert_checksum "${checksums_file_abs}" "${artifact_name}" "${artifact_sha}"
  packaging_log_info "built ${packager} package: ${artifact_name}"
}

main() {
  local repo_root manifest_path nfpm_config_path dist_dir output_dir checksums_file
  local manifest_abs nfpm_config_abs dist_dir_abs output_dir_abs checksums_file_abs
  local apt_version dnf_version

  repo_root="$(packaging_repo_root)"
  manifest_path="dist/release-manifest.json"
  nfpm_config_path="packaging/nfpm/nfpm.yaml"
  dist_dir="dist"
  output_dir=""
  checksums_file=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --manifest)
        manifest_path="${2:-}"
        shift 2
        ;;
      --config)
        nfpm_config_path="${2:-}"
        shift 2
        ;;
      --dist-dir)
        dist_dir="${2:-}"
        shift 2
        ;;
      --output-dir)
        output_dir="${2:-}"
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

  if [ -z "${output_dir}" ]; then
    output_dir="${dist_dir}/packages/linux"
  fi
  if [ -z "${checksums_file}" ]; then
    checksums_file="${dist_dir}/checksums.txt"
  fi

  manifest_abs="$(to_abs_path "${repo_root}" "${manifest_path}")"
  nfpm_config_abs="$(to_abs_path "${repo_root}" "${nfpm_config_path}")"
  dist_dir_abs="$(to_abs_path "${repo_root}" "${dist_dir}")"
  output_dir_abs="$(to_abs_path "${repo_root}" "${output_dir}")"
  checksums_file_abs="$(to_abs_path "${repo_root}" "${checksums_file}")"

  packaging_require_file "${manifest_abs}"
  packaging_require_file "${nfpm_config_abs}"
  packaging_require_command jq
  packaging_require_command nfpm

  mkdir -p "${output_dir_abs}"
  if [ ! -f "${checksums_file_abs}" ]; then
    : > "${checksums_file_abs}"
  fi

  apt_version="$(jq -r '.package_versions.apt // empty' "${manifest_abs}")"
  dnf_version="$(jq -r '.package_versions.dnf // empty' "${manifest_abs}")"

  [ -n "${apt_version}" ] || packaging_die "manifest missing package_versions.apt"
  [ -n "${dnf_version}" ] || packaging_die "manifest missing package_versions.dnf"

  build_package deb amd64 "${apt_version}" "${repo_root}" "${dist_dir_abs}" "${output_dir_abs}" "${nfpm_config_abs}" "${checksums_file_abs}"
  build_package deb arm64 "${apt_version}" "${repo_root}" "${dist_dir_abs}" "${output_dir_abs}" "${nfpm_config_abs}" "${checksums_file_abs}"
  build_package rpm amd64 "${dnf_version}" "${repo_root}" "${dist_dir_abs}" "${output_dir_abs}" "${nfpm_config_abs}" "${checksums_file_abs}"
  build_package rpm arm64 "${dnf_version}" "${repo_root}" "${dist_dir_abs}" "${output_dir_abs}" "${nfpm_config_abs}" "${checksums_file_abs}"

  packaging_sort_checksums_file "${checksums_file_abs}"
  packaging_log_info "linux packages generated in ${output_dir_abs}"
}

main "$@"
