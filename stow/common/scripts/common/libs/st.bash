#!/bin/bash

BOLD=
OFFBOLD=
RESET_COLOR=
RED=
GREEN=
YELLOW=
BLUE=
BLUE_CYAN=
GRAY_LIGHT=

if [ -t 1 ]; then
    BOLD=$(tput bold)
    OFFBOLD=$(tput sgr0)
    RESET_COLOR="$(tput sgr0)"
    RED="$(
        tput bold
        tput setaf 1
    )"
    GREEN="$(
        tput bold
        tput setaf 2
    )"
    YELLOW="$(
        tput bold
        tput setaf 3
    )"
    BLUE="$(
        tput bold
        tput setaf 4
    )"
    BLUE_CYAN="$(
        tput bold
        tput setaf 6
    )"
    GRAY_LIGHT="$(tput setaf 250)"
fi

DOING_MSG=

function st.cmd.exists() {
    command -v "$1" >/dev/null 2>&1
}

## Usage: st.var.exists A_VAR && echo PASS
function st.var.exists() {
    [ -n "${!1:-}" ]
}

function st.h1() {
    echo -e "st.h1> ${BOLD}$1${OFFBOLD}"
}

function st.h2() {
    echo -e "st.h2> ${BOLD}$1${OFFBOLD}"
}

function st.h3() {
    echo -e "st.h3> ${BOLD}$1${OFFBOLD}"
}

function st.doing() {
    DOING_MSG=$1

    echo "st.doing> ${BLUE}${DOING_MSG:-…}$RESET_COLOR"
}

function st.done() {
    local DONE="${1:-[DONE]}"

    echo "st.done> ${DOING_MSG:-} : ${GREEN}$DONE${RESET_COLOR}"
}

function st.success() {
    local MSG="${1:-[SUCCESS]}"

    echo "st.success> ${BOLD}${GREEN}${MSG}${RESET_COLOR}${OFFBOLD}"
}

function st.nothing() {
    local MSG="${1:-[NOTHING TO DO]}"
    echo "st.nothingtd> ${DOING_MSG:-} : ${GREEN}${MSG}${RESET_COLOR}"
}

function st.skipped() {
    local MSG="${1:-[SKIPPED]}"
    echo "st.skipped> ${DOING_MSG:-} : ${BLUE_CYAN}${MSG}${RESET_COLOR}"
}

function st.warn() {
    echo "st.warn> ${BOLD}${YELLOW}$1${RESET_COLOR}${OFFBOLD}"
}

function st.fail() {
    local MSG="${1:-[FAILED]}"
    echo -e "st.fail ${DOING_MSG:-} : ${RED}$MSG${RESET_COLOR}"
    false
}

function st.abort() {
    local MSG="${1:-[ABORTED]}"
    echo -e "st.abort> ${DOING_MSG:-} : ${BOLD}${RED}${MSG}${RESET_COLOR}${OFFBOLD}\n"
    false

    exit 1
}

function st.do() {
    local -a cmd=("$@")
    echo "st.do> ${GRAY_LIGHT}${cmd[*]}${RESET_COLOR}"
    "${cmd[@]}"
}
