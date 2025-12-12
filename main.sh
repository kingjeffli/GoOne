#!/bin/bash
# add by Iori 2020.9.1
# Maintained/optimized for robustness & usability

set -euo pipefail

# Root dir = script dir (not current working dir)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色支持（如果是非 TTY 或设置了 NO_COLOR，则自动关闭颜色）
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

require_file() {
  local f="$1"
  [[ -f "$f" ]] || die "File not found: $f"
}

require_cmd() {
  local c="$1"
  command -v "$c" >/dev/null 2>&1 || die "Command not found in PATH: $c"
}

print_header() {
  echo "${COLOR_BOLD}${COLOR_BLUE}========== GoOne Manager ==========${COLOR_RESET}"
  log_info "Root dir: $ROOT_DIR"
}

usage() {
  cat <<'EOF'
Usage:
  ./main.sh help
  ./main.sh doctor
  ./main.sh list envs
  ./main.sh list roles

  ./main.sh build [target]
    target: build.sh accepts {conn|main|mysql|db|gmconn|rcmd|info|game|opvp|mail|chat|friend|web|rank|guild}
    if omitted, build all

  ./main.sh deploy <env> <init|push|start|stop|restart|all> [role...]
    all = push + restart

Legacy (compat):
  ./main.sh all <env> [all|init|push|start|stop|restart] [role]
  ./main.sh dep <env> <init|push|start|stop|restart|all> [role]
  ./main.sh xls
  ./main.sh ptc

Env vars:
  NO_COLOR=1                    disable colors
  ANSIBLE_HOST_KEY_CHECKING=False (or set in deploy/deploy.sh)
EOF
}

list_envs() {
  local dir="$ROOT_DIR/deploy/playbook_dev"
  [[ -d "$dir" ]] || die "Not found: $dir"
  log_info "Available envs (from deploy/playbook_dev/*.yml):"
  (cd "$dir" && ls -1 *.yml 2>/dev/null | sed -e 's/\.yml$//' || true) | sed 's/^/  - /'
}

list_roles() {
  local dir="$ROOT_DIR/deploy/roles"
  [[ -d "$dir" ]] || die "Not found: $dir"
  log_info "Available roles (from deploy/roles/*):"
  (cd "$dir" && ls -1 2>/dev/null || true) | sed 's/^/  - /'
}

doctor() {
  print_header
  log_info "Checking environment..."
  require_cmd bash
  require_cmd go
  if command -v ansible-playbook >/dev/null 2>&1; then
    log_ok "ansible-playbook: OK"
  else
    log_warn "ansible-playbook: not found (deploy will fail until installed)"
  fi

  require_file "$ROOT_DIR/build.sh"
  require_file "$ROOT_DIR/deploy/deploy.sh"
  log_ok "Files: build.sh / deploy/deploy.sh OK"

  if [[ -d "$ROOT_DIR/excel" ]]; then
    log_ok "excel/: exists"
  else
    log_warn "excel/: not found (xls command will be skipped unless you add excel/run_me.sh)"
  fi

  if [[ -d "$ROOT_DIR/protocol" ]]; then
    log_ok "protocol/: exists"
  else
    log_warn "protocol/: not found (ptc command will be skipped unless you add protocol/gen_code.sh)"
  fi

  log_ok "Doctor finished."
}

run_build() {
  local target="${1:-}"
  require_file "$ROOT_DIR/build.sh"
  print_header
  log_info "Building... (target='${target:-all}')"
  (cd "$ROOT_DIR" && ./build.sh ${target:+$target})
  log_ok "Build done."
}

run_excel() {
  print_header
  if [[ -x "$ROOT_DIR/excel/run_me.sh" ]]; then
    log_info "Running excel importer: excel/run_me.sh"
    (cd "$ROOT_DIR/excel" && ./run_me.sh)
    log_ok "Excel import done."
    return 0
  fi

  # fallback hint
  if [[ -f "$ROOT_DIR/lib/util/xlstrans/buid.sh" ]]; then
    log_warn "excel/run_me.sh not found. Repo has xlstrans builder at lib/util/xlstrans/buid.sh"
    log_warn "You may generate excel tool/output following doc/setup_linux.md and your pipeline."
    return 0
  fi

  die "excel/run_me.sh not found. (xls command unavailable in this repo state)"
}

run_protocol() {
  print_header
  if [[ -x "$ROOT_DIR/protocol/gen_code.sh" ]]; then
    log_info "Running protocol generator: protocol/gen_code.sh"
    (cd "$ROOT_DIR/protocol" && ./gen_code.sh)
    log_ok "Protocol generation done."
    return 0
  fi

  die "protocol/gen_code.sh not found. (ptc command unavailable in this repo state)"
}

run_deploy() {
  local env="${1:-}"
  local action="${2:-}"
  shift 2 || true
  local roles=("$@")

  [[ -n "$env" ]] || die "deploy: missing <env>"
  [[ -n "$action" ]] || die "deploy: missing <init|push|start|stop|restart|all>"

  require_file "$ROOT_DIR/deploy/deploy.sh"

  print_header
  log_info "Deploying: env=${env}, action=${action}, roles='${roles[*]:-all}'"

  if [[ "$action" == "all" ]]; then
    (cd "$ROOT_DIR/deploy" && ./deploy.sh "$env" push "${roles[@]+"${roles[@]}"}")
    sleep 1
    (cd "$ROOT_DIR/deploy" && ./deploy.sh "$env" restart "${roles[@]+"${roles[@]}"}")
    log_ok "Deploy all (push+restart) done."
  else
    (cd "$ROOT_DIR/deploy" && ./deploy.sh "$env" "$action" "${roles[@]+"${roles[@]}"}")
    log_ok "Deploy done."
  fi
}

run_all() {
  local env="${1:-}"
  local action="${2:-all}"
  local role="${3:-}"

  [[ -n "$env" ]] || die "all: missing <env>"
  print_header
  run_build ""
  sleep 1
  if [[ -n "$role" ]]; then
    run_deploy "$env" "$action" "$role"
  else
    run_deploy "$env" "$action"
  fi
  log_ok "All done."
}

cmd="${1:-help}"
shift || true

case "$cmd" in
help|-h|--help)
  usage
  ;;

doctor)
  doctor
  ;;

list)
  sub="${1:-}"
  case "$sub" in
    envs) list_envs ;;
    roles) list_roles ;;
    *) die "Unknown list subcommand: '$sub' (use: list envs|roles)" ;;
  esac
  ;;

build)
  run_build "${1:-}"
  ;;

deploy|dep)
  run_deploy "${1:-}" "${2:-}" "${@:3}"
  ;;

# legacy aliases
all)
  run_all "${1:-}" "${2:-all}" "${3:-}"
  ;;
xls)
  run_excel
  ;;
ptc)
  run_protocol
  ;;

*)
  log_error "Unknown command: $cmd"
  usage
  exit 1
  ;;
esac
