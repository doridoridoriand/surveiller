#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  smoke_brew.sh [options]

Options:
  --manifest <path>        Release manifest path (default: dist/release-manifest.json)
  --dist-dir <path>        Dist directory (default: dist)
  --manifest-out <path>    Local manifest output path (default: <dist>/release-manifest.brew-smoke.json)
  --formula-out <path>     Local formula output path (default: <dist>/packages/homebrew/surveiller.smoke.rb)
  --tap-path <path>        Local tap path (default: <dist>/published/homebrew-tap-smoke)
  --tap-name <name>        Brew tap name (default: surveiller/tap-smoke)
  --package <name>         Formula package name (default: surveiller)
  -h, --help               Show this help
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
  local repo_root manifest_path dist_dir local_manifest_path formula_out tap_path tap_name package_name
  local manifest_abs dist_abs local_manifest_abs formula_abs tap_abs
  local generate_script render_script version commit_sha repository release_date release_base_url checksums_abs schema_abs

  repo_root="$(packaging_repo_root)"
  manifest_path="dist/release-manifest.json"
  dist_dir="dist"
  local_manifest_path=""
  formula_out=""
  tap_path=""
  tap_name="${HOMEBREW_TAP_NAME:-surveiller/tap-smoke}"
  package_name="surveiller"

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
      --manifest-out)
        local_manifest_path="${2:-}"
        shift 2
        ;;
      --formula-out)
        formula_out="${2:-}"
        shift 2
        ;;
      --tap-path)
        tap_path="${2:-}"
        shift 2
        ;;
      --tap-name)
        tap_name="${2:-}"
        shift 2
        ;;
      --package)
        package_name="${2:-}"
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

  if [ -z "${local_manifest_path}" ]; then
    local_manifest_path="${dist_dir}/release-manifest.brew-smoke.json"
  fi
  if [ -z "${formula_out}" ]; then
    formula_out="${dist_dir}/packages/homebrew/surveiller.smoke.rb"
  fi
  if [ -z "${tap_path}" ]; then
    tap_path="${dist_dir}/published/homebrew-tap-smoke"
  fi

  manifest_abs="$(to_abs_path "${repo_root}" "${manifest_path}")"
  dist_abs="$(to_abs_path "${repo_root}" "${dist_dir}")"
  local_manifest_abs="$(to_abs_path "${repo_root}" "${local_manifest_path}")"
  formula_abs="$(to_abs_path "${repo_root}" "${formula_out}")"
  tap_abs="$(to_abs_path "${repo_root}" "${tap_path}")"

  generate_script="${SCRIPT_DIR}/../generate_release_manifest.sh"
  render_script="${SCRIPT_DIR}/../render_homebrew_formula.sh"
  checksums_abs="${dist_abs}/checksums.txt"
  schema_abs="${repo_root}/packaging/release-manifest.schema.json"

  packaging_require_file "${manifest_abs}"
  packaging_require_file "${checksums_abs}"
  packaging_require_file "${schema_abs}"
  packaging_require_executable "${generate_script}"
  packaging_require_executable "${render_script}"
  packaging_require_command brew
  packaging_require_command jq
  packaging_require_command git

  version="$(jq -r '.version // empty' "${manifest_abs}")"
  commit_sha="$(jq -r '.commit_sha // empty' "${manifest_abs}")"
  repository="$(jq -r '.repository // empty' "${manifest_abs}")"
  [ -n "${version}" ] || packaging_die "manifest missing version"
  [ -n "${commit_sha}" ] || packaging_die "manifest missing commit_sha"
  [ -n "${repository}" ] || packaging_die "manifest missing repository"

  release_date="$(packaging_utc_now)"
  release_base_url="file://${dist_abs}"

  "${generate_script}" \
    --version "${version}" \
    --commit-sha "${commit_sha}" \
    --release-date "${release_date}" \
    --repository "${repository}" \
    --release-base-url "${release_base_url}" \
    --dist-dir "${dist_abs}" \
    --checksums-file "${checksums_abs}" \
    --schema "${schema_abs}" \
    --output "${local_manifest_abs}"

  "${render_script}" \
    --manifest "${local_manifest_abs}" \
    --dist-dir "${dist_abs}" \
    --output "${formula_abs}" \
    --checksums-file "${checksums_abs}"

  rm -rf "${tap_abs}"
  mkdir -p "${tap_abs}/Formula"
  cp -f "${formula_abs}" "${tap_abs}/Formula/${package_name}.rb"

  (
    cd "${tap_abs}"
    git init -q
    git add "Formula/${package_name}.rb"
    git -c user.name="surveiller-smoke" -c user.email="surveiller-smoke@local" commit -qm "smoke formula"
  )

  cleanup() {
    brew uninstall --ignore-dependencies --force "${package_name}" >/dev/null 2>&1 || true
    brew untap "${tap_name}" >/dev/null 2>&1 || true
  }
  trap cleanup EXIT

  brew untap "${tap_name}" >/dev/null 2>&1 || true
  brew tap "${tap_name}" "${tap_abs}"
  brew install "${tap_name}/${package_name}"

  "${package_name}" -version >/dev/null
  packaging_log_info "brew smoke install passed for formula ${tap_name}/${package_name}"
}

main "$@"
