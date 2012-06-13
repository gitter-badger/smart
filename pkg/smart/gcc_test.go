package smart

import (
        "bytes"
        "fmt"
        "os"
        "os/exec"
        "strings"
        "testing"
        //"path/filepath"
)

func getwd(t *testing.T) string {
        if s, e := os.Getwd(); e == nil {
                return s
        } else {
                t.Errorf("Getwd: %v", e)
        }
        return ""
}

var topdir string
var workdir [][]string
func chdir(t *testing.T, d string) error {
        if d == "-" && 0 < len(workdir) {
                a := workdir[len(workdir)-1]
                d = a[0]
                fmt.Printf("test: Leaving directory `%s'\n", a[1])
        } else if strings.HasPrefix(d, "+") {
                d = d[1:]
                workdir = append(workdir, []string{ getwd(t), d })
                fmt.Printf("test: Entering directory `%s'\n", d)
        }
        if e := os.Chdir(d); e != nil {
                t.Errorf("Chdir: %v", e)
                return e
        }
        return nil
}

func checkf(t *testing.T, fn string) {
        if fi, e := os.Stat(fn); fi == nil || e != nil {
                t.Errorf("%v", e)
        }
}

func checkd(t *testing.T, fn string) {
        if fi, e := os.Stat(fn); fi == nil || e != nil {
                t.Errorf("%v", e)
        } else if !fi.IsDir() {
                t.Errorf("NotDir: %v", fi)
        }
}

func newGcc() *gcc {
        tool := &gcc{}
        if top, e := os.Getwd(); e != nil {
                // TODO: error report
        } else {
                tool.SetTop(top)
        }
        return tool
}

func TestBuildSimple(t *testing.T) {
        chdir(t, "+testdata/gcc/simple")
        checkf(t, "simple.c")

        c := newGcc()
        if e := scan(c, c.top, c.top); e != nil {
                t.Errorf("scan: %v", e)
        }

        if c.target.Name != "simple" { t.Errorf("bad name: %s", c.target.Name) }
        if len(c.target.Depends) != 2 {
                t.Errorf("not 2 depends: %v", c.target.Depends)
        } else {
                if !c.target.IsFile { t.Errorf("not file: %v", c.target) }
                if !c.target.IsGoal { t.Errorf("not goal: %v", c.target) }

                d1 := c.target.Depends[0]
                d2 := c.target.Depends[1]
                if !d1.IsFile { t.Errorf("not file: %v", d1) }
                if !d1.IsIntermediate { t.Errorf("not intermediate: %v", d1) }
                if !d2.IsFile { t.Errorf("not file: %v", d1) }
                if !d2.IsIntermediate { t.Errorf("not intermediate: %v", d1) }
                if d1.Name != "say.c.o" && d1.Name != "simple.c.o" { t.Errorf("bad name: %s", d1.Name) }
                if d2.Name != "say.c.o" && d2.Name != "simple.c.o" { t.Errorf("bad name: %s", d2.Name) }
                if len(d1.Depends) != 1 {
                        t.Errorf("not 1 depend: %v", d1.Depends)
                } else {
                        d := d1.Depends[0]
                        if d.Name != "say.c" && d.Name != "simple.c" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile { t.Errorf("not file: %v", d) }
                        if !d.IsScanned { t.Errorf("not scanned: %v", d) }
                }
                if len(d2.Depends) != 1 {
                        t.Errorf("not 1 depend: %v", d2.Depends)
                } else {
                        d := d2.Depends[0]
                        if d.Name != "say.c" && d.Name != "simple.c" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile { t.Errorf("not file: %v", d) }
                        if !d.IsScanned { t.Errorf("not scanned: %v", d) }
                }
        }

        if e := c.Build(); e != nil {
                t.Errorf("build: %v", e)
        }

        checkf(t, "say.c.o")
        checkf(t, "simple.c.o")
        checkf(t, "simple");

        out := bytes.NewBuffer(nil)
        p := exec.Command("./simple")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("simple: %v", e)
        }
        if string(out.Bytes()) != "smart.gcc.test.simple\n" {
                t.Errorf("simple: %v", string(out.Bytes()))
        }

        os.Remove("say.c.o")
        os.Remove("simple.c.o")
        os.Remove("simple")

        chdir(t, "-")
}

