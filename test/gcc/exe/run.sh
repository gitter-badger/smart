if ! smart -v > temp.txt ; then
    echo "$BASH_SOURCE:$LINENO: failed building 'gcc/exe'"
fi
