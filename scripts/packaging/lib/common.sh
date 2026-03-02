#!/usr/bin/env bash

# Shared helpers for packaging scripts.

packaging_log_info() {
  printf '[packaging] %s\n' "$*"
}

packaging_log_warn() {
  printf '[packaging] WARN: %s\n' "$*" >&2
}

packaging_log_error() {
  printf '[packaging] ERROR: %s\n' "$*" >&2
}

packaging_die() {
  packaging_log_error "$*"
  exit 1
}

packaging_repo_root() {
  if git rev-parse --show-toplevel >/dev/null 2>&1; then
    git rev-parse --show-toplevel
  else
    pwd
  fi
}

packaging_utc_now() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

packaging_default_repository() {
  if [ -n "${GITHUB_REPOSITORY:-}" ]; then
    printf '%s\n' "${GITHUB_REPOSITORY}"
    return 0
  fi

  local remote_url
  remote_url="$(git config --get remote.origin.url 2>/dev/null || true)"

  case "${remote_url}" in
    https://github.com/*)
      remote_url="${remote_url#https://github.com/}"
      remote_url="${remote_url%.git}"
      printf '%s\n' "${remote_url}"
      return 0
      ;;
    git@github.com:*)
      remote_url="${remote_url#git@github.com:}"
      remote_url="${remote_url%.git}"
      printf '%s\n' "${remote_url}"
      return 0
      ;;
  esac

  packaging_die "could not determine repository; set GITHUB_REPOSITORY explicitly"
}

packaging_default_version() {
  if [ -n "${VERSION:-}" ]; then
    printf '%s\n' "${VERSION}"
    return 0
  fi

  if [ -n "${GITHUB_REF_NAME:-}" ]; then
    printf '%s\n' "${GITHUB_REF_NAME}"
    return 0
  fi

  if [ -n "${GITHUB_REF:-}" ] && [[ "${GITHUB_REF}" == refs/tags/* ]]; then
    printf '%s\n' "${GITHUB_REF#refs/tags/}"
    return 0
  fi

  local described
  described="$(git describe --tags --always --dirty 2>/dev/null || true)"
  if [ -n "${described}" ]; then
    printf '%s\n' "${described}"
    return 0
  fi

  packaging_die "could not determine release version; set VERSION explicitly"
}

packaging_release_base_url() {
  local repository="$1"
  local version="$2"
  printf 'https://github.com/%s/releases/download/%s\n' "${repository}" "${version}"
}

packaging_json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '%s' "${value}"
}

packaging_validate_sha256() {
  local value="$1"
  [[ "${value}" =~ ^[a-fA-F0-9]{64}$ ]]
}

packaging_require_file() {
  local file="$1"
  [ -f "${file}" ] || packaging_die "required file not found: ${file}"
}

packaging_require_executable() {
  local file="$1"
  [ -x "${file}" ] || packaging_die "required executable not found: ${file}"
}

packaging_require_command() {
  local command_name="$1"
  command -v "${command_name}" >/dev/null 2>&1 || \
    packaging_die "required command is not installed: ${command_name}"
}

packaging_sha256_file() {
  local file="$1"
  packaging_require_file "${file}"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print tolower($1)}'
    return 0
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${file}" | awk '{print tolower($1)}'
    return 0
  fi

  packaging_die "sha256 command not found (expected sha256sum or shasum)"
}

packaging_upsert_checksum() {
  local checksums_file="$1"
  local artifact_name="$2"
  local checksum="$3"

  if ! packaging_validate_sha256 "${checksum}"; then
    packaging_die "invalid sha256 value for ${artifact_name}: ${checksum}"
  fi

  mkdir -p "$(dirname "${checksums_file}")"

  local tmp
  tmp="$(mktemp)"
  if [ -f "${checksums_file}" ]; then
    awk -v target="${artifact_name}" '
      $1 ~ /^[a-fA-F0-9]{64}$/ {
        file=$0
        sub(/^[a-fA-F0-9]{64}[[:space:]]+\*?/, "", file)
        if (file == target) {
          next
        }
      }
      { print }
    ' "${checksums_file}" > "${tmp}"
  else
    : > "${tmp}"
  fi

  printf '%s  %s\n' "${checksum}" "${artifact_name}" >> "${tmp}"
  mv "${tmp}" "${checksums_file}"
}

packaging_sort_checksums_file() {
  local checksums_file="$1"
  [ -f "${checksums_file}" ] || return 0

  local tmp
  tmp="$(mktemp)"
  awk '
    $1 ~ /^[a-fA-F0-9]{64}$/ {
      sha=tolower($1)
      file=$0
      sub(/^[a-fA-F0-9]{64}[[:space:]]+\*?/, "", file)
      print file "\t" sha
    }
  ' "${checksums_file}" | LC_ALL=C sort | awk -F '\t' '{print $2 "  " $1}' > "${tmp}"
  mv "${tmp}" "${checksums_file}"
}

packaging_lookup_checksum() {
  local checksums_file="$1"
  local target_file="$2"
  awk -v target_file="${target_file}" '
    $1 ~ /^[a-fA-F0-9]{64}$/ {
      file=$0
      sub(/^[a-fA-F0-9]{64}[[:space:]]+\*?/, "", file)
      if (file == target_file) {
        print tolower($1)
        exit 0
      }
    }
  ' "${checksums_file}"
}

packaging_sorted_checksums() {
  local checksums_file="$1"
  awk '
    $1 ~ /^[a-fA-F0-9]{64}$/ {
      sha=tolower($1)
      file=$0
      sub(/^[a-fA-F0-9]{64}[[:space:]]+\*?/, "", file)
      print file "\t" sha
    }
  ' "${checksums_file}" | LC_ALL=C sort
}
