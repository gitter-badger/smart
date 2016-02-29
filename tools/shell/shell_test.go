//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        "testing"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testCleanFiles(t *testing.T) {
        if e := os.RemoveAll("foo"); e != nil { t.Errorf("failed remove `foo'") }
        if e := os.RemoveAll("foobar"); e != nil { t.Errorf("failed remove `foobar'") }
        if e := os.RemoveAll("o/o/o/foo"); e != nil { t.Errorf("failed remove `o/o/o/foo'") }
}

func testToolsetShell(t *testing.T) {
        testCleanFiles(t)

        ctx := Build(ComputeTestRunParams())
        modules := ctx.GetModules()

        var (
                m *Module
                ok bool
        )
        if m, ok = modules["touch-foo"]; !ok { t.Errorf("expecting module touch-foo"); return }
        if m.GetName(ctx) != "touch-foo" { t.Errorf("expecting touch-foo but %v", m.GetName(ctx)); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if fi, e := os.Stat("foo"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if m, ok = modules["touch-o-o-o-foo"]; !ok { t.Errorf("expecting module touch-o-o-o-foo"); return }
        if m.GetName(ctx) != "touch-o-o-o-foo" { t.Errorf("expecting touch-o-o-o-foo but %v", m.GetName(ctx)); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if fi, e := os.Stat("o/o/o/foo"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if m, ok = modules["touch-foobar"]; !ok { t.Errorf("expecting module touch-foobar"); return }
        if m.GetName(ctx) != "touch-foobar" { t.Errorf("expecting touch-foobar, but %v", m.GetName(ctx)); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if fi, e := os.Stat("foobar"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        testCleanFiles(t)
}

func TestToolsetShell(t *testing.T) {
        RunToolsetTestCase(t, "../..", "shell", testToolsetShell)
}
