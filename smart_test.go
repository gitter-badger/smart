package smart

import (
        "fmt"
        "os"
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
        return
}

func testToolsetGcc(t *testing.T) {
        if l := len(modules); l != 0 { t.Errorf("expecting len(modules) for 0, but %v", l); return }
        if l := len(moduleOrderList); l != 0 { t.Errorf("expecting len(moduleOrderList) for 0, but %v", l); return }
        if l := len(moduleBuildList); l != 0 { t.Errorf("expecting len(moduleBuildList) for 0, but %v", l); return }

        //if e := os.Chdir("exe"); e != nil { t.Errorf("Chdir: %v", e); return }
        if s, e := os.Getwd(); e == nil { fmt.Printf("test: %v\n", s) }
        run(computeTestRunParams())
        //if e := os.Chdir(".."); e != nil { t.Errorf("Chdir: %v", e); return }

        var m *module
        var ok bool
        if m, ok = modules["foo_gcc_exe"]; !ok { t.Errorf("expecting module foo_gcc_exe"); return }
        if m.name != "foo_gcc_exe" { t.Errorf("expecting module foo_gcc_exe, but %v", m.name); return }
        if m, ok = modules["foo_shared"]; !ok { t.Errorf("expecting module foo_shared"); return }
        if m.name != "foo_shared" { t.Errorf("expecting module foo_shared, but %v", m.name); return }
        if m, ok = modules["foo_static"]; !ok { t.Errorf("expecting module foo_static"); return }
        if m.name != "foo_static" { t.Errorf("expecting module foo_static, but %v", m.name); return }

        if fi, e := os.Stat("out/foo_gcc_exe"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_gcc_exe/foo_gcc_exe"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_shared"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_shared/foo_shared.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_static"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_static/libfoo_static.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
}

func testToolsetAndroidNDK(t *testing.T) {
        //if e := os.Chdir("exe"); e != nil { t.Errorf("Chdir: %v", e); return }
        if s, e := os.Getwd(); e == nil { fmt.Printf("test: %v\n", s) }
        //run(computeTestRunParams())
        //if e := os.Chdir(".."); e != nil { t.Errorf("Chdir: %v", e); return }
}

func testToolsetAndroidSDK(t *testing.T) {
        //if e := os.Chdir("exe"); e != nil { t.Errorf("Chdir: %v", e); return }
        if s, e := os.Getwd(); e == nil { fmt.Printf("test: %v\n", s) }
        //run(computeTestRunParams())
        //if e := os.Chdir(".."); e != nil { t.Errorf("Chdir: %v", e); return }
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
                        testToolset(tn, tc, ts)
                        if e := os.Chdir(wd); e != nil { t.Errorf("Chdir: %v", e); return }
                }
        }
}
