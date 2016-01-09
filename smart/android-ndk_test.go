package smart

import (
        "path/filepath"
        "testing"
        //"fmt"
        "os"
)

func testToolsetAndroidNDK(t *testing.T) {
        if l := len(modules); l != 0 { t.Errorf("expecting len(modules) for 0, but %v", l); return }
        if l := len(moduleOrderList); l != 0 { t.Errorf("expecting len(moduleOrderList) for 0, but %v", l); return }
        if l := len(moduleBuildList); l != 0 { t.Errorf("expecting len(moduleBuildList) for 0, but %v", l); return }
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }

        Build(computeTestRunParams())

        var m *module
        var ok bool
        if m, ok = modules["foo_androidndk_so"]; !ok { t.Errorf("expecting module foo_androidndk_so"); return }
        if m.name != "foo_androidndk_so" { t.Errorf("expecting module foo_androidndk_so, but %v", m.name); return }
        if m.dir != "shared" { t.Errorf("expecting dir `shared' for `%v', but %v", m.name, m.dir); return }
        if m.kind != "shared" { t.Errorf("expecting shared for `foo_androidndk_so', but %v", m.name, m.kind); return }
        if m.action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.action.targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.action.prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.action.prequisites[0].targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].targets[0]; s != filepath.Join("shared", "na.c.o") { t.Errorf("expect shared/na.c.o, but %v (%v)", s, m.action.prequisites[0].targets); return }
        if l := len(m.action.prequisites[0].prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].prequisites[0].targets[0]; s != filepath.Join("shared", "na.c") { t.Errorf("expect shared/na.c, but %v (%v)", s, m.action.prequisites[0].prequisites[0].targets); return }

        if m, ok = modules["native_app_glue"]; !ok { t.Errorf("expecting module native_app_glue"); return }
        if m.name != "native_app_glue" { t.Errorf("expecting module native_app_glue, but %v", m.name); return }
        if m.dir != "native_app_glue" { t.Errorf("expecting dir `%v', but %v", m.name, m.dir); return }
        if m.kind != "static" { t.Errorf("expecting static for `%v', but %v", m.name, m.kind); return }
        if m.action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.action.targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.action.prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.action.prequisites[0].targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].targets[0]; s != filepath.Join("native_app_glue", "android_native_app_glue.c.o") { t.Errorf("expect native_app_glue/android_native_app_glue.c.o, but %v (%v)", s, m.action.prequisites[0].targets); return }
        if l := len(m.action.prequisites[0].prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].prequisites[0].targets[0]; s != filepath.Join("native_app_glue", "android_native_app_glue.c") { t.Errorf("expect native_app_glue/android_native_app_glue.c, but %v (%v)", s, m.action.prequisites[0].prequisites[0].targets); return }

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/native_app_glue"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/native_app_glue/libnative_app_glue.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidndk_so"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidndk_so/foo_androidndk_so.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if s, ex := runcmd("file", "out/native_app_glue/libnative_app_glue.a"), "out/native_app_glue/libnative_app_glue.a: current ar archive\n"; s != ex { t.Errorf("expectiong '%v', but: '%v'", ex, s); return }
        if s, ex := runcmd("file", "out/foo_androidndk_so/foo_androidndk_so.so"), "out/foo_androidndk_so/foo_androidndk_so.so: ELF 32-bit LSB shared object, ARM, version 1 (SYSV), dynamically linked, not stripped\n"; s != ex { t.Errorf("expectiong '%v', but: '%v'", ex, s); return }

        os.RemoveAll("out")
}

func TestToolsetAndroidNDK(t *testing.T) {
        //runToolsetTestCase(t, "android-ndk", testToolsetAndroidNDK)
}
