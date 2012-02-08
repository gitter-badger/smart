#!/bin/bash
set -e

[[ -f ../smart.go && -f ../main.go && -f ../build.sh ]] || {
    echo $BASH_SCRIPT:$LINENO "not in test subdir"
    exit -1
}

needs_build() {
    [[ -f ../smart ]] || return 1
    T=`stat -c %Y ../smart`
    for g in ../*.go; do
        [[ "$T" -lt "`stat -c %Y $g`" ]] && return 1
    done
    return 0
}

enter() {
    local D=$1
    cd $D && echo "smart: Entering directory \`$D'"
}

leave() {
    local D=$1
    cd - > /dev/null && echo "smart: Entering directory \`$D'"
}

needs_build || {
    enter ..
    ./build.sh
    leave ..
}

smart="../smart"

#$smart
#$smart -C gcc -T .
$smart -a
