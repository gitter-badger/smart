package smart_test

import (
        . "./smart"
        "testing"
        "bytes"
        "bufio"
)

func TestParse(t *testing.T) {
        var s = `
a = a
i = i
sh$ared = shared
stat$ic = static
a$$a = foo
xxx$(use $(sh$ared),$(stat$ic))-$(a$$a)-xxx
`
        var buf = bytes.NewBufferString(s)
        p := &parser{
                file:"test",
                in:bufio.NewReader(buf),
                variables:make(map[string]*variable, 200),
        }
        if err = p.parse(); err != nil {
                return
        }
}
