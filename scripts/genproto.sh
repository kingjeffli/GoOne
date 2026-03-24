#!/usr/bin/env bash
set -euo pipefail

# WSL/Linux friendly wrapper.
# Generates:
# - api/gen/**.pb.go (protoc-gen-go)
# - api/gen/**.goone_ssrpc.gen.go (protoc-gen-goone)
#
# Usage:
#   ./scripts/genproto.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

go run ./tools/cmd/genproto


