//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "path/filepath"
        "testing"
        //"fmt"
        "os"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testToolsetGcc(t *testing.T) {
        modules, moduleOrderList, moduleBuildList := GetModules(), GetModuleOrderList(), GetModuleBuildList()
        if l := len(modules); l != 0 { t.Errorf("expecting len(modules) for 0, but %v", l); return }
        if l := len(moduleOrderList); l != 0 { t.Errorf("expecting len(moduleOrderList) for 0, but %v", l); return }
        if l := len(moduleBuildList); l != 0 { t.Errorf("expecting len(moduleBuildList) for 0, but %v", l); return }
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }

        Build(ComputeTestRunParams())

        var m *Module
        var ok bool
        if m, ok = modules["foo_gcc_exe"]; !ok { t.Errorf("expecting module foo_gcc_exe"); return }
        if m.Name != "foo_gcc_exe" { t.Errorf("expecting module foo_gcc_exe, but %v", m.Name); return }
        if m.GetDir() != "exe" { t.Errorf("expecting dir `exe', but %v", m.GetDir()); return }
        if m.Kind != "exe" { t.Errorf("expecting exe for foo_gcc_exe, but %v", m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.Action.Targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if fn := filepath.Join("out", m.Name, m.Name); m.Action.Targets[0] != fn { t.Errorf("expecting action target %v, but %v", fn, m.Action.Targets); return }
        if l := len(m.Action.Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.Action.Prequisites[0].Targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Targets[0]; s != filepath.Join("exe", "foo.c.o") { t.Errorf("expect exe/foo.c.o, but %v (%v)", s, m.Action.Prequisites[0].Targets); return }
        if l := len(m.Action.Prequisites[0].Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Prequisites[0].Targets[0]; s != filepath.Join("exe", "foo.c") { t.Errorf("expect exe/foo.c, but %v (%v)", s, m.Action.Prequisites[0].Prequisites[0].Targets); return }

        if m, ok = modules["foo_shared"]; !ok { t.Errorf("expecting module foo_shared"); return }
        if m.Name != "foo_shared" { t.Errorf("expecting module foo_shared, but %v", m.Name); return }
        if m.GetDir() != "shared" { t.Errorf("expecting dir `shared', but %v", m.GetDir()); return }
        if m.Kind != "shared" { t.Errorf("expecting shared for foo_shared, but %v", m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.Action.Targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.Action.Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.Action.Prequisites[0].Targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Targets[0]; s != filepath.Join("shared", "foo.c.o") { t.Errorf("expect shared/foo.c.o, but %v (%v)", s, m.Action.Prequisites[0].Targets); return }
        if l := len(m.Action.Prequisites[0].Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Prequisites[0].Targets[0]; s != filepath.Join("shared", "foo.c") { t.Errorf("expect shared/foo.c, but %v (%v)", s, m.Action.Prequisites[0].Prequisites[0].Targets); return }

        if m, ok = modules["foo_static"]; !ok { t.Errorf("expecting module foo_static"); return }
        if m.Name != "foo_static" { t.Errorf("expecting module foo_static, but %v", m.Name); return }
        if m.GetDir() != "static" { t.Errorf("expecting dir `static', but %v", m.GetDir()); return }
        if m.Kind != "static" { t.Errorf("expecting static for foo_static, but %v", m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.Action.Targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.Action.Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.Action.Prequisites[0].Targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Targets[0]; s != filepath.Join("static", "foo.c.o") { t.Errorf("expect static/foo.c.o, but %v (%v, %v)", s, m.Name, m.Action.Prequisites[0].Targets); return }
        if l := len(m.Action.Prequisites[0].Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Prequisites[0].Targets[0]; s != filepath.Join("static", "foo.c") { t.Errorf("expect static/foo.c, but %v (%v)", s, m.Action.Prequisites[0].Prequisites[0].Targets); return }

        if m, ok = modules["foo_gcc_exe_use_shared"]; !ok { t.Errorf("expecting module foo_gcc_exe_use_shared"); return }
        if m.Name != "foo_gcc_exe_use_shared" { t.Errorf("expecting module foo_gcc_exe_use_shared, but %v", m.Name); return }
        if m.GetDir() != "exe_use_shared" { t.Errorf("expecting dir `exe_use_shared', but %v", m.GetDir()); return }
        if m.Kind != "exe" { t.Errorf("expecting exe for foo_gcc_exe, but %v", m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.Action.Targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if fn := filepath.Join("out", m.Name, m.Name); m.Action.Targets[0] != fn { t.Errorf("expecting action target %v, but %v", fn, m.Action.Targets); return }
        if l := len(m.Action.Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.Action.Prequisites[0].Targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Targets[0]; s != filepath.Join("exe_use_shared", "foo.c.o") { t.Errorf("expect exe_use_shared/foo.c.o, but %v (%v)", s, m.Action.Prequisites[0].Targets); return }
        if l := len(m.Action.Prequisites[0].Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Prequisites[0].Targets[0]; s != filepath.Join("exe_use_shared", "foo.c") { t.Errorf("expect exe_use_shared/foo.c, but %v (%v)", s, m.Action.Prequisites[0].Prequisites[0].Targets); return }

        if m, ok = modules["foo_gcc_exe_use_static"]; !ok { t.Errorf("expecting module foo_gcc_exe_static"); return }
        if m.Name != "foo_gcc_exe_use_static" { t.Errorf("expecting module foo_gcc_exe_static, but %v", m.Name); return }
        if m.GetDir() != "exe_use_static" { t.Errorf("expecting dir `exe_use_static', but %v", m.GetDir()); return }
        if m.Kind != "exe" { t.Errorf("expecting exe for foo_gcc_exe_use_static, but %v", m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.Action.Targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if fn := filepath.Join("out", m.Name, m.Name); m.Action.Targets[0] != fn { t.Errorf("expecting action target %v, but %v", fn, m.Action.Targets); return }
        if l := len(m.Action.Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.Action.Prequisites[0].Targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Targets[0]; s != filepath.Join("exe_use_static", "foo.c.o") { t.Errorf("expect exe_use_static/foo.c.o, but %v (%v)", s, m.Action.Prequisites[0].Targets); return }
        if l := len(m.Action.Prequisites[0].Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Prequisites[0].Targets[0]; s != filepath.Join("exe_use_static", "foo.c") { t.Errorf("expect exe_use_static/foo.c, but %v (%v)", s, m.Action.Prequisites[0].Prequisites[0].Targets); return }

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_shared"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_shared/foo_shared.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_static"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_static/libfoo_static.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe"); fi == nil || e != nil || !fi.IsDir() || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe/foo_gcc_exe"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_static"); fi == nil || e != nil || !fi.IsDir() || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_static/foo_gcc_exe_use_static"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_shared"); fi == nil || e != nil || !fi.IsDir() || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_shared/foo_gcc_exe_use_shared"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if s := Runcmd("out/foo_gcc_exe/foo_gcc_exe"); s != "hello: out/foo_gcc_exe/foo_gcc_exe\n" { t.Errorf("unexpected foo_gcc_exe output: '%v'", s); return }
        if s := Runcmd("out/foo_gcc_exe_use_shared/foo_gcc_exe_use_shared"); s != "hello: out/foo_gcc_exe_use_shared/foo_gcc_exe_use_shared (shared: 100)\n" { t.Errorf("unexpected foo_gcc_exe output: '%v'", s); return }
        if s := Runcmd("out/foo_gcc_exe_use_static/foo_gcc_exe_use_static"); s != "hello: out/foo_gcc_exe_use_static/foo_gcc_exe_use_static (static: 100)\n" { t.Errorf("unexpected foo_gcc_exe output: '%v'", s); return }

        os.RemoveAll("out")
}

func TestToolsetGCC(t *testing.T) {
        RunToolsetTestCase(t, "gcc", testToolsetGcc)
}
