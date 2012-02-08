package smart

import (
        "bytes"
        "flag"
        "fmt"
        "io"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
)

// An toolset represents a toolchain like gcc or utilities.
type toolset struct {
        name string
        noignore bool
}

func (ts *toolset) processFile(dname string, fi os.FileInfo) {
        
}

// An action represents a action to be performed while generating a required target.
type action struct {
        target string
        prequisites []*action
}

func (a *action) needsUpdate() (required bool) {
        required = true
        return
}

func (a *action) updatePrequisites() (updated int) {
        for _, p := range a.prequisites {
                if p.update() { updated++ }
        }
        return
}

func (a *action) update() (updated bool) {
        updatedPreNum := a.updatePrequisites()
        if 0 < updatedPreNum || a.needsUpdate() {
                updated = a.updateForcibly()
        }
        return
}

func (a *action) updateForcibly() (updated bool) {
        var bufOut, bufErr bytes.Buffer
        cmd := exec.Command("ls", "-l")
        cmd.Stdout, cmd.Stderr = &bufOut, &bufErr
        if err := cmd.Run(); err == nil {
                updated = true
                fmt.Printf("%s", string(bufOut.Bytes()))
        } else {
                fmt.Printf("error: %v: %s", err, string(bufErr.Bytes()))
        }
        return
}

type filerule struct {
        s string
        m os.FileMode
        r *regexp.Regexp
}

func (r *filerule) match(fi os.FileInfo) bool {
        if fi.Mode() & r.m != 0 && r.r.MatchString(fi.Name()) {
                return true
        }
        return false
}

var toolsets = map[string]*toolset{}
var root *action
var generalMetaFiles = []*filerule{
        { "backup", os.ModeDir |^ os.ModeType, regexp.MustCompile(`[^~]*~$`) },
        //{ "git", os.ModeDir |^ os.ModeType, regexp.MustCompile(`\.git(ignore)?`) },
        { "git", os.ModeDir, regexp.MustCompile(`\.git`) },
        { "git", ^os.ModeType, regexp.MustCompile(`\.gitignore`) },
        { "mercurial", os.ModeDir, regexp.MustCompile(`\.hg`) },
        { "subversion", os.ModeDir, regexp.MustCompile(`\.svn`) },
        { "cvs", ^os.ModeType, regexp.MustCompile(`CVS`) },
}
var (
        flag_a = flag.Bool("a", false, "automode")
        flag_g = flag.Bool("g", true, "ignore names like \".git\", \".svn\", etc.")
        flag_o = flag.String("o", "", "output directory")
        flag_T = flag.String("T", "", "traverse")
        flag_C = flag.String("C", "", "change directory")
)

func registerToolset(ts *toolset) {
        if _, has := toolsets[ts.name]; has {
                panic("toolset already registered: "+ts.name)
        }

        toolsets[ts.name] = ts;
}

type traverseCallback func(dname string, fi os.FileInfo) bool
func traverse(d string, fun traverseCallback) (err error) {
        fd, err := os.Open(d)
        if err != nil {
                fmt.Printf("error: Open: %v, %v\n", err, d)
                return
        }

        defer fd.Close()

        names, err := fd.Readdirnames(100)
        if err != nil {
                //fmt.Printf("Readdirnames: %v, %v\n", err, d)
                if err == io.EOF { err = nil }
                return
        }

        var fi os.FileInfo
        for _, name := range names {
                dname := filepath.Join(d, name)
                //fmt.Printf("traverse: %s\n", dname)

                fi, err = os.Stat(dname)
                if err != nil {
                        fmt.Printf("error: Stat: %v\n", dname)
                        return
                }

                if !fun(dname, fi) {
                        continue
                }

                if fi.IsDir() {
                        if err = traverse(dname, fun); err != nil {
                                fmt.Printf("error: traverse: %v\n", dname)
                                return
                        }
                        continue
                }
        }
        return
}

func isGeneralMeta(fi os.FileInfo) bool {
        for _, g := range generalMetaFiles {
                if g.match(fi) { return true }
        }
        return false
}

func processFile(dname string, fi os.FileInfo) bool {
        if *flag_g && isGeneralMeta(fi) {
                return false
        }

        for _, ts := range toolsets {
                ts.processFile(dname, fi)
        }

        fmt.Printf("traverse: %s\n", dname)
        return true
}

func auto() {
        var d string
        if d = *flag_C; d == "" { d = "." }

        for name, ts := range toolsets {
                fmt.Printf("toolset: %v, %v\n", name, ts)
        }

        err := traverse(d, processFile)
        if err != nil {
                fmt.Printf("error: %v\n", err)
        }
}

func Main() {
        //fmt.Printf("args: %v\n", os.Args)
        flag.Parse()

        if *flag_a {
                auto();
        }

        if root != nil {
                fmt.Printf("target: %s\n", root.target)
                root.update()
        }
}
