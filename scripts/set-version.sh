#!/usr/bin/env bash
# Update root VERSION for geocoder builds (YYYY.MM.DD+BUILD).
#
# Usage:
#   scripts/set-version.sh                 # validate and normalize VERSION
#   scripts/set-version.sh 2026.09.02+14  # write VERSION then apply
#   scripts/set-version.sh --bump-build    # today's date, BUILD+1
#
# Format: YYYY.MM.DD+BUILD (zero-padded month and day).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION_FILE="$ROOT/VERSION"

normalize_version() {
  local raw=$1
  raw="${raw#v}"
  if [[ "$raw" =~ ^([0-9]{4})\.([0-9]{1,2})\.([0-9]{1,2})\+([0-9]+)$ ]]; then
    printf '%04d.%02d.%02d+%s' \
      "$((10#${BASH_REMATCH[1]}))" \
      "$((10#${BASH_REMATCH[2]}))" \
      "$((10#${BASH_REMATCH[3]}))" \
      "${BASH_REMATCH[4]}"
    return 0
  fi
  return 1
}

read_version_line() {
  if [[ ! -f "$VERSION_FILE" ]]; then
    echo "error: missing $VERSION_FILE" >&2
    return 1
  fi
  local line
  line="$(
    grep -v '^[[:space:]]*#' "$VERSION_FILE" | grep -v '^[[:space:]]*$' | head -n 1 | tr -d '[:space:]'
  )"
  line="${line#v}"
  if ! normalize_version "$line"; then
    echo "error: VERSION must be YYYY.MM.DD+BUILD (e.g. 2026.09.02+1), got: ${line:-<empty>}" >&2
    return 1
  fi
}

write_version_file() {
  local ver=$1
  {
    echo "# Product version (single source of truth)."
    echo "# Format: YYYY.MM.DD+BUILD  (zero-padded month/day + monotonic build number)"
    echo "#   Example: 2026.09.02+1"
    echo "$ver"
  } >"$VERSION_FILE"
}

if [[ $# -ge 1 ]]; then
  case "$1" in
    --bump-build | -b)
      current="$(read_version_line)"
      if [[ ! "$current" =~ ^([0-9]{4}\.[0-9]{2}\.[0-9]{2})\+([0-9]+)$ ]]; then
        echo "error: cannot parse VERSION for build bump: $current" >&2
        exit 1
      fi
      prev_marketing="${BASH_REMATCH[1]}"
      build="${BASH_REMATCH[2]}"
      next_build=$((10#$build + 1))
      marketing="$(date +%Y.%m.%d)"
      write_version_file "${marketing}+${next_build}"
      echo "bumped ${prev_marketing}+${build} → ${marketing}+${next_build}"
      ;;
    -*)
      echo "usage: $0 [YYYY.MM.DD+BUILD | --bump-build]" >&2
      exit 1
      ;;
    *)
      if ! normalized="$(normalize_version "$1")"; then
        echo "error: VERSION must be YYYY.MM.DD+BUILD (e.g. 2026.09.02+1), got: $1" >&2
        exit 1
      fi
      write_version_file "$normalized"
      ;;
  esac
fi

if [[ ! -f "$VERSION_FILE" ]]; then
  echo "error: missing $VERSION_FILE" >&2
  exit 1
fi

version="$(read_version_line)"
write_version_file "$version"

if [[ ! "$version" =~ ^([0-9]{4}\.[0-9]{2}\.[0-9]{2})\+([0-9]+)$ ]]; then
  echo "error: internal parse failed for: $version" >&2
  exit 1
fi

marketing="${BASH_REMATCH[1]}"
build="${BASH_REMATCH[2]}"

echo "VERSION ${version}"
echo "  date  = ${marketing}"
echo "  build = ${build}"
echo "  file  = ${VERSION_FILE}"
