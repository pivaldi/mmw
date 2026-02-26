#!/usr/bin/env bash

set -euo pipefail

echo "Installing development tools..."

# unclutter your .profile
go install github.com/direnv/direnv@latest

# LSP
go install golang.org/x/tools/gopls@latest

# golangci-lint LSP wrapper
go install github.com/nametake/golangci-lint-langserver@latest

# Formatting
go install golang.org/x/tools/cmd/goimports@latest

# Debugger
go install github.com/go-delve/delve/cmd/dlv@latest

# Static analysis
go install honnef.co/go/tools/cmd/staticcheck@latest

# Protobuf / gRPC generators
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Shell formatting
go install mvdan.cc/sh/v3/cmd/shfmt@latest

# Find symbol information in Go source
go install github.com/rogpeppe/godef@latest

# Test runner (it is in the go tool)
# go install gotest.tools/gotestsum@latest

# Go enum generator (It's in the go tool)
# go install github.com/abice/go-enum

if ! command -v goda >/dev/null 2>&1; then
    go install github.com/loov/goda@latest
fi

if ! command -v dep-tree >/dev/null 2>&1; then
    echo "Installing dep-tree..."
    if command -v brew >/dev/null 2>&1; then
        echo "Using brew to install dep-tree..."
        brew install dep-tree
    elif command -v pip >/dev/null 2>&1; then
        echo "Using pip to install dep-tree..."
        pip install dep-tree
    elif command -v npm >/dev/null 2>&1; then
        echo "Using npm to install dep-tree..."
        npm install -g dep-tree
    else
        echo "Warning: Could not install dep-tree - no package manager found (brew, pip, or npm)"
    fi
fi

echo "Development tools installed successfully!"
