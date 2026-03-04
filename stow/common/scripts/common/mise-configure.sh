#!/usr/bin/env bash

# See https://gist.github.com/pivaldi/8c23c383d86469fae9077f82f8aced21

set -euo pipefail

# Détecte le shell courant de façon plus fiable que $SHELL
detect_shell() {
    # $0 = nom du shell (bash, -bash, zsh, -zsh, etc.)
    local sh="${0#-}"
    sh="${sh##*/}"
    case "$sh" in
    bash | zsh) echo "$sh" ;;
    *) return 1 ;;
    esac
}

SHELL_NAME="$(detect_shell || true)"
[ -n "${SHELL_NAME:-}" ] || exit 0

# 1) Fichier d'init central
INIT_DIR="$HOME/.config/mise"
INIT_FILE="$INIT_DIR/shell-init.sh"
mkdir -p "$INIT_DIR"

cat >"$INIT_FILE" <<'EOF'
# mise shell init (generated)
# - fast: no network work on shell startup
# - safe: only runs if mise exists
# - idempotent completions

if ! command -v mise >/dev/null 2>&1; then
  return 0 2>/dev/null || exit 0
fi

# Activate mise
# (shell-specific: bash or zsh)
case "${0#-}" in
  *bash) eval "$(mise activate bash)" ;;
  *zsh)  eval "$(mise activate zsh)"  ;;
esac

# Completions (generate once)
gen_completion() {
  local shell="$1"
  if [ "$shell" = "bash" ]; then
    local comp_dir="$HOME/.local/share/bash-completion/completions"
    mkdir -p "$comp_dir"
    local target="$comp_dir/mise"

    # (Re)generate if missing
    if [ ! -s "$target" ]; then
      mise completion bash --include-bash-completion-lib >"$target"
    fi
  else
    local comp_dir="$HOME/.local/share/zsh/completions"
    mkdir -p "$comp_dir"
    local target="$comp_dir/_mise"

    if [ ! -s "$target" ]; then
      mise completion zsh >"$target"
    fi

    # Ensure comp_dir is in fpath
    if [[ ":$FPATH:" != *":$comp_dir:"* ]]; then
      fpath=("$comp_dir" $fpath)
    fi
  fi
}

case "${0#-}" in
  *bash) gen_completion bash ;;
  *zsh)  gen_completion zsh  ;;
esac
EOF

chmod 0644 "$INIT_FILE"

# 2) Ajouter un “source” unique dans le bon rc
append_once() {
    local line="$1" file="$2"
    mkdir -p "$(dirname "$file")"
    touch "$file"
    grep -Fqx "$line" "$file" || printf "\n%s\n" "$line" >>"$file"
}

if [ "$SHELL_NAME" = "bash" ]; then
    append_once '[ -f "$HOME/.config/mise/shell-init.sh" ] && . "$HOME/.config/mise/shell-init.sh"' "$HOME/.bashrc"
elif [ "$SHELL_NAME" = "zsh" ]; then
    append_once '[ -f "$HOME/.config/mise/shell-init.sh" ] && . "$HOME/.config/mise/shell-init.sh"' "$HOME/.zshrc"
fi
