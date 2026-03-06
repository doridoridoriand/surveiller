#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  smoke_apt.sh [options]

Options:
  --manifest <path>       Release manifest path (default: dist/release-manifest.json)
  --dist-dir <path>       Dist directory (default: dist)
  --repo-dir <path>       APT repo output directory (default: <dist>/published/apt-smoke)
  --status-file <path>    Publish status path (default: <dist>/publication-status/smoke-apt.json)
  --package <name>        Package name to install (default: surveiller)
  --attempt <number>      Attempt number for status payload (default: 1)
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

run_as_root() {
  if command -v sudo >/dev/null 2>&1; then
    sudo "$@"
  else
    "$@"
  fi
}

main() {
  local repo_root manifest_path dist_dir repo_dir status_file package_name attempt
  local manifest_abs dist_abs repo_abs status_abs publish_script version source_list

  repo_root="$(packaging_repo_root)"
  manifest_path="dist/release-manifest.json"
  dist_dir="dist"
  repo_dir=""
  status_file=""
  package_name="surveiller"
  attempt="${ATTEMPT:-1}"

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
      --repo-dir)
        repo_dir="${2:-}"
        shift 2
        ;;
      --status-file)
        status_file="${2:-}"
        shift 2
        ;;
      --package)
        package_name="${2:-}"
        shift 2
        ;;
      --attempt)
        attempt="${2:-}"
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

  if [ -z "${repo_dir}" ]; then
    repo_dir="${dist_dir}/published/apt-smoke"
  fi
  if [ -z "${status_file}" ]; then
    status_file="${dist_dir}/publication-status/smoke-apt.json"
  fi

  manifest_abs="$(to_abs_path "${repo_root}" "${manifest_path}")"
  dist_abs="$(to_abs_path "${repo_root}" "${dist_dir}")"
  repo_abs="$(to_abs_path "${repo_root}" "${repo_dir}")"
  status_abs="$(to_abs_path "${repo_root}" "${status_file}")"
  publish_script="${SCRIPT_DIR}/../publish_apt_repo.sh"

  packaging_require_file "${manifest_abs}"
  packaging_require_executable "${publish_script}"
  packaging_require_command jq
  packaging_require_command apt-get
  packaging_require_command apt-ftparchive

  version="$(jq -r '.version // empty' "${manifest_abs}")"
  [ -n "${version}" ] || packaging_die "manifest missing version"

  "${publish_script}" \
    --version "${version}" \
    --manifest "${manifest_abs}" \
    --dist-dir "${dist_abs}" \
    --repo-dir "${repo_abs}" \
    --dry-run false \
    --attempt "${attempt}" \
    --status-file "${status_abs}"

  source_list="/etc/apt/sources.list.d/surveiller-smoke.list"
  cleanup() {
    if [ -n "${source_list:-}" ]; then
      run_as_root rm -f "${source_list}" || true
    fi
  }
  trap cleanup EXIT

  printf 'deb [trusted=yes] file:%s stable main\n' "${repo_abs}" | run_as_root tee "${source_list}" >/dev/null

  run_as_root apt-get update
  run_as_root apt-get install -y "${package_name}"

  "${package_name}" -version >/dev/null
  packaging_log_info "apt smoke install passed for package ${package_name}"
}

main "$@"
