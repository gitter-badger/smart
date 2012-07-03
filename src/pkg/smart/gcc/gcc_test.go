package smart_gcc

import (
        ".."
        "../t"
        "bytes"
        //"fmt"
        "os"
        "os/exec"
        "regexp"
        "strings"
        "testing"
        //"path/filepath"
)

func newTestGcc() *gcc {
        tool := &gcc{}
        if top, e := os.Getwd(); e != nil {
                // TODO: error report
        } else {
                tool.SetTop(top)
        }
        smart.ResetTargets()
        return tool
}

func TestBuildSimple(t *testing.T) {
        tt.Chdir(t, "+testdata/simple"); defer tt.Chdir(t, "-")
        tt.Checkf(t, "simple.c")

        c := newTestGcc()
        if e := smart.Scan(c.NewCollector(nil), c.top, c.top); e != nil {
                t.Errorf("scan: %v", e)
        }

        if c.target.Name != "simple" { t.Errorf("bad name: %s", c.target.Name) }
        if len(c.target.Depends) != 2 {
                t.Errorf("not 2 depends: %v", c.target.Depends)
        } else {
                if !c.target.IsFile() { t.Errorf("not file: %v", c.target) }
                if !c.target.IsGoal() { t.Errorf("not goal: %v", c.target) }

                d1 := c.target.Depends[0]
                d2 := c.target.Depends[1]
                if !d1.IsFile() { t.Errorf("not file: %v", d1) }
                if !d1.IsIntermediate() { t.Errorf("not intermediate: %v", d1) }
                if !d2.IsFile() { t.Errorf("not file: %v", d1) }
                if !d2.IsIntermediate() { t.Errorf("not intermediate: %v", d1) }
                if d1.Name != "say.c.o" && d1.Name != "simple.c.o" { t.Errorf("bad name: %s", d1.Name) }
                if d2.Name != "say.c.o" && d2.Name != "simple.c.o" { t.Errorf("bad name: %s", d2.Name) }
                if len(d1.Depends) != 1 {
                        t.Errorf("not 1 depend: %v", d1.Depends)
                } else {
                        d := d1.Depends[0]
                        if d.Name != "say.c" && d.Name != "simple.c" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile() { t.Errorf("not file: %v", d) }
                        if !d.IsScanned { t.Errorf("not scanned: %v", d) }
                }
                if len(d2.Depends) != 1 {
                        t.Errorf("not 1 depend: %v", d2.Depends)
                } else {
                        d := d2.Depends[0]
                        if d.Name != "say.c" && d.Name != "simple.c" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile() { t.Errorf("not file: %v", d) }
                        if !d.IsScanned { t.Errorf("not scanned: %v", d) }
                }
        }

        if e, _ := smart.Generate(c, c.Goals()); e != nil {
                t.Errorf("build: %v", e)
                return
        }

        tt.Checkf(t, "say.c.o")
        tt.Checkf(t, "simple.c.o")
        tt.Checkf(t, "simple");

        out := bytes.NewBuffer(nil)
        p := exec.Command("./simple")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("simple: %v", e)
        } else if string(out.Bytes()) != "smart.gcc.test.simple\n" {
                t.Errorf("simple: %v", string(out.Bytes()))
        }

        os.Remove("say.c.o")
        os.Remove("simple.c.o")
        os.Remove("simple")
}

