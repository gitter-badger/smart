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
        "strings"
)

var toolsets = map[string]*toolsetStub{}
var generalMetaFiles = []*filerule{
        { "backup", os.ModeDir |^ os.ModeType, `[^~]*~$` },
        //{ "git", os.ModeDir |^ os.ModeType, `\.git(ignore)?` },
        { "git", os.ModeDir, `^\.git$` },
        { "git", ^os.ModeType, `^\.gitignore$` },
        { "mercurial", os.ModeDir, `^\.hg$` },
        { "subversion", os.ModeDir, `^\.svn$` },
        { "cvs", ^os.ModeType, `^CVS$` },
}
var (
        flag_a = flag.Bool("a", false, "automode")
        flag_g = flag.Bool("g", true, "ignore names like \".git\", \".svn\", etc.")
        flag_o = flag.String("o", "", "output directory")
        flag_v = flag.Bool("v", false, "prompt command")
        flag_C = flag.String("C", "", "change directory")
        flag_T = flag.String("T", "", "traverse")
        flag_V = flag.Bool("V", false, "print command verbolly")
)

// An toolset represents a toolchain like gcc or utilities.
type toolset interface {
        setupModule(p *parser, args []string) bool
        buildModule(p *parser, args []string) bool
        /*
        processFile(dname string, fi os.FileInfo)
        updateAll()
        cleanAll()
        */
}

type toolsetStub struct {
        name string
        toolset toolset
}

/*
func (stub *toolsetStub) processFile(dname string, fi os.FileInfo) {
        stub.toolset.processFile(dname, fi)
}

func (stub *toolsetStub) auto(cmds []string) {
        for _, cmd := range cmds {
                switch cmd {
                case "update":
                        stub.toolset.updateAll()
                case "clean":
                        stub.toolset.cleanAll()
                default:
                        fmt.Printf("smart:0: unknown command '%v'\n", cmd)
                }
        }
}
*/

func registerToolset(name string, ts toolset) {
        if _, has := toolsets[name]; has {
                panic("toolset already registered: "+name)
        }

        toolsets[name] = &toolsetStub{ name:name, toolset:ts };
}

// A command executed by an action while updating a target.
type command interface {
        execute(target string, prequisites []string) bool
}

type execCommand struct {
        slient bool
        name string
        path string
        dir string
        mkdir string
        precall func() bool
        postcall func(*bytes.Buffer)
        cmd func() bool
}

func (c *execCommand) run(target string, args ...string) bool {
        if strings.HasPrefix(c.path, "~/") {
                c.path = os.Getenv("HOME") + c.path[1:]
        }
        if c.mkdir != "" {
                if e := os.MkdirAll(c.mkdir, 0755); e != nil {
                        return false
                }
        }

        var buf bytes.Buffer
        updated := false
        if c.name == "" {
                if c.cmd != nil {
                        updated = c.cmd()
                } else {
                        // TODO: should panic smart error
                        fmt.Printf("%v: %v\n", c.name, target)
                        return false
                }
        } else {
                cmd := exec.Command(c.name, args...)
                cmd.Stdout, cmd.Stderr = &buf, &buf
                if c.path != "" { cmd.Path = c.path }
                if c.dir != "" { cmd.Dir = c.dir }

                if !c.slient {
                        if *flag_v {
                                fmt.Printf("%v: %v\n", c.name, target)
                        } else if *flag_V {
                                fmt.Printf("%v\n", strings.Join(cmd.Args, " "))
                        }
                }

                if c.precall != nil && c.precall() == false {
                        return false
                }

                if err := cmd.Run(); err == nil {
                        updated = true
                } else {
                        fmt.Printf("smart:0: %s(%v):\n", c.name, err)
                        fmt.Printf("%v\n", buf.String())
                }

                if c.postcall != nil {
                        c.postcall(&buf)
                }
        }

        return updated
}

type inmemCommand interface {
        target() string
        needsUpdate() bool
}

// An action represents a action to be performed while generating a required target.
type action struct {
        target string
        prequisites []*action
        command command
}

func (a *action) getPrequisites() (l []string) {
        for _, p := range a.prequisites { l = append(l, p.target) }
        return
}

func (a *action) update() (updated bool) {
        var im inmemCommand
        if a.command != nil {
                if i, ok := a.command.(inmemCommand); ok {
                        im = i
                }
        }

        fi, err := os.Stat(a.target)
        if err != nil {
                fi = nil
        }

        //fmt.Printf("action.update: %v, %v, %v\n", a.target, a.getPrequisites(), a.command)

        updatedPreNum, outdatedNum := 0, 0
        for _, p := range a.prequisites {
                if p.update() {
                        updatedPreNum++
                        outdatedNum++
                } else if _, ok := p.command.(inmemCommand); ok {
                        //...
                } else {
                        pfi, err := os.Stat(p.target)
                        if err != nil {
                                fmt.Printf("smart:0: `%v' not found\n", p.target)
                                return // continue
                        }
                        if fi == nil || fi.ModTime().Before(pfi.ModTime()) {
                                outdatedNum++
                        }
                }
        }

        if fi == nil || 0 < updatedPreNum || 0 < outdatedNum || (im != nil && im.needsUpdate()) {
                updated = a.updateForcibly(fi)
        }
        return
}

