#!/bin/bash
set -e

out="./out"
smart="../smart"

[[ -f ../smart.go && -f ../main.go && -f ../build.sh ]] || {
    echo $BASH_SCRIPT:$LINENO "not in test subdir"
    exit -1
}

needs_build() {
    [[ -f $smart ]] || return 1
    T=`stat -c %Y $smart`
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
    cd - > /dev/null && echo "smart: Leaving directory \`$D'"
}

needs_build || {
    enter ..
    ./build.sh
    leave ..
}

check() {
    local D=$1
    for f in $D/* ; do
        case $f in
            *check.sh)
                . $f #$(. $f)
                ;;
            *)
                [[ -d $f ]] && {
                    check $f || continue
                }
                ;;
        esac
    done
}

checkdir() {
    local L=$1
    local D=$2
    [[ -d $D ]] || {
        echo "$L: $D not found"
        #exit -1
    }
}

checkfile() {
    local L=$1
    local F=$2
    [[ -f $F ]] || {
        echo "$L: $F not found"
        #exit -1
    }
}

#$smart -a toolset=gcc && ./a.out
$smart toolset=gcc && ./foo
#cd android-sdk && ../$smart

check .