func TestBuildCombineObjects(t *testing.T) {
        tt.Chdir(t, "+testdata/combine"); defer tt.Chdir(t, "-")
        tt.Checkd(t, "sub.o")
        tt.Checkf(t, "sub.o/sub1.c")
        tt.Checkf(t, "sub.o/sub2.c")
        tt.Checkf(t, "main.c")

        c := newTestGcc()
        if e := smart.Scan(c.NewCollector(nil), c.top, c.top); e != nil {
                t.Errorf("scan: %v", e)
        }

        if c.target.Name != "combine" { t.Errorf("bad name: %s", c.target.Name) }

        if len(c.target.Depends) != 2 {
                t.Errorf("not 2 depends: %v", c.target.Depends)
        } else {
                if !c.target.IsFile() { t.Errorf("not file: %v", c.target) }
                if !c.target.IsGoal() { t.Errorf("not goal: %v", c.target) }

                d1 := c.target.Depends[0]
                if !d1.IsFile() { t.Errorf("not file: %v", d1) }
                if !d1.IsIntermediate() { t.Errorf("not intermediate: %v", d1) }
                if d1.Name != "main.c.o" { t.Errorf("bad name: %s", d1.Name) }
                if len(d1.Depends) != 1 {
                        t.Errorf("not 1 depend: %v (%v)", d1.Depends, d1)
                } else {
                        d := d1.Depends[0]
                        if d.Name != "main.c" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile() { t.Errorf("not file: %v", d) }
                        if !d.IsScanned { t.Errorf("not scanned: %v", d) }
                }

                d2 := c.target.Depends[1]
                if !d2.IsFile() { t.Errorf("not file: %v", d2) }
                if !d2.IsGoal() { t.Errorf("not goal: %v", d2) }
                if d2.Name != "sub.o/_.o" { t.Errorf("bad name: %s", d2.Name) }
                if len(d2.Depends) != 2 {
                        t.Errorf("not 2 depends: %v (%v)", d2.Depends, d2)
                } else {
                        d := d2.Depends[0]
                        if d.Name != "sub.o/sub1.c.o" && d.Name != "sub.o/sub2.c.o" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile() { t.Errorf("not file: %v", d) }
                        if !d.IsIntermediate() { t.Errorf("not scanned: %v", d) }
                        if len(d.Depends) != 1 {
                                t.Errorf("not 1 depends: %v (%v)", d.Depends, d)
                        } else {
                                dd := d.Depends[0]
                                if dd.Name != "sub.o/sub1.c" && dd.Name != "sub.o/sub2.c" { t.Errorf("bad name: %s", dd.Name) }
                                if !dd.IsFile() { t.Errorf("not file: %v", dd) }
                                //if !dd.IsScanned { t.Errorf("not scanned: %v", dd) }
                        }

                        d = d2.Depends[1]
                        if d.Name != "sub.o/sub1.c.o" && d.Name != "sub.o/sub2.c.o" { t.Errorf("bad name: %s", d.Name) }
                        if !d.IsFile() { t.Errorf("not file: %v", d) }
                        if !d.IsIntermediate() { t.Errorf("not scanned: %v", d) }
                        if len(d.Depends) != 1 {
                                t.Errorf("not 1 depends: %v (%v)", d.Depends, d)
                        } else {
                                dd := d.Depends[0]
                                if dd.Name != "sub.o/sub1.c" && dd.Name != "sub.o/sub2.c" { t.Errorf("bad name: %s", dd.Name) }
                                if !dd.IsFile() { t.Errorf("not file: %v", dd) }
                                //if !dd.IsScanned { t.Errorf("not scanned: %v", dd) }
                        }
                }
        }

        if e, _ := smart.Generate(c, c.Goals()); e != nil {
                t.Errorf("build: %v", e)
                return
        }

        tt.Checkf(t, "main.c.o")
        tt.Checkf(t, "sub.o/sub1.c.o")
        tt.Checkf(t, "sub.o/sub2.c.o")
        tt.Checkf(t, "sub.o/_.o")
        tt.Checkd(t, "sub.o")

        out := bytes.NewBuffer(nil)
        p := exec.Command("./combine")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("combine: %v", e)
        } else if string(out.Bytes()) != "1 + 2 = 3\n" {
                t.Errorf("combine: %v", string(out.Bytes()))
        }

        os.Remove("combine")
        os.Remove("main.c.o")
        os.Remove("sub.o/_.o")
        os.Remove("sub.o/sub1.c.o")
        os.Remove("sub.o/sub2.c.o")
}

func lddrx(s string) *regexp.Regexp {
        s = `\n\s*(` + s
        s += `\s*=>\s*([^\s]+))\s*\(` //)
        return regexp.MustCompile(s)
}