func (a *action) updateForcibly(fi os.FileInfo) (updated bool) {
        if a.command == nil && fi == nil {
                fmt.Printf("smart: no sense for `%s'\n", a.target)
                // TODO: should panic a smart error here
                return
        }

        var pres []string
        for _, p := range a.prequisites {
                if pc, ok := p.command.(inmemCommand); ok {
                        pres = append(pres, strings.Split(pc.target(), " ")...)
                } else {
                        pres = append(pres, p.target)
                }
        }
        updated = a.command.execute(a.target, pres)

        if updated {
                if _, ok := a.command.(inmemCommand); ok {
                        // ...
                } else if _, e := os.Stat(a.target); e != nil {
                        fmt.Printf("smart:0: `%s' not built\n", a.target)
                        updated = false
                }
        }
        return
}

func (a *action) clean() {
        fmt.Printf("smart: TODO: clean `%s'\n", a.target)
}

func newAction(target string, c command, pre ...*action) *action {
        a := &action{
        command: c,
        target: target,
        prequisites: pre,
        }
        return a
}

type module struct {
        dir string
        location location // where does it defined
        name string
        toolset toolset
        kind string
        sources string
        action *action // action for building this module
        variables map[string]*variable
        using, usedBy []*module
}

var modules = map[string]*module{}

func (m *module) update() {
        //fmt.Printf("update: module: %v\n", m.name)

        if m.action == nil {
                fmt.Printf("%v: no action for module \"%v\"\n", &(m.location), m.name)
                return
        }

        if updated := m.action.update(); !updated {
                fmt.Printf("smart: Noting done for module `%v'\n", m.name)
        }
}

type filerule struct {
        name string
        mode os.FileMode
        r string //*regexp.Regexp
}

func (r *filerule) match(fi os.FileInfo) bool {
        re := regexp.MustCompile(r.r)
        if fi.Mode() & r.mode != 0 && re.MatchString(fi.Name()) {
                return true
        }
        return false
}

func (r *filerule) matchName(fn string) bool {
        re := regexp.MustCompile(r.r)
        if re.MatchString(fn) {
                return true
        }
        return false
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

func findFiles(d string, sre string, num int) (files []string, err error) {
        re := regexp.MustCompile(sre)
        err = traverse(d, func(dname string, fi os.FileInfo) bool {
                if re.MatchString(dname) {
                        files = append(files, dname)
                        num -= 1
                        if num == 0 { return false }
                }
                return true
        })
        return
}

func findFile(d string, sre string) (file string) {
        if fs, err := findFiles(d, sre, 1); err == nil {
                if 0 < len(fs) { file = fs[0] }
        }
        return
}

func copyFile(s, d string) (err error) {
        var f1, f2 *os.File
        if f1, err = os.Open(s); err == nil {
                defer f1.Close()
                if f2, err = os.Create(d); err == nil {
                        defer f2.Close()
                        if _, err = io.Copy(f2, f1); err != nil {
                                os.Remove(d)
                        }
                }
        }
        return
}

func matchFile(fi os.FileInfo, rules []*filerule) *filerule {
        for _, g := range rules {
                if g.match(fi) { return g }
        }
        return nil
}

func matchFileName(fn string, rules []*filerule) *filerule {
        for _, g := range rules {
                if g.matchName(fn) { return g }
        }
        return nil
}

func processFile(dname string, fi os.FileInfo) bool {
        fr := matchFile(fi, generalMetaFiles)

        if *flag_g && fr != nil {
                return false
        }

        if fi.Name() == ".smart" {
                if err := parse(dname); err != nil {
                        fmt.Printf("smart:0: open `%v', %v\n", dname, err)
                        return false
                }
        }

        /*
        for _, stub := range toolsets {
                stub.processFile(dname, fi)
        }
        */

        //fmt.Printf("traverse: %s\n", dname)
        return true
}

func run(vars map[string]string, cmds []string) {
        var d string
        if d = *flag_C; d == "" { d = "." }

        err := traverse(d, processFile)
        if err != nil {
                fmt.Printf("error: %v\n", err)
        }

        if *flag_a {
                /*
                for _, stub := range toolsets {
                        stub.auto(cmds)
                }
                */
        } else {
                // ...
        }

        for _, m := range modules {
                m.update()
        }
}

func Main() {
        flag.Parse()

        var vars = map[string]string{}
        var cmds []string
        for _, arg := range os.Args[1:] {
                if arg[0] == '-' { continue }
                if i := strings.Index(arg, "="); 0 < i /* false at '=foo' */ {
                        vars[arg[0:i]] = arg[i+1:]
                        continue
                }
                cmds = append(cmds, arg)
        }

        if 0 == len(cmds) {
                cmds = append(cmds, "update")
        }

        run(vars, cmds);
}
