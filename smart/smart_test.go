//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
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

func TestSplitVarArgs(t *testing.T) {
        vars, rest := splitVarArgs([]string{
                "a", "FOO=foo", "b", "BAR=bar", "c", " FOOBAR = foobar ",
        })

        if vars == nil || rest == nil { t.Errorf("vars and rest is invalid"); return }
        if s, ok := vars["FOO"]; !ok || s != "foo" { t.Errorf("FOO is incorrect: %v", s); return }
        if s, ok := vars["FOOBAR"]; !ok || s != "foobar" { t.Errorf("FOOBAR is incorrect: %v", s); return }
        if s, ok := vars["BAR"]; !ok || s != "bar" { t.Errorf("BAR is incorrect: %v", s); return }
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
        *flagVV, *flagV = true, true
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

func runToolsetTestCase(t *testing.T, tn string, tf func(t *testing.T)) {
        tc := filepath.Join("../test", tn)

        if fi, _ := os.Stat(tc); fi != nil && fi.IsDir() {
                fmt.Printf("test: no test `%v' (%v)\n", tc, tn)
        }

        wd, e := os.Getwd()
        if e != nil { t.Errorf("Getwd: %v", e); return }
        if e := os.Chdir(tc); e != nil { t.Errorf("Chdir: %v", e); return }
        fmt.Printf("test: Entering directory `%v'\n", tc)

        modules = map[string]*module{}
        moduleOrderList = []*module{}
        moduleBuildList = []pendedBuild{}

        tf(t)

        fmt.Printf("test: Leaving directory `%v'\n", tc)
        if e := os.Chdir(wd); e != nil { t.Errorf("Chdir: %v", e); return }
}
