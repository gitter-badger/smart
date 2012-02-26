package smart

import (
        //. "smart"
        "os"
        "testing"
)

func TestTraverse(t *testing.T) {
        m := map[string]bool{}
        err := traverse("main", func(fn string, fi os.FileInfo) bool {
                m[fi.Name()] = true
                return true
        })
        if err != nil { t.Errorf("error: %v", err) }
        if !m["main.go"] { t.Error("main.go not found") }
}
