#!/usr/bin/env bash
set -euo pipefail

cd "${BASH_SOURCE[0]%/*}/.."

export MYSQL_DSN='gorm:gorm@tcp(127.0.0.1:9910)/gorm?parseTime=true&charset=utf8mb4&loc=Local'

echo "Running root module tests..." >&2
go test -count=1 ./...

echo "Running examples module tests..." >&2
(
  cd examples
  go test -count=1 -tags json1 ./...
)

echo "Done." >&2