func TestBuildSublibdir(t *testing.T) {
        tt.Chdir(t, "+testdata/sub_lib_dir"); defer tt.Chdir(t, "-")
        tt.Checkf(t, "main.c")
        tt.Checkd(t, "bar.a")
        tt.Checkf(t, "bar.a/bar.c")
        tt.Checkd(t, "foo.so")
        tt.Checkf(t, "foo.so/foo.c")

        os.Remove("bar.a/bar.c.o")
        os.Remove("bar.a/libbar.a")
        os.Remove("foo.so/foo.c.o")
        os.Remove("foo.so/libfoo.so")
        os.Remove("main.c.o")
        os.Remove("sub_lib_dir")

        c := newTestGcc()
        if e := smart.Scan(c.NewCollector(nil), c.top, c.top); e != nil {
                t.Errorf("scan: %v", e)
        }

        smart.Graph();

        if c.target == nil { t.Errorf("no target"); return }
        if c.target.Name != "sub_lib_dir" { t.Errorf("bad target: %s", c.target.Name) }

        if e, _ := smart.Generate(c, c.Goals()); e != nil {
                t.Errorf("build: %v", e)
                return
        }

        tt.Checkf(t, "sub_lib_dir")
        tt.Checkf(t, "main.c.o")
        tt.Checkf(t, "bar.a/bar.c.o")
        tt.Checkf(t, "bar.a/libbar.a")
        tt.Checkf(t, "foo.so/foo.c.o")
        tt.Checkf(t, "foo.so/libfoo.so")

        out := bytes.NewBuffer(nil)
        p := exec.Command("ldd", "sub_lib_dir")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("libfoo.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/libfoo.so" != s[2] {
                                t.Errorf("ldd(sub_lib_dir): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(sub_lib_dir): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("./sub_lib_dir")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("sub_lib_dir: %v", e)
        } else if string(out.Bytes()) != "foobar\n" {
                t.Errorf("sub_lib_dir: %v", string(out.Bytes()))
        }

        os.Remove("bar.a/bar.c.o")
        os.Remove("bar.a/libbar.a")
        os.Remove("foo.so/foo.c.o")
        os.Remove("foo.so/libfoo.so")
        os.Remove("main.c.o")
        os.Remove("sub_lib_dir")
}

func TestBuildSublibdirs(t *testing.T) {
        tt.Chdir(t, "+testdata/sub_lib_dirs"); defer tt.Chdir(t, "-")
        tt.Checkf(t, "main.c")
        tt.Checkd(t, "foo.so")
        tt.Checkf(t, "foo.so/foo.h")
        tt.Checkf(t, "foo.so/foo.c")
        tt.Checkd(t, "foo.so/oo.so")
        tt.Checkf(t, "foo.so/oo.so/oo.h")
        tt.Checkf(t, "foo.so/oo.so/oo.c")
        tt.Checkd(t, "foo.so/oo.so/bar.so")
        tt.Checkf(t, "foo.so/oo.so/bar.so/bar.h")
        tt.Checkf(t, "foo.so/oo.so/bar.so/bar.c")
        tt.Checkd(t, "foo.so/oo.so/bar.so/ln.so")
        tt.Checkf(t, "foo.so/oo.so/bar.so/ln.so/ln.h")
        tt.Checkf(t, "foo.so/oo.so/bar.so/ln.so/ln.c")

        os.Remove("main.c.o")
        os.Remove("sub_lib_dirs")
        os.Remove("foo.so/foo.c.o")
        os.Remove("foo.so/libfoo.so")
        os.Remove("foo.so/oo.so/oo.c.o")
        os.Remove("foo.so/oo.so/liboo.so")
        os.Remove("foo.so/oo.so/bar.so/bar.c.o")
        os.Remove("foo.so/oo.so/bar.so/libbar.so")
        os.Remove("foo.so/oo.so/bar.so/ln.so/ln.c.o")
        os.Remove("foo.so/oo.so/bar.so/ln.so/libln.so")

        c := newTestGcc()
        if e := smart.Scan(c.NewCollector(nil), c.top, c.top); e != nil {
                t.Errorf("scan: %v", e)
        }

        smart.Graph();

        if c.target == nil { t.Errorf("no target"); return }
        if c.target.Name != "sub_lib_dirs" { t.Errorf("bad target: %s", c.target.Name) }

        if e, _ := smart.Generate(c, c.Goals()); e != nil {
                t.Errorf("build: %v", e)
                return
        }

        tt.Checkf(t, "main.c.o")
        tt.Checkf(t, "sub_lib_dirs")
        tt.Checkf(t, "foo.so/foo.c.o")
        tt.Checkf(t, "foo.so/libfoo.so")
        tt.Checkf(t, "foo.so/oo.so/oo.c.o")
        tt.Checkf(t, "foo.so/oo.so/liboo.so")
        tt.Checkf(t, "foo.so/oo.so/bar.so/bar.c.o")
        tt.Checkf(t, "foo.so/oo.so/bar.so/libbar.so")
        tt.Checkf(t, "foo.so/oo.so/bar.so/ln.so/ln.c.o")
        tt.Checkf(t, "foo.so/oo.so/bar.so/ln.so/libln.so")

        out := bytes.NewBuffer(nil)
        p := exec.Command("ldd", "sub_lib_dirs")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("libfoo.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/libfoo.so" != s[2] {
                                t.Errorf("ldd(sub_lib_dirs): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(sub_lib_dirs): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("ldd", "foo.so/libfoo.so")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("liboo.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/oo.so/liboo.so" != s[2] {
                                t.Errorf("ldd(foo.so/libfoo.so): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(foo.so/libfoo.so): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("ldd", "foo.so/oo.so/liboo.so")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("libbar.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/oo.so/bar.so/libbar.so" != s[2] {
                                t.Errorf("ldd(foo.so/oo.so/liboo.so): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(foo.so/oo.so/liboo.so): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("ldd", "foo.so/oo.so/bar.so/libbar.so")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("libln.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/oo.so/bar.so/ln.so/libln.so" != s[2] {
                                t.Errorf("ldd(foo.so/oo.so/bar.so/libbar.so): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(foo.so/oo.so/bar.so/libbar.so): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("./sub_lib_dirs")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("sub_lib_dirs: %v", e)
        } else if string(out.Bytes()) != "foooobar\n" {
                t.Errorf("sub_lib_dirs: %v", string(out.Bytes()))
        }

        os.Remove("main.c.o")
        os.Remove("sub_lib_dirs")
        os.Remove("foo.so/foo.c.o")
        os.Remove("foo.so/libfoo.so")
        os.Remove("foo.so/oo.so/oo.c.o")
        os.Remove("foo.so/oo.so/liboo.so")
        os.Remove("foo.so/oo.so/bar.so/bar.c.o")
        os.Remove("foo.so/oo.so/bar.so/libbar.so")
        os.Remove("foo.so/oo.so/bar.so/ln.so/ln.c.o")
        os.Remove("foo.so/oo.so/bar.so/ln.so/libln.so")
}

func TestSmartBuild(t *testing.T) {
        tt.Chdir(t, "+testdata/sub_lib_dirs"); defer tt.Chdir(t, "-")
        tt.Checkf(t, "main.c")
        tt.Checkd(t, "foo.so")
        tt.Checkf(t, "foo.so/foo.h")
        tt.Checkf(t, "foo.so/foo.c")
        tt.Checkd(t, "foo.so/oo.so")
        tt.Checkf(t, "foo.so/oo.so/oo.h")
        tt.Checkf(t, "foo.so/oo.so/oo.c")
        tt.Checkd(t, "foo.so/oo.so/bar.so")
        tt.Checkf(t, "foo.so/oo.so/bar.so/bar.h")
        tt.Checkf(t, "foo.so/oo.so/bar.so/bar.c")
        tt.Checkd(t, "foo.so/oo.so/bar.so/ln.so")
        tt.Checkf(t, "foo.so/oo.so/bar.so/ln.so/ln.h")
        tt.Checkf(t, "foo.so/oo.so/bar.so/ln.so/ln.c")

        inters := []string{
                "main.c.o",
                "sub_lib_dirs",
                "foo.so/foo.c.o",
                "foo.so/libfoo.so",
                "foo.so/oo.so/oo.c.o",
                "foo.so/oo.so/liboo.so",
                "foo.so/oo.so/bar.so/bar.c.o",
                "foo.so/oo.so/bar.so/libbar.so",
                "foo.so/oo.so/bar.so/ln.so/ln.c.o",
                "foo.so/oo.so/bar.so/ln.so/libln.so",
        }

        removeInters := func() { for _, s := range inters { os.Remove(s) } }
        removeInters()

        c := newTestGcc()

        if e := smart.Build(c); e != nil {
                t.Errorf("build: %v", e)
                return
        }

        defer removeInters()

        if c.target == nil { t.Errorf("no target"); return }
        if c.target.Name != "sub_lib_dirs" { t.Errorf("bad target: %s", c.target.Name) }

        for _, s := range inters { tt.Checkf(t, s) }

        fiinters := make(map[string]os.FileInfo)
        fisources := make(map[string]os.FileInfo)
        for _, s := range inters {
                var fi1, fi2 os.FileInfo
                if fi, e := os.Stat(s); e != nil {
                        t.Errorf("stat: %v: %v", s, e)
                } else {
                        fiinters[s] = fi
                        fi1 = fi
                }

                if !strings.HasSuffix(s, ".c.o") {
                        continue
                }

                s = s[0:len(s)-2]

                if fi, e := os.Stat(s); e != nil {
                        t.Errorf("stat: %v: %v", s, e)
                } else {
                        fisources[s] = fi
                        fi2 = fi
                }

                if !fi1.ModTime().After(fi2.ModTime()) {
                        t.Errorf("time: `%v' !after `%v'", fi1.Name(), fi2.Name())
                }
        }

        out := bytes.NewBuffer(nil)
        p := exec.Command("ldd", "sub_lib_dirs")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("libfoo.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/libfoo.so" != s[2] {
                                t.Errorf("ldd(sub_lib_dirs): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(sub_lib_dirs): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("ldd", "foo.so/libfoo.so")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("liboo.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/oo.so/liboo.so" != s[2] {
                                t.Errorf("ldd(foo.so/libfoo.so): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(foo.so/libfoo.so): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("ldd", "foo.so/oo.so/liboo.so")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("libbar.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/oo.so/bar.so/libbar.so" != s[2] {
                                t.Errorf("ldd(foo.so/oo.so/liboo.so): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(foo.so/oo.so/liboo.so): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("ldd", "foo.so/oo.so/bar.so/libbar.so")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("ldd: %v", e)
        } else {
                re := lddrx("libln.so")
                if s := re.FindStringSubmatch(string(out.Bytes())); s != nil && 3 == len(s) {
                        if "foo.so/oo.so/bar.so/ln.so/libln.so" != s[2] {
                                t.Errorf("ldd(foo.so/oo.so/bar.so/libbar.so): %v", s[1])
                        }
                } else {
                        t.Errorf("ldd(foo.so/oo.so/bar.so/libbar.so): %v", string(out.Bytes()))
                }
        }

        out.Reset()
        p = exec.Command("./sub_lib_dirs")
        p.Stdout = out
        p.Stderr = out
        if e := p.Run(); e != nil {
                t.Errorf("sub_lib_dirs: %v", e)
        } else if string(out.Bytes()) != "foooobar\n" {
                t.Errorf("sub_lib_dirs: %v", string(out.Bytes()))
        }

        //////////////////////////////////////////////////
        // Try rebuild:

        oldTargets := smart.All()
        smart.ResetTargets()
        c.target = nil
        c.top = ""

        if e := smart.Build(c); e != nil {
                t.Errorf("rebuild: %v", e)
                return
        }

        for k, ta := range smart.All() {
                if ot, ok := oldTargets[k]; !ok {
                        t.Errorf("rebuild: mismatched: %v", k)
                } else if ot.Name != ta.Name {
                        t.Errorf("rebuild: mismatched: %v != %v", ot, ta)
                }
        }

        for _, s := range inters {
                if fi, e := os.Stat(s); e != nil {
                        t.Errorf("stat: %v: %v", s, e)
                } else {
                        if i, ok := fiinters[s]; !ok {
                                t.Errorf("FileInfo: %v", s)
                        } else if i.ModTime() != fi.ModTime() {
                                t.Errorf("ModTime: mismatched: %v", s)
                        }
                }
        }
}
