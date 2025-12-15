#!/bin/bash
#
# Shared helpers for deploy scripts (logging, colors, guards, small utils).
#

set -euo pipefail

# Default ansible behavior: don't prompt on first-time host key.
# You can override by setting ANSIBLE_HOST_KEY_CHECKING=True in your environment.
export ANSIBLE_HOST_KEY_CHECKING="${ANSIBLE_HOST_KEY_CHECKING:-False}"

# Colors (disabled if NO_COLOR is set or stdout not a TTY)
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  COLOR_RED=$'\033[0;31m'
  COLOR_GREEN=$'\033[0;32m'
  COLOR_YELLOW=$'\033[1;33m'
  COLOR_BLUE=$'\033[0;34m'
  COLOR_CYAN=$'\033[0;36m'
  COLOR_BOLD=$'\033[1m'
  COLOR_RESET=$'\033[0m'
else
  COLOR_RED=""
  COLOR_GREEN=""
  COLOR_YELLOW=""
  COLOR_BLUE=""
  COLOR_CYAN=""
  COLOR_BOLD=""
  COLOR_RESET=""
fi

log_info()  { echo "${COLOR_CYAN}[INFO]${COLOR_RESET} $*"; }
log_warn()  { echo "${COLOR_YELLOW}[WARN]${COLOR_RESET} $*"; }
log_error() { echo "${COLOR_RED}[ERROR]${COLOR_RESET} $*" >&2; }
log_ok()    { echo "${COLOR_GREEN}[OK]${COLOR_RESET} $*"; }

die() { log_error "$*"; exit 1; }

require_cmd() {
  local c="$1"
  command -v "$c" >/dev/null 2>&1 || die "Command not found in PATH: $c"
}

require_file() {
  local f="$1"
  [[ -f "$f" ]] || die "File not found: $f"
}

script_dir() {
  cd "$(dirname "${BASH_SOURCE[1]}")" && pwd
}

join_by() {
  local d="$1"
  shift
  local first=true
  local out=""
  local x
  for x in "$@"; do
    if [[ "$first" == true ]]; then
      out="$x"
      first=false
    else
      out="${out}${d}${x}"
    fi
  done
  echo "$out"
}

mktemp_compat() {
  # Prefer mktemp, fallback to $RANDOM.
  local template="${1:-.tmpXXXXXX}"
  if command -v mktemp >/dev/null 2>&1; then
    mktemp "$template"
    return 0
  fi

local_rand="${RANDOM}${RANDOM}"
  echo "${template/XXXXXX/${local_rand}}"
}

trim() {
  local s="$1"
  # shellcheck disable=SC2001
  s="$(echo "$s" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
  echo "$s"
}

load_dotenv() {
  # Very small .env loader: only supports KEY=VALUE lines; ignores comments/blank lines.
  # Does NOT execute arbitrary code.
  local file="$1"
  [[ -f "$file" ]] || return 0

  while IFS= read -r line || [[ -n "$line" ]]; do
    line="$(trim "$line")"
    [[ -z "$line" ]] && continue
    [[ "$line" == \#* ]] && continue
    if [[ "$line" != *"="* ]]; then
      continue
    fi
    local key="${line%%=*}"
    local value="${line#*=}"
    key="$(trim "$key")"
    value="$(trim "$value")"
    # Strip optional surrounding quotes
    if [[ "$value" == \"*\" && "$value" == *\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi
    if [[ -n "$key" ]]; then
      export "$key=$value"
    fi
  done < "$file"
}


