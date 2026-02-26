#!/bin/bash
# Git pre-commit hook - runs via mise
set -e

echo "Running pre-commit checks..."

# Get list of staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

if [ -z "$STAGED_FILES" ]; then
  echo "No staged files to check"
  exit 0
fi

# Fix trailing whitespace and ensure newline at end of file
echo "$STAGED_FILES" | while read -r file; do
  if [ -f "$file" ]; then
    # Remove trailing whitespace
    sed -i 's/[[:space:]]*$//' "$file"
    # Ensure file ends with newline
    [ -n "$(tail -c1 "$file")" ] && echo >>"$file"
    git add "$file"
  fi
done

# Check YAML syntax
YAML_FILES=$(echo "$STAGED_FILES" | grep -E '\.ya?ml$' || true)
if [ -n "$YAML_FILES" ]; then
  echo "Checking YAML files..."
  echo "$YAML_FILES" | xargs -r yamllint -d relaxed 2>/dev/null || true
fi

# Check for large files (>500KB)
echo "$STAGED_FILES" | while read -r file; do
  if [ -f "$file" ]; then
    size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
    if [ "$size" -gt 512000 ]; then
      echo "ERROR: File $file is larger than 500KB ($size bytes)"
      exit 1
    fi
  fi
done

# Go linting (if any Go files changed)
if echo "$STAGED_FILES" | grep -q '\.go$'; then
  echo "Running golangci-lint..."
  golangci-lint config verify
  golangci-lint run --fix
  # Re-add any files that were fixed
  echo "$STAGED_FILES" | grep '\.go$' | xargs -r git add
fi

# Buf lint/generate for proto changes
if echo "$STAGED_FILES" | grep -q '\.proto$'; then
  echo "Running buf lint and generate..."
  cd internal/user/api && buf lint && buf generate
  cd - >/dev/null
  # Add generated files
  git add internal/user/api/gen/ 2>/dev/null || true
fi

echo "Pre-commit checks passed!"
