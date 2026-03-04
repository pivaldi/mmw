#!/usr/bin/env bash
# shellcheck disable=SC2119,SC2120

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace
(shopt -p inherit_errexit &>/dev/null) && shopt -s inherit_errexit

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
source "$SCRIPT_DIR/init.bash" || exit 1

l.trap_error

UPDATE=${UPDATE:-false}

st.doing "Installing direnv"
if $UPDATE || ! command -v direnv >/dev/null 2>&1; then
    st.do go install github.com/direnv/direnv/v2@latest
    st.done
else
    st.nothing
fi

# st.doing "Installing migrate..."
# st.do go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

st.done "Tools installed successfully!"

[ -z "$APP_ENV" ] && {
    st.fail 'APP_ENV not set.'
}

if [ "$APP_ENV" = "development" ]; then
    st.h1 "Installing development tools..."

    st.doing "Installing buf..."
    if $UPDATE || ! command -v buf >/dev/null 2>&1; then
        st.do go install github.com/bufbuild/buf/cmd/buf@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing protoc-gen-go..."
    if $UPDATE || ! command -v protoc-gen-go >/dev/null 2>&1; then
        st.do go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing protoc-gen-connect-go..."
    if $UPDATE || ! command -v protoc-gen-connect-go >/dev/null 2>&1; then
        st.do go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing Goda"
    if $UPDATE || ! command -v goda >/dev/null 2>&1; then
        st.do go install github.com/loov/goda@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing arch-go"
    if $UPDATE || ! command -v arch-go >/dev/null 2>&1; then
        st.do go install -v github.com/arch-go/arch-go/v2@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing gopls (LSP)"
    if $UPDATE || ! command -v gopls >/dev/null 2>&1; then
        st.do go install golang.org/x/tools/gopls@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing golangci-lint LSP wrapper"
    if $UPDATE || ! command -v golangci-lint-langserver >/dev/null 2>&1; then
        st.do go install github.com/nametake/golangci-lint-langserver@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing  Formatting"
    if $UPDATE || ! command -v goimports >/dev/null 2>&1; then
        st.do go install golang.org/x/tools/cmd/goimports@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing  Debugger"
    if $UPDATE || ! command -v dlv >/dev/null 2>&1; then
        st.do go install github.com/go-delve/delve/cmd/dlv@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing Static Analysis"
    if $UPDATE || ! command -v staticcheck >/dev/null 2>&1; then
        st.do go install honnef.co/go/tools/cmd/staticcheck@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing gRPC generators"
    if $UPDATE || ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
        st.do go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing Shell Formatting"
    if $UPDATE || ! command -v shfmt >/dev/null 2>&1; then
        st.do go install mvdan.cc/sh/v3/cmd/shfmt@latest
        st.done
    else
        st.nothing
    fi

    st.doing "Installing godef"
    if $UPDATE || ! command -v godef >/dev/null 2>&1; then
        st.do go install github.com/rogpeppe/godef@latest
        st.done
    else
        st.nothing
    fi

    # Test runner (it is in the go tool)
    # go install gotest.tools/gotestsum@latest

    # Go enum generator (It's in the go tool)
    # st.do go install github.com/abice/go-enum

    st.h1 "Installing dep-tree..."
    if $UPDATE || ! command -v dep-tree >/dev/null 2>&1; then
        if command -v brew >/dev/null 2>&1; then
            st.doing "Using brew to install dep-tree..."
            $UPDATE && HOMEBREW_NO_ENV_HINTS=1 st.do brew reinstall dep-tree
            $UPDATE || HOMEBREW_NO_ENV_HINTS=1 st.do brew install dep-tree
            st.done
        elif command -v pip >/dev/null 2>&1; then
            st.doing "Using pip to install dep-tree..."
            st.do pip install dep-tree
            st.done
        elif command -v npm >/dev/null 2>&1; then
            st.doing "Using npm to install dep-tree..."
            st.do npm install -g dep-tree
            st.done
        else
            st.warn "Warning: Could not install dep-tree - no package manager found (brew, pip, or npm)"
        fi
    else
        st.nothing
    fi

    st.done "Development tools installed successfully!"
fi
