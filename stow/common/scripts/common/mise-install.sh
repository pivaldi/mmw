#!/usr/bin/env bash
# shellcheck disable=SC2119,SC2120

set -o errexit
set -o nounset
set -o pipefail
set -o errtrace
(shopt -p inherit_errexit &>/dev/null) && shopt -s inherit_errexit

MI_SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
readonly MI_SCRIPT_DIR
. "$MI_SCRIPT_DIR/init.bash"

l.trap_error

[ -z "${APP_ENV:-}" ] && {
    st.abort "APP_ENV not set"
}

st.doing 'Installing Mise-en-place'
if ! type mise &>/dev/null; then
    if type apt &>/dev/null; then
        sudo install -dm 755 /etc/apt/keyrings &&
            curl -fsSL https://mise.jdx.dev/gpg-key.pub | sudo tee /etc/apt/keyrings/mise-archive-keyring.asc >/dev/null &&
            echo "deb [signed-by=/etc/apt/keyrings/mise-archive-keyring.asc] https://mise.jdx.dev/deb stable main" | sudo tee /etc/apt/sources.list.d/mise.list >/dev/null &&
            sudo apt-get update && sudo apt-get install -y mise
    else
        curl https://mise.run | sh
    fi

    st.done
else
    st.nothing
fi

mise settings experimental=true

st.doing "Activating mise in current shell"
eval "$(mise activate bash)"
st.done

st.doing "Configuring mise in standart shell"
st.do "$MI_SCRIPT_DIR/mise-configure.sh"
st.done

st.doing 'Insalling mise usage'
st.do mise use -g usage
st.done
