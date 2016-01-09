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

func runToolsetTestCase(t *testing.T, tn string, tf func(t *testing.T)) {
        tc := filepath.Join("../test", tn)

        if fi, _ := os.Stat(tc); fi != nil && fi.IsDir() {
                fmt.Printf("test: no test `%v' (%v)\n", tc, tn)
        }

        if wd, e := os.Getwd(); e != nil { t.Errorf("Getwd: %v", e); return } else {
                if e := os.Chdir(tc); e != nil { t.Errorf("Chdir: %v", e); return }
                fmt.Printf("test: Entering directory `%v'\n", tc)

                modules = map[string]*module{}
                moduleOrderList = []*module{}
                moduleBuildList = []pendedBuild{}

                tf(t)

                fmt.Printf("test: Leaving directory `%v'\n", tc)
                if e := os.Chdir(wd); e != nil { t.Errorf("Chdir: %v", e); return }
        }
}
