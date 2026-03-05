#!/usr/bin/env bash
#!/usr/bin/env bash
# shellcheck disable=SC2119,SC2120
###############################################################################
## this script is "materializing" or "de-stowing" the stow shared components. #
###############################################################################

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace
(shopt -p inherit_errexit &>/dev/null) && shopt -s inherit_errexit

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
readonly SCRIPT_DIR

. "${SCRIPT_DIR}/../stow/common/scripts/common/init.bash"
l.trap_error
st.quiet

# Target directory (defaults to current directory)
APP_ROOT_PATH=${1:-"."}
# Ensure the directory does not end by a slash
APP_ROOT_PATH=$(l.str_replace_last "$APP_ROOT_PATH" "/" "")
cd "$APP_ROOT_PATH" || exit 1
# The absolute path to your central stow storage
STOW_ABS_PATH=$(readlink -f "stow")

SERVICES=()
for dir in services/*/; do
    SERVICES+=("$dir")
done

st.h1 "Materializing Stow-managed links"

echo "${BLUE}Select the service to materialize${RESET_COLOR}"
SERVICE_DIR=$(l.choose "${SERVICES[@]}")
SERVICE_DIR_FULL_PATH=$(l.str_replace_last "$APP_ROOT_PATH/$SERVICE_DIR" "/" "")

LOG_FILE="${SERVICE_DIR_FULL_PATH}/un-stow.log"
function h.log {
    echo "$(date -Iseconds)> $1" >>"$LOG_FILE"
}

SHORT_LOG_FILE=$(l.str_replace "$LOG_FILE" "$APP_ROOT_PATH" "")
st.h2 "Materializing Stow-managed links in ${SERVICE_DIR}"
echo "${RED}You are about to remove all **Stow-managed** symlinks of the service '$SERVICE_DIR'.
This will replace them with their actual/materialized contents.
To go back, you will have to carefully delete these contents and rerun the 'mise run stow' command.
All actions will be logged in the file .${SHORT_LOG_FILE}.${RESET_COLOR}"

CONTINUE=$(l.ask "Are you sure you want to continue?")
[ "$CONTINUE" == "NO" ] && exit 0

find "$SERVICE_DIR_FULL_PATH" -type l | while read -r link; do
    # Get the real path of the link
    REAL_PATH=$(readlink -f "$link")
    shortl=".$(l.str_replace "$link" "$APP_ROOT_PATH" "")"
    shortr=".$(l.str_replace "$REAL_PATH" "$APP_ROOT_PATH" "")"

    # CHECK: Does the real path start with our stow-central path?
    if [[ "$REAL_PATH" == "$STOW_ABS_PATH"* ]]; then
        st.h1 "Valid Stow link found: $shortl -> $shortr"
        st.doing "Removing $shortl"
        st.do rm "$link"
        h.log "remove $link"
        st.done

        st.doing "Replace with physical copy"
        if [ -d "$REAL_PATH" ]; then
            st.do cp -r "$REAL_PATH" "$link"
        else
            st.do cp "$REAL_PATH" "$link"
        fi

        h.log "copy $REAL_PATH to $link"

        st.done
    else
        st.skipped "non-stow link: $shortl (points to $shortr)"
        h.log "skipped non-stow link: $link"
    fi
done

st.success "Materialize Stow-managed Links Succeed"

echo "=> All actions was logged in $LOG_FILE"
