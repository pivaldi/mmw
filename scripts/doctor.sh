#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC2034,1091
. "$SCRIPT_DIR/color.rc"

echo "${yellow}== mise ==${resetColor}"
mise --version || true

echo
echo "${yellow}== go ==${resetColor}"
go version
echo "GOBIN=$(go env GOBIN)"
echo "GOPATH=$(go env GOPATH)"

echo
echo "${yellow}== gopls ==${resetColor}"
command -v gopls >/dev/null && gopls version || echo "gopls: not found (run: mise run tools)"

echo
echo "${yellow}== golangci-lint-langserver ==${resetColor}"
command -v golangci-lint-langserver || echo "golangci-lint-langserver not found (run: mise run tools)"

echo
echo "${yellow}== golangci-lint ==${resetColor}"
command -v golangci-lint >/dev/null && golangci-lint --version || echo "golangci-lint: not found (run: mise install)"

echo
echo "${yellow}== buf ==${resetColor}"
command -v buf >/dev/null && buf --version || echo "buf: not found (run: mise install)"

echo
echo "${yellow}== protoc ==${resetColor}"
command -v protoc >/dev/null && protoc --version || echo "protoc: not found (apt install protobuf-compiler) or use devbox"

echo
echo "${yellow}== protoc-gen-go ==${resetColor}"
command -v protoc-gen-go >/dev/null && protoc-gen-go --version || echo "protoc-gen-go: not found (run: mise run tools)"

echo
echo "${yellow}== protoc-gen-go-grpc ==${resetColor}"
command -v protoc-gen-go-grpc >/dev/null && protoc-gen-go-grpc --version || echo "protoc-gen-go-grpc: not found (run: mise run tools)"
