#!/usr/bin/env bash

INIT_SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
# readonly INIT_SCRIPT_DIR

source "$INIT_SCRIPT_DIR/libs/lobash.bash" || exit 1
source "$INIT_SCRIPT_DIR/libs/st.bash" || exit 1

IN_DOCKER=false

if grep -q docker /proc/1/cgroup; then
    IN_DOCKER=true
fi

function isConfigAppEnvNeeded() {
    [ ! -e "${APP_ROOT_PATH}/.envrc" ] || grep -q 'CONFIG_APP_ENV' "${APP_ROOT_PATH}/.envrc"
}

function configAppEnv() {
    local env_name
    local env_choice

    # shellcheck disable=SC2015
    isConfigAppEnvNeeded && {
        echo -e "\t${yellow}1${resetColor} : development\n"
        echo -e "\t${yellow}2${resetColor} : staging\n"
        echo -e "\t${yellow}3${resetColor} : production\n"
        echo -e "Enter your choice: \c"
        # shellcheck disable=SC2162
        read env_choice

        while [ -z "$env_choice" ] || [[ "$env_choice" != "1" && "$env_choice" != "2" && "$env_choice" != "3" ]]; do
            tput el
            echo -e "Unsupported choice, enter your choice : \c"
            # shellcheck disable=SC2162
            read env_choice

        done

        case $env_choice in
        1)
            env_name="development"
            ;;
        2)
            env_name="staging"
            ;;
        3)
            env_name="production"
            ;;
        esac

        sed -i "s/CONFIG_APP_ENV/${env_name}/g" "${APP_ROOT_PATH}/.envrc"
        cp "${APP_ROOT_PATH}/config_${env_name}.toml.example" "${APP_ROOT_PATH}/config_${env_name}.toml"

        _done
    } || _doneNTD

    [ -z "$env_name" ] || export APP_ENV=$env_name
}
