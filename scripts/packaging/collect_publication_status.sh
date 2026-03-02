#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_STATUS_LIST_FILE=""

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  collect_publication_status.sh [options]

Options:
  --status-dir <path>        Directory containing manager status JSON files
                             (default: dist/publication-status)
  --output <path>            Summary JSON path (default: <status-dir>/summary.json)
  --summary-md <path>        Summary markdown path (default: <status-dir>/summary.md)
  --require-managers <csv>   Required managers (default: apt,dnf,brew,choco)
  --github-output <path>     Write outputs for GitHub Actions
  --fail-on-failed <bool>    Exit non-zero when any manager is failed (default: true)
  -h, --help                 Show this help
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
  local repo_root status_dir output_json summary_md require_managers github_output fail_on_failed
  local status_abs output_abs summary_abs github_abs
  local managers_csv manager status_file list_file
  local total_count failed_count queued_count published_count failed_managers

  repo_root="$(packaging_repo_root)"
  status_dir="dist/publication-status"
  output_json=""
  summary_md=""
  require_managers="apt,dnf,brew,choco"
  github_output=""
  fail_on_failed="true"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --status-dir)
        status_dir="${2:-}"
        shift 2
        ;;
      --output)
        output_json="${2:-}"
        shift 2
        ;;
      --summary-md)
        summary_md="${2:-}"
        shift 2
        ;;
      --require-managers)
        require_managers="${2:-}"
        shift 2
        ;;
      --github-output)
        github_output="${2:-}"
        shift 2
        ;;
      --fail-on-failed)
        fail_on_failed="${2:-}"
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

  if [ -z "${output_json}" ]; then
    output_json="${status_dir}/summary.json"
  fi
  if [ -z "${summary_md}" ]; then
    summary_md="${status_dir}/summary.md"
  fi

  status_abs="$(to_abs_path "${repo_root}" "${status_dir}")"
  output_abs="$(to_abs_path "${repo_root}" "${output_json}")"
  summary_abs="$(to_abs_path "${repo_root}" "${summary_md}")"
  github_abs=""
  if [ -n "${github_output}" ]; then
    github_abs="$(to_abs_path "${repo_root}" "${github_output}")"
  fi

  packaging_require_command jq
  mkdir -p "${status_abs}" "$(dirname "${output_abs}")" "$(dirname "${summary_abs}")"

  managers_csv="$(printf '%s' "${require_managers}" | tr -d '[:space:]')"
  [ -n "${managers_csv}" ] || packaging_die "--require-managers cannot be empty"

  list_file="$(mktemp)"
  TMP_STATUS_LIST_FILE="${list_file}"
  trap 'rm -f "${TMP_STATUS_LIST_FILE}"' EXIT

  IFS=',' read -r -a manager_array <<< "${managers_csv}"
  for manager in "${manager_array[@]}"; do
    if [ "${manager}" = "choro" ]; then
      manager="choco"
    fi
    status_file="${status_abs}/${manager}.json"
    [ -f "${status_file}" ] || packaging_die "missing status file for manager ${manager}: ${status_file}"
    printf '%s\n' "${status_file}" >> "${list_file}"
  done

  jq -s \
    --arg generated_at "$(packaging_utc_now)" \
    '{
      generated_at: $generated_at,
      statuses: .
    }' $(cat "${list_file}") > "${output_abs}"

  total_count="$(jq -r '.statuses | length' "${output_abs}")"
  failed_count="$(jq -r '[.statuses[] | select(.state == "failed")] | length' "${output_abs}")"
  queued_count="$(jq -r '[.statuses[] | select(.state == "queued")] | length' "${output_abs}")"
  published_count="$(jq -r '[.statuses[] | select(.state == "published")] | length' "${output_abs}")"
  failed_managers="$(jq -r '[.statuses[] | select(.state == "failed") | .manager] | join(",")' "${output_abs}")"

  jq \
    --argjson total "${total_count}" \
    --argjson failed "${failed_count}" \
    --argjson queued "${queued_count}" \
    --argjson published "${published_count}" \
    --arg failed_managers "${failed_managers}" \
    '. + {
      counts: {
        total: $total,
        published: $published,
        queued: $queued,
        failed: $failed
      },
      failed_managers: (if $failed_managers == "" then [] else ($failed_managers | split(",")) end)
    }' "${output_abs}" > "${output_abs}.tmp"
  mv "${output_abs}.tmp" "${output_abs}"

  {
    printf '# Package Publication Summary\n\n'
    printf '| Manager | State | Attempt | Artifact Count | Target | Error |\n'
    printf '|---|---|---:|---:|---|---|\n'
    jq -r '.statuses[] | [
      .manager,
      .state,
      (.attempt // 0 | tostring),
      (.artifact_count // 0 | tostring),
      (.repository_target // ""),
      (.error_message // "")
    ] | @tsv' "${output_abs}" \
      | while IFS=$'\t' read -r col1 col2 col3 col4 col5 col6; do
          printf '| %s | %s | %s | %s | %s | %s |\n' "${col1}" "${col2}" "${col3}" "${col4}" "${col5}" "${col6}"
        done
    printf '\n'
    printf -- '- Total: %s\n' "${total_count}"
    printf -- '- Published: %s\n' "${published_count}"
    printf -- '- Queued: %s\n' "${queued_count}"
    printf -- '- Failed: %s\n' "${failed_count}"
  } > "${summary_abs}"

  if [ -n "${github_abs}" ]; then
    {
      printf 'failed_managers=%s\n' "${failed_managers}"
      if [ "${failed_count}" -eq 0 ]; then
        printf 'all_success=true\n'
      else
        printf 'all_success=false\n'
      fi
    } >> "${github_abs}"
  fi

  packaging_log_info "publication summary generated: ${output_abs}"
  packaging_log_info "publication summary markdown: ${summary_abs}"

  if packaging_bool_is_true "${fail_on_failed}" && [ "${failed_count}" -gt 0 ]; then
    packaging_die "publication failed for managers: ${failed_managers}"
  fi

  rm -f "${list_file}"
  TMP_STATUS_LIST_FILE=""
}

main "$@"
