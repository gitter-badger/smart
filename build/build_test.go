//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        "fmt"
        "bytes"
        "testing"
)

func TestTraverse(t *testing.T) {
        m := map[string]bool{}
        err := traverse("../data", func(fn string, fi os.FileInfo) bool {
                m[fi.Name()] = true
                return true
        })
        if err != nil      { t.Errorf("error: %v\n", err) }
        //if !m["main.go"] { t.Error("main.go not found") }
        if !m["keystore"]  { t.Error("keystore not found") }
        if !m["keypass"]   { t.Error("keypass not found") }
        if !m["storepass"] { t.Error("storepass not found") }
}

func TestBuildRules(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestBuildRules", `
all: foo bar
foo:; @touch $@.txt $(info noop: $@)
bar:
	@touch $@.txt $(info noop: $@.1)
	@echo $@ >> $@.txt $(info noop: $@.2)
`);     if err != nil { t.Errorf("parse error:", err) }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        buildInContext(ctx)
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        buildInContext(ctx, "all")
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        buildInContext(ctx, "foo")
        if fi, e := os.Stat("foo.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }
        if fi, e := os.Stat("bar.txt"); fi != nil || e == nil { t.Errorf("TestBuildRules: bar.txt should not exists!") }

        os.Remove("foo.txt")
        os.Remove("bar.txt")
        buildInContext(ctx, "bar")
        if fi, e := os.Stat("foo.txt"); fi != nil || e == nil { t.Errorf("TestBuildRules: foo.txt should not exists!") }
        if fi, e := os.Stat("bar.txt"); fi == nil || e != nil { t.Errorf("TestBuildRules: %s", e) }

        if s, x := info.String(), fmt.Sprintf(`noop: foo
noop: bar.1
noop: bar.2
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
}
