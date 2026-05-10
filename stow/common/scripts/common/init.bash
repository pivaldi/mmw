#!/usr/bin/env bash

INIT_SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
# readonly INIT_SCRIPT_DIR

source "$INIT_SCRIPT_DIR/libs/lobash.bash" || exit 1
source "$INIT_SCRIPT_DIR/libs/st.bash" || exit 1

IN_DOCKER=false

if grep -q docker /proc/1/cgroup; then
    IN_DOCKER=true
fi

configureEnv() {
    st.h1 'Configuration of APP_ENV'
    st.doing 'Configuration of APP_ENV'
    if [ -e "${APP_ROOT_PATH}/.envrc" ]; then
        st.nothing
    else
        cp "${APP_ROOT_PATH}/.envrc.example" "${APP_ROOT_PATH}/.envrc"
        st.done
    fi

    . "$APP_ROOT_PATH/.envrc" || exit 1
}
