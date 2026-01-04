#!/usr/bin/env bash
set -euo pipefail

# One-click proto generation (pb.go + goone ssrpc wrappers).
#
# Generates:
# - api/gen/**/*.pb.go               (protoc-gen-go)
# - api/gen/**/*.goone_ssrpc.gen.go  (protoc-gen-goone)
#
# Defaults:
# - MODULE: github.com/Iori372552686/GoOne
# - OUT: repo root (.)
# - PROTO_ROOT: api/proto
#
# Usage:
#   ./scripts/proto_goone.sh
#   MODULE=xxx ./scripts/proto_goone.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

MODULE="${MODULE:-github.com/Iori372552686/GoOne}"
OUT_DIR="${OUT_DIR:-.}"
PROTO_ROOT="${PROTO_ROOT:-api/proto}"
BIN_DIR="${BIN_DIR:-.bin}"

mkdir -p "$BIN_DIR"

echo "[proto_goone] build plugins -> $BIN_DIR"
go build -o "$BIN_DIR/protoc-gen-goone" ./tools/protoc-gen-goone
go build -o "$BIN_DIR/protoc-gen-go" google.golang.org/protobuf/cmd/protoc-gen-go

PROTOC="${PROTOC:-}"
if [[ -z "$PROTOC" ]]; then
  if command -v protoc >/dev/null 2>&1; then
    PROTOC="$(command -v protoc)"
  elif [[ -x "lib/contrib/protoc/protoc-33.2-linux-x86_64/bin/protoc" ]]; then
    PROTOC="$ROOT_DIR/lib/contrib/protoc/protoc-33.2-linux-x86_64/bin/protoc"
  elif [[ -x "lib/util/deps/protoc/protoc-33.2-linux-x86_64/bin/protoc" ]]; then
    PROTOC="$ROOT_DIR/lib/util/deps/protoc/protoc-33.2-linux-x86_64/bin/protoc"
  else
    echo "[proto_goone] ERROR: protoc not found. Install protoc or run in WSL with vendored protoc available." >&2
    exit 1
  fi
fi

# Include dirs for google well-known types (for empty.proto / descriptor.proto etc.)
INCLUDE_DIRS=()
if [[ -d "lib/contrib/protoc/protoc-33.2-linux-x86_64/include" ]]; then
  INCLUDE_DIRS+=("lib/contrib/protoc/protoc-33.2-linux-x86_64/include")
fi
if [[ -d "lib/util/deps/protoc/protoc-33.2-linux-x86_64/include" ]]; then
  INCLUDE_DIRS+=("lib/util/deps/protoc/protoc-33.2-linux-x86_64/include")
fi

echo "[proto_goone] collect proto inputs (exclude api/proto/third_party)"
mapfile -t PROTOS < <(
  find "$PROTO_ROOT/goone" "$PROTO_ROOT/game" -type f -name '*.proto' -print \
    | sed "s|^$PROTO_ROOT/||" \
    | LC_ALL=C sort
)

echo "[proto_goone] protoc=$PROTOC"
echo "[proto_goone] module=$MODULE"
echo "[proto_goone] inputs=${#PROTOS[@]}"

ARGS=()
ARGS+=("-I" "$PROTO_ROOT")
for inc in "${INCLUDE_DIRS[@]}"; do
  ARGS+=("-I" "$inc")
done

ARGS+=("--plugin=protoc-gen-go=$BIN_DIR/protoc-gen-go")
ARGS+=("--plugin=protoc-gen-goone=$BIN_DIR/protoc-gen-goone")
ARGS+=("--go_out=$OUT_DIR" "--go_opt=module=$MODULE" "--go_opt=paths=import")
ARGS+=("--goone_out=$OUT_DIR" "--goone_opt=module=$MODULE" "--goone_opt=paths=import")

"$PROTOC" "${ARGS[@]}" "${PROTOS[@]}"

echo "[proto_goone] done"


