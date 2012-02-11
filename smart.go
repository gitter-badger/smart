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
        processFile(dname string, fi os.FileInfo)
        updateAll()
        cleanAll()
}

type toolsetStub struct {
        name string
        toolset toolset
}

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
        name string
}

func (c *execCommand) run(args ...string) bool {
        var buf bytes.Buffer
        cmd := exec.Command(c.name, args...)
        cmd.Stdout, cmd.Stderr = &buf, &buf
        //cmd.Dir = ""

        if *flag_v {
                fmt.Printf("%v\n", c.name)
        } else if *flag_V {
                fmt.Printf("%v\n", strings.Join(cmd.Args, " "))
        }

        updated := false
        if err := cmd.Run(); err == nil {
                updated = true
        } else {
                fmt.Printf("smart:0: %s(%v):\n", c.name, err)
                fmt.Printf("%v\n", buf.String())
        }

        return updated
}

// An action represents a action to be performed while generating a required target.
type action struct {
        target string
        prequisites []*action
        command command
}

func (a *action) update() (updated bool) {
        fi, err := os.Stat(a.target)
        if err != nil {
                //fi = nil
        }

        updatedPreNum, outdatedNum := 0, 0
        for _, p := range a.prequisites {
                if p.update() {
                        updatedPreNum++
                        outdatedNum++
                } else {
                        pfi, err := os.Stat(p.target)
                        if err != nil {
                                fmt.Printf("smart:0: '%v' not found\n", p.target)
                                return // continue
                        }
                        if fi == nil || fi.ModTime().Before(pfi.ModTime()) {
                                outdatedNum++
                        }
                }
        }

        if 0 < updatedPreNum || 0 < outdatedNum {
                updated = a.updateForcibly(fi)
        }
        return
}

func (a *action) updateForcibly(fi os.FileInfo) (updated bool) {
        if a.command == nil && fi == nil {
                fmt.Printf("smart: no sense for `%s'\n", a.target)
                // TODO: should panic here
                return
        }

        var pres []string
        for _, p := range a.prequisites { pres = append(pres, p.target) }
        updated = a.command.execute(a.target, pres)
        return
}

func (a *action) clean() {
        fmt.Printf("smart: TODO: clean `%s'\n", a.target)
}

func makeAction(target string) *action {
        a := &action{
        target: target,
        }
        return a
}

type module struct {
        dir string
        name string
        toolset string
        kind string
        sources string
        action *action // action for building this module
        variables map[string]*variable
}

func makeModule(conf string) (mod *module, err error) {
        mod = &module{
                dir: filepath.Dir(conf),
                variables: make(map[string]*variable, 200),
        }

        if err = mod.parse(conf); err != nil {
                mod = nil
                return
        }

        return
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

func matchFile(fi os.FileInfo, rules []*filerule) *filerule {
        for _, g := range rules {
                if g.match(fi) { return g }
        }
        return nil
}

func processFile(dname string, fi os.FileInfo) bool {
        fr := matchFile(fi, generalMetaFiles)

        if *flag_g && fr != nil {
                return false
        }

        var mod *module
        var err error

        if fi.Name() == ".smart" {
                mod, err = makeModule(dname)
                if err != nil {
                        fmt.Printf("smart:0: open `%v', %v\n", dname, err)
                        return false
                }
        }

        if mod == nil {}

        for _, stub := range toolsets {
                stub.processFile(dname, fi)
        }

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
                for _, stub := range toolsets {
                        stub.auto(cmds)
                }
        } else {
                
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
