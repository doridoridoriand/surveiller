#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  publish_homebrew_tap.sh [options]

Options:
  --version <tag>         Release tag (e.g. v1.2.3)
  --dist-dir <path>       Dist directory (default: dist)
  --formula <path>        Formula source path (default: <dist>/packages/homebrew/surveiller.rb)
  --tap-path <path>       Local tap checkout path (default: <dist>/published/homebrew-tap)
  --status-file <path>    Status output JSON (default: <dist>/publication-status/brew.json)
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

ensure_tap_checkout() {
  local tap_path="$1"
  local tap_repo="$2"
  local push_token="${3:-}"

  if [ -d "${tap_path}/.git" ]; then
    return 0
  fi

  [ -n "${tap_repo}" ] || return 0

  local clone_url
  if [ -n "${push_token}" ]; then
    clone_url="https://x-access-token:${push_token}@github.com/${tap_repo}.git"
  else
    clone_url="https://github.com/${tap_repo}.git"
  fi

  mkdir -p "$(dirname "${tap_path}")"
  git clone "${clone_url}" "${tap_path}" >/dev/null 2>&1
}

main() {
  local repo_root version dist_dir formula_path tap_path status_file dry_run attempt
  local dist_abs formula_abs tap_abs status_abs tap_repo tap_token
  local state published_at error_message artifact_count target

  repo_root="$(packaging_repo_root)"
  version="${VERSION:-}"
  dist_dir="dist"
  formula_path=""
  tap_path="${HOMEBREW_TAP_PATH:-}"
  status_file=""
  dry_run="${DRY_RUN:-true}"
  attempt="${ATTEMPT:-1}"
  tap_repo="${HOMEBREW_TAP_REPO:-}"
  tap_token="${HOMEBREW_TAP_PUSH_TOKEN:-}"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version)
        version="${2:-}"
        shift 2
        ;;
      --dist-dir)
        dist_dir="${2:-}"
        shift 2
        ;;
      --formula)
        formula_path="${2:-}"
        shift 2
        ;;
      --tap-path)
        tap_path="${2:-}"
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

  if [ -z "${version}" ]; then
    version="$(packaging_default_version)"
  fi

  if [ -z "${formula_path}" ]; then
    formula_path="${dist_dir}/packages/homebrew/surveiller.rb"
  fi
  if [ -z "${tap_path}" ]; then
    tap_path="${dist_dir}/published/homebrew-tap"
  fi
  if [ -z "${status_file}" ]; then
    status_file="${dist_dir}/publication-status/brew.json"
  fi

  dist_abs="$(to_abs_path "${repo_root}" "${dist_dir}")"
  formula_abs="$(to_abs_path "${repo_root}" "${formula_path}")"
  tap_abs="$(to_abs_path "${repo_root}" "${tap_path}")"
  status_abs="$(to_abs_path "${repo_root}" "${status_file}")"

  packaging_require_file "${formula_abs}"
  artifact_count=1

  state="failed"
  published_at=""
  error_message=""
  target="${tap_repo:-${tap_abs}}"

  if packaging_bool_is_true "${dry_run}"; then
    state="queued"
  else
    set +e
    {
      ensure_tap_checkout "${tap_abs}" "${tap_repo}" "${tap_token}"
      mkdir -p "${tap_abs}/Formula"
      cp -f "${formula_abs}" "${tap_abs}/Formula/surveiller.rb"

      if [ -d "${tap_abs}/.git" ]; then
        (
          cd "${tap_abs}"
          git add Formula/surveiller.rb
          if ! git diff --cached --quiet; then
            git -c user.name="surveiller-bot" -c user.email="surveiller-bot@users.noreply.github.com" \
              commit -m "chore: update surveiller formula ${version}" >/dev/null
            if packaging_bool_is_true "${HOMEBREW_TAP_PUSH:-false}"; then
              git push origin HEAD >/dev/null
            fi
          fi
        )
      fi
    }
    rc=$?
    set -e

    if [ "${rc}" -eq 0 ]; then
      state="published"
      published_at="$(packaging_utc_now)"
    else
      state="failed"
      error_message="homebrew tap publish failed (exit=${rc})"
    fi
  fi

  packaging_write_publication_status \
    "${status_abs}" \
    "brew" \
    "${version}" \
    "${state}" \
    "${target}" \
    "${attempt}" \
    "${artifact_count}" \
    "${published_at}" \
    "${error_message}"

  if [ "${state}" = "failed" ]; then
    packaging_die "${error_message}"
  fi

  packaging_log_info "homebrew publication state=${state} status_file=${status_abs}"
}

main "$@"
