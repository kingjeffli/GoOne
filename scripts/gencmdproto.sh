#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Examples:
#   ./scripts/gencmdproto.sh
#   ./scripts/gencmdproto.sh -prefix CMD_MAIN_ -prefix CMD_ROOM_CENTER_

go run ./tools/cmd/gencmdproto "$@"


