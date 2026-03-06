#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  publish_dnf_repo.sh [options]

Options:
  --version <tag>         Release tag (e.g. v1.2.3)
  --manifest <path>       Release manifest path (default: dist/release-manifest.json)
  --dist-dir <path>       Dist directory (default: dist)
  --repo-dir <path>       DNF repo output directory (default: <dist>/published/dnf)
  --status-file <path>    Status output JSON (default: <dist>/publication-status/dnf.json)
  --dry-run <bool>        Dry run mode (default: true)
  --attempt <number>      Attempt number (default: 1)
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

main() {
  local repo_root version manifest_path dist_dir repo_dir status_file dry_run attempt
  local manifest_abs dist_abs repo_abs status_abs dnf_version
  local state published_at error_message artifact_count
  local rpm_files packages_dir

  repo_root="$(packaging_repo_root)"
  version="${VERSION:-$(packaging_default_version)}"
  manifest_path="dist/release-manifest.json"
  dist_dir="dist"
  repo_dir=""
  status_file=""
  dry_run="${DRY_RUN:-true}"
  attempt="${ATTEMPT:-1}"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version)
        version="${2:-}"
        shift 2
        ;;
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
      --dry-run)
        dry_run="${2:-}"
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
    repo_dir="${dist_dir}/published/dnf"
  fi
  if [ -z "${status_file}" ]; then
    status_file="${dist_dir}/publication-status/dnf.json"
  fi

  manifest_abs="$(to_abs_path "${repo_root}" "${manifest_path}")"
  dist_abs="$(to_abs_path "${repo_root}" "${dist_dir}")"
  repo_abs="$(to_abs_path "${repo_root}" "${repo_dir}")"
  status_abs="$(to_abs_path "${repo_root}" "${status_file}")"

  packaging_require_file "${manifest_abs}"
  packaging_require_command jq

  dnf_version="$(jq -r '.package_versions.dnf // empty' "${manifest_abs}")"
  [ -n "${dnf_version}" ] || packaging_die "manifest missing package_versions.dnf"

  shopt -s nullglob
  rpm_files=("${dist_abs}/packages/linux"/*-"${dnf_version}"-*.rpm)
  shopt -u nullglob
  artifact_count="${#rpm_files[@]}"
  [ "${artifact_count}" -gt 0 ] || packaging_die "no rpm artifacts for dnf version ${dnf_version}"

  state="failed"
  published_at=""
  error_message=""

  if packaging_bool_is_true "${dry_run}"; then
    state="queued"
    error_message=""
  else
    set +e
    (
      set -e
      packages_dir="${repo_abs}/packages"
      mkdir -p "${packages_dir}"
      for rpm_pkg in "${rpm_files[@]}"; do
        cp -f "${rpm_pkg}" "${packages_dir}/"
      done

      if command -v createrepo_c >/dev/null 2>&1; then
        createrepo_c --update "${packages_dir}" >/dev/null
      else
        packaging_log_warn "createrepo_c is not installed; repodata generation skipped"
      fi

      if [ -n "${RPM_GPG_PRIVATE_KEY:-}" ] && [ -f "${packages_dir}/repodata/repomd.xml" ]; then
        local gpg_home
        gpg_home="$(mktemp -d)"
        chmod 700 "${gpg_home}"
        printf '%s\n' "${RPM_GPG_PRIVATE_KEY}" | gpg --batch --homedir "${gpg_home}" --import >/dev/null 2>&1
        gpg --batch --yes --homedir "${gpg_home}" --pinentry-mode loopback \
          --passphrase "${RPM_GPG_PASSPHRASE:-}" \
          --output "${packages_dir}/repodata/repomd.xml.asc" \
          --detach-sign "${packages_dir}/repodata/repomd.xml"
        rm -rf "${gpg_home}"
      fi
    )
    rc=$?
    set -e

    if [ "${rc}" -eq 0 ]; then
      state="published"
      published_at="$(packaging_utc_now)"
    else
      state="failed"
      error_message="dnf repository publish failed (exit=${rc})"
    fi
  fi

  packaging_write_publication_status \
    "${status_abs}" \
    "dnf" \
    "${version}" \
    "${state}" \
    "${repo_abs}" \
    "${attempt}" \
    "${artifact_count}" \
    "${published_at}" \
    "${error_message}"

  if [ "${state}" = "failed" ]; then
    packaging_die "${error_message}"
  fi

  packaging_log_info "dnf publication state=${state} status_file=${status_abs}"
}

main "$@"
