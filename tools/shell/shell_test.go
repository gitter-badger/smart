//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        //"bytes"
        "testing"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testCleanFiles(t *testing.T) {
        if e := os.RemoveAll("foo"); e != nil { t.Errorf("failed remove `foo'") }
        if e := os.RemoveAll("foobar"); e != nil { t.Errorf("failed remove `foobar'") }
        if e := os.RemoveAll("o/o/o/fooo"); e != nil { t.Errorf("failed remove `o/o/o/fooo'") }
}

func testToolsetShell(t *testing.T) {
        testCleanFiles(t)

        /*
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        } */
        
        ctx := Build(make(map[string]string))
        modules := ctx.GetModules()

        var (
                m *Module
                ok bool
        )
        if m, ok = modules["touch-foo"]; !ok { t.Errorf("expecting module touch-foo"); return }
        if m.GetName(ctx) != "touch-foo" { t.Errorf("expecting touch-foo but %v", m.GetName(ctx)); return }
        if fi, e := os.Stat("foo"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if m, ok = modules["touch-foobar"]; !ok { t.Errorf("expecting module touch-foobar"); return }
        if m.GetName(ctx) != "touch-foobar" { t.Errorf("expecting touch-foobar, but %v", m.GetName(ctx)); return }
        if fi, e := os.Stat("foobar"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if m, ok = modules["touch-o-o-o-foo"]; !ok { t.Errorf("expecting module touch-o-o-o-foo"); return }
        if m.GetName(ctx) != "touch-o-o-o-foo" { t.Errorf("expecting touch-o-o-o-foo but %v", m.GetName(ctx)); return }
        if fi, e := os.Stat("o/o/o/fooo"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        //if s := info.String(); s != `touch-foobar` { t.Errorf("info: '%s'", s) }
        
        testCleanFiles(t)
}

func TestToolsetShell(t *testing.T) {
        RunToolsetTestCase(t, "../..", "shell", testToolsetShell)
}
