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

# In WSL, ensure we build Linux binaries (Windows-built plugins won't execute).
GO_BUILD_ENV=()
if grep -qi microsoft /proc/version 2>/dev/null; then
  GO_BUILD_ENV+=("GOOS=linux" "GOARCH=amd64")
fi

env "${GO_BUILD_ENV[@]}" go build -o "$BIN_DIR/protoc-gen-goone" ./tools/protoc-gen-goone
env "${GO_BUILD_ENV[@]}" go build -o "$BIN_DIR/protoc-gen-go" google.golang.org/protobuf/cmd/protoc-gen-go

PROTOC="${PROTOC:-}"
if [[ -z "$PROTOC" ]]; then
  # Prefer vendored protoc for reproducibility and to avoid incompatible wrappers on PATH.
  if [[ -x "lib/contrib/protoc/protoc-33.2-linux-x86_64/bin/protoc" ]]; then
    PROTOC="$ROOT_DIR/lib/contrib/protoc/protoc-33.2-linux-x86_64/bin/protoc"
  elif [[ -x "lib/util/deps/protoc/protoc-33.2-linux-x86_64/bin/protoc" ]]; then
    PROTOC="$ROOT_DIR/lib/util/deps/protoc/protoc-33.2-linux-x86_64/bin/protoc"
  elif command -v protoc >/dev/null 2>&1; then
    PROTOC="$(command -v protoc)"
  else
    echo "[proto_goone] ERROR: protoc not found. Install protoc or run in WSL with vendored protoc available." >&2
    exit 1
  fi
fi

# Detect whether protoc supports modern plugin option flags: --go_opt / --<plugin>_opt.
# Probe is more reliable than --help, but it must use the same plugin wiring.
SUPPORTS_PLUGIN_OPT=1
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t goone_proto)"
trap 'rm -rf "$TMP_DIR"' EXIT
cat >"$TMP_DIR/_goone_probe.proto" <<'EOF'
syntax = "proto3";
package goone.probe;
option go_package = "github.com/Iori372552686/GoOne/api/gen/goone/probe;probe";
message Probe {}
EOF

# Probe: if the *flag* is unsupported, protoc will error before plugins matter.
# We still pass the plugin path to avoid false negatives when PATH lacks protoc-gen-go.
PROBE_ERR="$TMP_DIR/_probe.err"
if ! "$PROTOC" -I "$TMP_DIR" --descriptor_set_out="$TMP_DIR/_probe.pb" \
  --plugin=protoc-gen-go="$BIN_DIR/protoc-gen-go" \
  --go_out="$TMP_DIR" --go_opt=module="$MODULE" --go_opt=paths=import "$TMP_DIR/_goone_probe.proto" \
  >/dev/null 2>"$PROBE_ERR"; then
  if grep -q -- "Unknown flag: --go_opt" "$PROBE_ERR"; then
    SUPPORTS_PLUGIN_OPT=0
  else
    echo "[proto_goone] probe failed, but not due to flag support. stderr:" >&2
    tail -n 50 "$PROBE_ERR" >&2 || true
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

if [[ "$SUPPORTS_PLUGIN_OPT" -eq 1 ]]; then
  ARGS+=("--go_out=$OUT_DIR" "--go_opt=module=$MODULE" "--go_opt=paths=import")
  ARGS+=("--goone_out=$OUT_DIR" "--goone_opt=module=$MODULE" "--goone_opt=paths=import")
else
  echo "[proto_goone] WARN: protoc does not support --*_opt flags; falling back to legacy ':outdir' parameter style"
  ARGS+=("--go_out=module=$MODULE,paths=import:$OUT_DIR")
  ARGS+=("--goone_out=module=$MODULE,paths=import:$OUT_DIR")
fi

"$PROTOC" "${ARGS[@]}" "${PROTOS[@]}"

echo "[proto_goone] done"

