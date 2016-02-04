//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        "testing"
)

func TestTraverse(t *testing.T) {
        m := map[string]bool{}
        err := traverse("../data", func(fn string, fi os.FileInfo) bool {
                m[fi.Name()] = true
                return true
        })
        if err != nil { t.Errorf("error: %v\n", err) }
        //if !m["main.go"] { t.Error("main.go not found") }
        if !m["keystore"] { t.Error("keystore not found") }
        if !m["keypass"] { t.Error("keypass not found") }
        if !m["storepass"] { t.Error("storepass not found") }
}
