#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Examples:
#   ./scripts/dumpg1cmd.sh CMD_ROOM_CENTER_INNER_
#   ./scripts/dumpg1cmd.sh --exact CMD_MAIN_LOGIN_REQ

if [[ "${1:-}" == "--exact" ]]; then
  shift
  go run ./tools/cmd/dumpg1cmd -exact "${1:?missing CMD name}"
  exit 0
fi

go run ./tools/cmd/dumpg1cmd -prefix "${1:-CMD_}"


