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
        if m.Name != "touch-foo" { t.Errorf("expecting touch-foo but %v", m.Name); return }
        if m.Kind != "touch" { t.Errorf("expecting touch but %v", m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if fi, e := os.Stat("foo"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if m, ok = modules["touch-foobar"]; !ok { t.Errorf("expecting module touch-foobar"); return }
        if m.Name != "touch-foobar" { t.Errorf("expecting touch-foobar, but %v", m.Name); return }
        if m.Kind != "touch" { t.Errorf("expecting touch, but %v", m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if fi, e := os.Stat("foobar"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        testCleanFiles(t)
}

func TestToolsetShell(t *testing.T) {
        RunToolsetTestCase(t, "../..", "shell", testToolsetShell)
}
