#!/bin/bash
# shellcheck disable=SC2119,SC2120
# Git pre-commit hook - runs via mise

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace
(shopt -p inherit_errexit &>/dev/null) && shopt -s inherit_errexit

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
source "$SCRIPT_DIR/init.bash" || exit 1

l.trap_error

st.h1 "Running pre-commit checks..."

[ -z "${APP_ROOT_PATH:-}" ] && st.fail 'Environment variable APP_ROOT_PATH not set'

cd "$APP_ROOT_PATH" || l.fail

# Get list of staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

if [ -z "$STAGED_FILES" ]; then
    echo "No staged files to check"
    st.nothing
    exit 0
fi

st.h2 "Fix trailing whitespace and ensure newline at end of file"
echo "$STAGED_FILES" | while read -r file; do
    if [ -f "$file" ]; then
        st.doing "Removing trailing whitespace on $file"
        st.do sed -i 's/[[:space:]]*$//' "$file"
        st.done
        st.doing "Ensure file ends with newline"
        st.do sed -i -e "\$a\\" "$file"
        st.done
        st.doing 'Re-add file that was fixed'
        st.do git add "$file"
        st.done
    fi
done

st.h2 "Check YAML syntax"
st.doing "Checking YAML files..."
PASS=false
YAML_FILES=$(echo "$STAGED_FILES" | grep -E '\.ya?ml$' || true)
if [ -n "$YAML_FILES" ]; then
    st.do echo "$YAML_FILES" | xargs -r yamllint -d relaxed 2>/dev/null || true
    PASS=true
fi

if $PASS; then
    st.done
else
    st.nothing
fi

DOING_MSG="Check for large files (>500KB)"
st.h2 "$DOING_MSG"
echo "$STAGED_FILES" | while read -r file; do
    if [ -f "$file" ]; then
        size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
        if [ "$size" -gt 512000 ]; then
            st.fail "ERROR: File $file is larger than 500KB ($size bytes)"
        fi
    fi
done
st.done

st.h2 "Go linting if any Go files changed"
if echo "$STAGED_FILES" | grep -q '\.go$'; then
    st.doing "Running golangci-lint..."
    st.do golangci-lint config verify
    st.do golangci-lint run --fix
    st.done
    st.doing 'Re-add any files that were fixed'
    st.do echo "$STAGED_FILES" | grep '\.go$' | xargs -r git add
    st.done
else
    st.nothing
fi

st.h2 "Buf lint/generate for proto changes"
if echo "$STAGED_FILES" | grep -q '\.proto$'; then
    echo "Running buf lint and generate..."
    cd internal/user/api && buf lint && buf generate
    cd - >/dev/null
    # Add generated files
    git add internal/user/api/gen/ 2>/dev/null || true
else
    st.nothing
fi

st.h2 "Pre-Commit Checks"
st.success "PRE-COMMIT CHECKS PASSED 🚀"
