package smart

import (
        "bytes"
        "fmt"
        "os"
        "os/exec"
        "strings"
        "testing"
        "path/filepath"
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

func computeTestRunParams() (vars map[string]string, cmds []string) {
        vars = map[string]string{}
        for _, arg := range os.Args[1:] {
                if arg[0] == '-' { continue }
                if i := strings.Index(arg, "="); 0 < i /* false at '=foo' */ {
                        vars[arg[0:i]] = arg[i+1:]
                        continue
                }
                cmds = append(cmds, arg)
        }
        *flag_V, *flag_v = true, false
        return
}

func runcmd(name string, args ...string) string {
        var buf bytes.Buffer
        cmd := exec.Command(name, args...)
        cmd.Stdout, cmd.Stderr = &buf, &buf
        if err := cmd.Run(); err != nil {
                fmt.Printf("test: (run: %v) %v\n", name, err)
        }
        return buf.String()
}

func testToolsetGcc(t *testing.T) {
        if l := len(modules); l != 0 { t.Errorf("expecting len(modules) for 0, but %v", l); return }
        if l := len(moduleOrderList); l != 0 { t.Errorf("expecting len(moduleOrderList) for 0, but %v", l); return }
        if l := len(moduleBuildList); l != 0 { t.Errorf("expecting len(moduleBuildList) for 0, but %v", l); return }

        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }
        run(computeTestRunParams())

        var m *module
        var ok bool
        if m, ok = modules["foo_gcc_exe"]; !ok { t.Errorf("expecting module foo_gcc_exe"); return }
        if m.name != "foo_gcc_exe" { t.Errorf("expecting module foo_gcc_exe, but %v", m.name); return }
        if m.dir != "exe" { t.Errorf("expecting dir `exe', but %v", m.dir); return }
        if m.kind != "exe" { t.Errorf("expecting exe for foo_gcc_exe, but %v", m.kind); return }
        if m.action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.action.targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if fn := filepath.Join("out", m.name, m.name); m.action.targets[0] != fn { t.Errorf("expecting action target %v, but %v", fn, m.action.targets); return }
        if l := len(m.action.prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.action.prequisites[0].targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].targets[0]; s != filepath.Join("exe", "foo.c.o") { t.Errorf("expect exe/foo.c.o, but %v (%v)", s, m.action.prequisites[0].targets); return }
        if l := len(m.action.prequisites[0].prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].prequisites[0].targets[0]; s != filepath.Join("exe", "foo.c") { t.Errorf("expect exe/foo.c, but %v (%v)", s, m.action.prequisites[0].prequisites[0].targets); return }

        if m, ok = modules["foo_shared"]; !ok { t.Errorf("expecting module foo_shared"); return }
        if m.name != "foo_shared" { t.Errorf("expecting module foo_shared, but %v", m.name); return }
        if m.dir != "shared" { t.Errorf("expecting dir `shared', but %v", m.dir); return }
        if m.kind != "shared" { t.Errorf("expecting shared for foo_shared, but %v", m.kind); return }
        if m.action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.action.targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.action.prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.action.prequisites[0].targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].targets[0]; s != filepath.Join("shared", "foo.c.o") { t.Errorf("expect shared/foo.c.o, but %v (%v)", s, m.action.prequisites[0].targets); return }
        if l := len(m.action.prequisites[0].prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].prequisites[0].targets[0]; s != filepath.Join("shared", "foo.c") { t.Errorf("expect shared/foo.c, but %v (%v)", s, m.action.prequisites[0].prequisites[0].targets); return }

        if m, ok = modules["foo_static"]; !ok { t.Errorf("expecting module foo_static"); return }
        if m.name != "foo_static" { t.Errorf("expecting module foo_static, but %v", m.name); return }
        if m.dir != "static" { t.Errorf("expecting dir `static', but %v", m.dir); return }
        if m.kind != "static" { t.Errorf("expecting static for foo_static, but %v", m.kind); return }
        if m.action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.action.targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.action.prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.action.prequisites[0].targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].targets[0]; s != filepath.Join("static", "foo.c.o") { t.Errorf("expect static/foo.c.o, but %v (%v, %v)", s, m.name, m.action.prequisites[0].targets); return }
        if l := len(m.action.prequisites[0].prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].prequisites[0].targets[0]; s != filepath.Join("static", "foo.c") { t.Errorf("expect static/foo.c, but %v (%v)", s, m.action.prequisites[0].prequisites[0].targets); return }

        if m, ok = modules["foo_gcc_exe_use_shared"]; !ok { t.Errorf("expecting module foo_gcc_exe_use_shared"); return }
        if m.name != "foo_gcc_exe_use_shared" { t.Errorf("expecting module foo_gcc_exe_use_shared, but %v", m.name); return }
        if m.dir != "exe_use_shared" { t.Errorf("expecting dir `exe_use_shared', but %v", m.dir); return }
        if m.kind != "exe" { t.Errorf("expecting exe for foo_gcc_exe, but %v", m.kind); return }
        if m.action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.action.targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if fn := filepath.Join("out", m.name, m.name); m.action.targets[0] != fn { t.Errorf("expecting action target %v, but %v", fn, m.action.targets); return }
        if l := len(m.action.prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.action.prequisites[0].targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].targets[0]; s != filepath.Join("exe_use_shared", "foo.c.o") { t.Errorf("expect exe_use_shared/foo.c.o, but %v (%v)", s, m.action.prequisites[0].targets); return }
        if l := len(m.action.prequisites[0].prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].prequisites[0].targets[0]; s != filepath.Join("exe_use_shared", "foo.c") { t.Errorf("expect exe_use_shared/foo.c, but %v (%v)", s, m.action.prequisites[0].prequisites[0].targets); return }

        if m, ok = modules["foo_gcc_exe_use_static"]; !ok { t.Errorf("expecting module foo_gcc_exe_static"); return }
        if m.name != "foo_gcc_exe_use_static" { t.Errorf("expecting module foo_gcc_exe_static, but %v", m.name); return }
        if m.dir != "exe_use_static" { t.Errorf("expecting dir `exe_use_static', but %v", m.dir); return }
        if m.kind != "exe" { t.Errorf("expecting exe for foo_gcc_exe_use_static, but %v", m.kind); return }
        if m.action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.action.targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if fn := filepath.Join("out", m.name, m.name); m.action.targets[0] != fn { t.Errorf("expecting action target %v, but %v", fn, m.action.targets); return }
        if l := len(m.action.prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.action.prequisites[0].targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].targets[0]; s != filepath.Join("exe_use_static", "foo.c.o") { t.Errorf("expect exe_use_static/foo.c.o, but %v (%v)", s, m.action.prequisites[0].targets); return }
        if l := len(m.action.prequisites[0].prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.action.prequisites[0].prequisites[0].targets[0]; s != filepath.Join("exe_use_static", "foo.c") { t.Errorf("expect exe_use_static/foo.c, but %v (%v)", s, m.action.prequisites[0].prequisites[0].targets); return }

        if fi, e := os.Stat("out/foo_shared"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_shared/foo_shared.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_static"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_static/libfoo_static.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe/foo_gcc_exe"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_static"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_static/foo_gcc_exe_use_static"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_shared"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe_use_shared/foo_gcc_exe_use_shared"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if s := runcmd("out/foo_gcc_exe/foo_gcc_exe"); s != "hello: out/foo_gcc_exe/foo_gcc_exe\n" { t.Errorf("unexpected foo_gcc_exe output: '%v'", s); return }
        if s := runcmd("out/foo_gcc_exe_use_shared/foo_gcc_exe_use_shared"); s != "hello: out/foo_gcc_exe_use_shared/foo_gcc_exe_use_shared (shared: 100)\n" { t.Errorf("unexpected foo_gcc_exe output: '%v'", s); return }
        if s := runcmd("out/foo_gcc_exe_use_static/foo_gcc_exe_use_static"); s != "hello: out/foo_gcc_exe_use_static/foo_gcc_exe_use_static (static: 100)\n" { t.Errorf("unexpected foo_gcc_exe output: '%v'", s); return }

        os.RemoveAll("out")
}

func testToolsetAndroidNDK(t *testing.T) {
        //run(computeTestRunParams())
}

func testToolsetAndroidSDK(t *testing.T) {
        //run(computeTestRunParams())
}

func TestToolsets(t *testing.T) {
        m := map[string]func(t *testing.T){
                "gcc": testToolsetGcc,
                "android-ndk": testToolsetAndroidNDK,
                "android-sdk": testToolsetAndroidSDK,
        }

        testToolset := func(tn, tc string, ts *toolsetStub) {
                if f, ok := m[tn]; ok {
                        modules = map[string]*module{}
                        moduleOrderList = []*module{}
                        moduleBuildList = []pendedBuild{}
                        f(t)
                } else {
                        t.Errorf("no test for %v (%v)", tc, tn)
                }
        }

        for tn, ts := range toolsets {
                tc := filepath.Join("test", tn)
                if fi, _ := os.Stat(tc); fi != nil && fi.IsDir() {
                        var wd string
                        if s, e := os.Getwd(); e != nil { t.Errorf("Getwd: %v", e); return } else { wd = s }
                        if e := os.Chdir(tc); e != nil { t.Errorf("Chdir: %v", e); return }
                        fmt.Printf("TestToolsets: Entering directory `%v'\n", tc)
                        testToolset(tn, tc, ts)
                        fmt.Printf("TestToolsets: Leaving directory `%v'\n", tc)
                        if e := os.Chdir(wd); e != nil { t.Errorf("Chdir: %v", e); return }
                }
        }
}
