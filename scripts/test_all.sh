#!/usr/bin/env bash
set -euo pipefail

cd "${BASH_SOURCE[0]%/*}/.."

echo "[1/2] Running root module tests..." >&2
go test -count=1 ./...

echo "[2/2] Running examples module tests..." >&2
(
  cd examples
  go test -count=1 -tags json1 ./...
)

echo "All tests passed." >&2
