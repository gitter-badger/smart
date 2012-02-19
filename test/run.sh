#!/bin/bash
set -e

out="./out"
smart="$(dir $PWD)/smart"

[[ -f ../smart.go && -f ../main.go && -f ../build.sh ]] || {
    echo $BASH_SOURCE:$LINENO "not in test subdir"
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
    cd $1 && echo "smart: Entering directory \`$1'"
}

leave() {
    cd - > /dev/null && echo "smart: Leaving directory \`$1'"
}

needs_build || {
    enter ..
    #./build.sh
    leave ..
}

run() {
    local D=$1
    local e=""
    local f
    for f in $D/* $D/.* ; do
        case $f in
            */.|*/..|*/out|*/src|*/res|./run.sh)
                #echo "ignore: $f"
                ;;
            */.smart)
                #echo $*/.smart
                ;;
            */run.sh)
                enter $*
                (. run.sh) || e="$BASH_SOURCE:$LINENO: failed '$f'"
                leave $*
                if [[ "x${e}x" != "xx" ]]; then
                    echo $e
                fi
                ;;
            *)
                [[ -d $f ]] && {
                    #enter $f
                    #leave $f
                    run $f
                }
                ;;
        esac
    done
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

PATH="$(dirname $PWD):${PATH##*/smart-build/bin}"
run .
check .