func TestBuildCombineObjects(t *testing.T) {
        chdir(t, "+testdata/gcc/combine")
        checkd(t, "sub.o")
        checkf(t, "sub.o/sub1.c")
        checkf(t, "sub.o/sub2.c")
        checkf(t, "main.c")

        c := newGcc()
        if e := scan(c, c.top, c.top); e != nil {
                t.Errorf("scan: %v", e)
        }

        if c.target.Name != "combine" { t.Errorf("bad name: %s", c.target.Name) }

        if len(c.target.Depends) != 2 {
                t.Errorf("not 2 depends: %v", c.target.Depends)
        } else {
                if !c.target.IsFile { t.Errorf("not file: %v", c.target) }
                if !c.target.IsGoal { t.Errorf("not goal: %v", c.target) }

                d1 := c.target.Depends[0]
                if !d1.IsFile { t.Errorf("not file: %v", d1) }
                if !d1.IsIntermediate { t.Errorf("not intermediate: %v", d1) }
                if d1.Name != "main.c.o" { t.Errorf("bad name: %s", d1.Name) }
                if len(d1.Depends) != 1 {
                        t.Errorf("not 1 depend: %v (%v)", d1.Depends, d1)
                } else {
                        d := d1.Depends[0]
                        if d.Name != "main.c" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile { t.Errorf("not file: %v", d) }
                        if !d.IsScanned { t.Errorf("not scanned: %v", d) }
                }

                d2 := c.target.Depends[1]
                if !d2.IsFile { t.Errorf("not file: %v", d2) }
                if !d2.IsGoal { t.Errorf("not goal: %v", d2) }
                if d2.Name != "sub.o/_.o" { t.Errorf("bad name: %s", d2.Name) }
                if len(d2.Depends) != 2 {
                        t.Errorf("not 2 depends: %v (%v)", d2.Depends, d2)
                } else {
                        d := d2.Depends[0]
                        if d.Name != "sub.o/sub1.c.o" && d.Name != "sub.o/sub2.c.o" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile { t.Errorf("not file: %v", d) }
                        if !d.IsIntermediate { t.Errorf("not scanned: %v", d) }
                        if len(d.Depends) != 1 {
                                t.Errorf("not 1 depends: %v (%v)", d.Depends, d)
                        } else {
                                dd := d.Depends[0]
                                if dd.Name != "sub.o/sub1.c" && dd.Name != "sub.o/sub2.c" { t.Errorf("bad name: %s", dd.Name) }
                                if !dd.IsFile { t.Errorf("not file: %v", dd) }
                                //if !dd.IsScanned { t.Errorf("not scanned: %v", dd) }
                        }

                        d = d2.Depends[1]
                        if d.Name != "sub.o/sub1.c.o" && d.Name != "sub.o/sub2.c.o" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile { t.Errorf("not file: %v", d) }
                        if !d.IsIntermediate { t.Errorf("not scanned: %v", d) }
                        if len(d.Depends) != 1 {
                                t.Errorf("not 1 depends: %v (%v)", d.Depends, d)
                        } else {
                                dd := d.Depends[0]
                                if dd.Name != "sub.o/sub1.c" && dd.Name != "sub.o/sub2.c" { t.Errorf("bad name: %s", dd.Name) }
                                if !dd.IsFile { t.Errorf("not file: %v", dd) }
                                //if !dd.IsScanned { t.Errorf("not scanned: %v", dd) }
                        }
                }
        }

        if e := c.Build(); e != nil {
                t.Errorf("build: %v", e)
        }

        checkf(t, "main.c.o")
        checkf(t, "sub.o/sub1.c.o")
        checkf(t, "sub.o/sub2.c.o")
        checkf(t, "sub.o/_.o")
        checkd(t, "sub.o")

        out := bytes.NewBuffer(nil)
        p := exec.Command("./combine")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("combine: %v", e)
        }
        if string(out.Bytes()) != "1 + 2 = 3\n" {
                t.Errorf("combine: %v", string(out.Bytes()))
        }

        os.Remove("combine")
        os.Remove("main.c.o")
        os.Remove("sub.o/_.o")
        os.Remove("sub.o/sub1.c.o")
        os.Remove("sub.o/sub2.c.o")

        chdir(t, "-")
}

func TestBuildComplex(t *testing.T) {
        chdir(t, "+testdata/gcc/complex")
        checkf(t, "main.c")

        c := newGcc()
        if e := scan(c, c.top, c.top); e != nil {
                t.Errorf("scan: %v", e)
        }

        if c.target == nil { t.Errorf("no target"); return }
        if c.target.Name != "complex" { t.Errorf("bad name: %s", c.target.Name) }

        if e := c.Build(); e != nil {
                t.Errorf("build: %v", e)
        }

        checkf(t, "main.c.o")

        out := bytes.NewBuffer(nil)
        p := exec.Command("./complex")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("complex: %v", e)
        }
        if string(out.Bytes()) != "1 + 2 = 3\n" {
                t.Errorf("complex: %v", string(out.Bytes()))
        }

        os.Remove("main.c.o")
        os.Remove("complex")

        chdir(t, "-")
}
