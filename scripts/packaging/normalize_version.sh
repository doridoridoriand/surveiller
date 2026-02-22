#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/packaging/lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'EOF'
Usage:
  normalize_version.sh <manager> <version>
  normalize_version.sh --manager <manager> --version <version>

Managers:
  generic | apt | deb | dnf | rpm | brew | choco | nuget
EOF
}

sanitize_segment() {
  local raw="$1"
  printf '%s' "${raw}" \
    | tr -c '0-9A-Za-z._~-' '.' \
    | sed -E 's/[.]{2,}/./g; s/^\.//; s/\.$//'
}

normalize_version_for_manager() {
  local manager="$1"
  local version="$2"

  if [[ ! "${version}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
    packaging_die "invalid semantic version: ${version}"
  fi

  local normalized prerelease build base
  normalized="${version#v}"

  build=""
  if [[ "${normalized}" == *"+"* ]]; then
    build="${normalized#*+}"
    normalized="${normalized%%+*}"
  fi

  prerelease=""
  if [[ "${normalized}" == *"-"* ]]; then
    prerelease="${normalized#*-}"
    base="${normalized%%-*}"
  else
    base="${normalized}"
  fi

  prerelease="$(sanitize_segment "${prerelease}")"
  build="$(sanitize_segment "${build}")"

  case "${manager}" in
    generic|brew)
      printf '%s' "${base}"
      if [ -n "${prerelease}" ]; then
        printf -- '-%s' "${prerelease}"
      fi
      ;;
    apt|deb)
      printf '%s' "${base}"
      if [ -n "${prerelease}" ]; then
        printf -- '~%s' "${prerelease}"
      fi
      if [ -n "${build}" ]; then
        printf -- '+%s' "${build}"
      fi
      ;;
    dnf|rpm)
      printf '%s' "${base}"
      if [ -n "${prerelease}" ]; then
        printf -- '~%s' "${prerelease}"
      fi
      if [ -n "${build}" ]; then
        printf -- '.%s' "${build}"
      fi
      ;;
    choco|nuget)
      printf '%s' "${base}"
      if [ -n "${prerelease}" ]; then
        printf -- '-%s' "${prerelease}"
      fi
      if [ -n "${build}" ]; then
        printf -- '.%s' "${build}"
      fi
      ;;
    *)
      packaging_die "unsupported manager: ${manager}"
      ;;
  esac
}

main() {
  local manager="" version=""

  if [ "$#" -eq 2 ] && [[ "$1" != -* ]]; then
    manager="$1"
    version="$2"
  else
    while [ "$#" -gt 0 ]; do
      case "$1" in
        --manager)
          manager="${2:-}"
          shift 2
          ;;
        --version)
          version="${2:-}"
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
  fi

  if [ -z "${manager}" ] || [ -z "${version}" ]; then
    usage
    exit 1
  fi

  normalize_version_for_manager "${manager}" "${version}"
  printf '\n'
}

main "$@"
