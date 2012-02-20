smart > temp.txt
diff=`diff temp.txt expect.txt`
if [[ $? != 0 ]]; then
    echo "$BASH_SOURCE:$LINENO: unexpected 'smart' output"
    echo "========== DIFF begin =========="
    echo "$diff"
    echo "========== DIFF end ============"
fi
rm temp.txt
