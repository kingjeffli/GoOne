#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<EOF
${COLOR_BOLD}GoOne Deploy Installer${COLOR_RESET}

Usage:
  $0 help
  $0 doctor
  $0 ansible [--venv <dir>]

Notes:
  - This is for the CONTROL machine (where you run ansible-playbook), not the game servers.
  - On Windows, prefer WSL/Git-Bash + Python. If venv layout differs, this script will print the next steps.
EOF
}

detect_python() {
  if command -v python3 >/dev/null 2>&1; then
    echo "python3"
    return 0
  fi
  if command -v python >/dev/null 2>&1; then
    echo "python"
    return 0
  fi
  return 1
}

venv_python_path() {
  local venv_dir="$1"
  if [[ -x "${venv_dir}/bin/python" ]]; then
    echo "${venv_dir}/bin/python"
    return 0
  fi
  if [[ -x "${venv_dir}/Scripts/python.exe" ]]; then
    echo "${venv_dir}/Scripts/python.exe"
    return 0
  fi
  return 1
}

cmd="${1:-help}"
shift || true

case "$cmd" in
  help|-h|--help)
    usage
    exit 0
    ;;

  doctor)
    log_info "Checking tools..."
    if command -v ansible-playbook >/dev/null 2>&1; then
      log_ok "ansible-playbook: OK"
    else
      log_warn "ansible-playbook: NOT FOUND"
    fi
    if detect_python >/dev/null 2>&1; then
      log_ok "python: $(detect_python)"
    else
      log_warn "python: NOT FOUND"
    fi
    exit 0
    ;;

  ansible)
    ;;

  *)
    die "Unknown command: $cmd (use: help)"
    ;;
esac

VENV_DIR="${SCRIPT_DIR}/.venv-ansible"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --venv)
      VENV_DIR="${2:-}"
      [[ -n "$VENV_DIR" ]] || die "Missing value for --venv"
      shift 2
      ;;
    *)
      die "Unknown argument: $1"
      ;;
  esac
done

py="$(detect_python)" || die "python not found (need python3/python on PATH)"
log_info "Creating venv: $VENV_DIR"
"$py" -m venv "$VENV_DIR"

vp="$(venv_python_path "$VENV_DIR")" || die "Cannot locate venv python under: $VENV_DIR"
log_info "Upgrading pip..."
"$vp" -m pip install --upgrade pip
log_info "Installing ansible..."
"$vp" -m pip install --upgrade ansible

log_ok "Installed ansible into venv: $VENV_DIR"
log_info "Next: run ansible-playbook using venv python, e.g.:"
log_info "  $vp -m ansible.playbook --version"


