#!/usr/bin/env bash

ITS_SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

SCRIPT="$ITS_SCRIPT_DIR/install-tools.sh"
STEPPS=$(grep -c 'st.done' "$SCRIPT")

[ -z "${TOOLS_PASS:-}" ] || exit 0

UPDATE=${UPDATE:-false} stream-stepper --wait=1 --processor=stbash --steps="$STEPPS" "$SCRIPT"
