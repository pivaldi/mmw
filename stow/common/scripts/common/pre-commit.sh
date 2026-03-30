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

st.quiet

st.h1 "Running pre-commit checks..."

[ -z "${APP_ROOT_PATH:-}" ] && st.fail 'Environment variable APP_ROOT_PATH not set'

cd "$APP_ROOT_PATH" || l.fail

# Get list of staged files using null-terminated strings
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM -z)

if [ -z "$STAGED_FILES" ]; then
    echo "No staged files to check"
    st.nothing
    exit 0
fi

st.h2 "Fix trailing whitespace and ensure newline at end of file"
printf "%s" "$STAGED_FILES" | while IFS= read -r -d '' file; do
    if [ -f "$file" ]; then
        st.doing "Removing trailing whitespace on $file"
        st.do sed -i -e 's/[[:space:]]*$//' "$file"
        st.done

        st.doing "Ensure file ends with newline"
        # Only add a newline if the last character is NOT already a newline
        if [ -n "$(tail -c 1 "$file")" ]; then
            echo "" >>"$file"
        fi
        st.done

        st.doing 'Re-add file that was fixed'
        st.do git add "$file"
        st.done
    fi
done

st.h2 "Check YAML syntax"
st.doing "Checking YAML files..."
PASS=false
# Use grep -z to filter null-terminated strings
YAML_FILES=$(printf "%s" "$STAGED_FILES" | grep -z -E '\.ya?ml$' || true)

if [ -n "$YAML_FILES" ]; then
    # Use xargs -0 to read the null-terminated strings
    st.do printf "%s" "$YAML_FILES" | xargs -0 -r yamllint -d relaxed 2>/dev/null || true
    PASS=true
fi

if $PASS; then
    st.done
else
    st.nothing
fi

DOING_MSG="Check for large files (>500KB)"
st.h2 "$DOING_MSG"
printf "%s" "$STAGED_FILES" | while IFS= read -r -d '' file; do
    if [ -f "$file" ]; then
        size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
        if [ "$size" -gt 512000 ]; then
            st.fail "ERROR: File $file is larger than 500KB ($size bytes)"
        fi
    fi
done
st.done

st.h2 "Go linting/formatting (staged files only)"
STAGED_GO_FILES=$(printf "%s" "$STAGED_FILES" | grep -z '\.go$' || true)

if [ -n "$STAGED_GO_FILES" ]; then
    st.doing "Running golangci-lint..."
    st.do golangci-lint config verify

    printf "%s" "$STAGED_GO_FILES" | while IFS= read -r -d '' file; do
        [ -f "$file" ] || continue
        st.do golangci-lint run --fix "$file"
        st.do gofumpt -l -w "$file"
        st.do git add "$file"
    done
    st.done
else
    st.nothing
fi

st.h2 "Pre-Commit Checks"
st.success "PRE-COMMIT CHECKS PASSED 🚀"
