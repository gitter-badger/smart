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
        "runtime"
        "strings"
        "sort"
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

type smarterror struct {
        number int
        message string
}

func (e *smarterror) String() string {
        return fmt.Sprintf("%v (%v)", e.message, e.number)
}

func errorf(num int, f string, a ...interface{}) {
        panic(&smarterror{
        number: num,
        message: fmt.Sprintf(f, a...),
        })
}

// An toolset represents a toolchain like gcc or utilities.
type toolset interface {
        setupModule(p *parser, args []string, vars map[string]string) bool
        buildModule(p *parser, args []string) bool
}

type toolsetStub struct {
        name string
        toolset toolset
}

func registerToolset(name string, ts toolset) {
        if _, has := toolsets[name]; has {
                panic("toolset already registered: "+name)
        }

        toolsets[name] = &toolsetStub{ name:name, toolset:ts };
}

// A command executed by an action while updating a target.
type command interface {
        execute(targets []string, prequisites []string) bool
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
        ia32 bool
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
                        errorf(0, "can't update `%v' (%v)", target, c.name)
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

                ia32Command := func() {
                        switch runtime.GOARCH {
                        case "amd64":
                                cmd = exec.Command("linux32", append([]string{ cmd.Path }, args...)...)
                                cmd.Stdout, cmd.Stderr = &buf, &buf
                                //fmt.Printf("%v\n", strings.Join(cmd.Args, " "))
                        }
                }

                if c.ia32 && runtime.GOOS == "linux" {
                        ia32Command()
                }

                if err := cmd.Run(); err == nil {
                        updated = true
                } else {
                        fmt.Printf("smart:0: %v:\n", err)
                        fmt.Printf("%v\n", buf.String())
                        errorf(0, "failed executing command \"%v\"", c.name)
                }

                if c.postcall != nil {
                        c.postcall(&buf)
                }
        }

        return updated
}

type incommand interface {
        command
        targets(prequisites []*action) (names []string, needsUpdate bool)
}

func computeInterTargets(d, sre string, prequisites []*action) (targets []string, outdates int, outdateMap map[int]int) {
        re := regexp.MustCompile(sre)
        outdateMap = map[int]int{}
        traverse(d, func(fn string, fi os.FileInfo) bool {
                if !re.MatchString(fn) { return true }
                var i int;
                i, targets = len(targets), append(targets, fn)
                outdateMap[i] = 0
                for _, p := range prequisites {
                        if pc, ok := p.command.(incommand); ok {
                                if _, nu := pc.targets(p.prequisites); nu {
                                        outdateMap[i] += 1
                                }
                        } else {
                                for _, t := range p.targets {
                                        if pfi, _ := os.Stat(t); pfi == nil {
                                                errorf(0, "`%v' not found", t)
                                        } else if fi.ModTime().Before(pfi.ModTime()) {
                                                outdateMap[i] += 1
                                        }
                                }
                        }
                }
                outdates += outdateMap[i]
                return true
        })
        return
}

// An action represents a action to be performed while generating a required target.
type action struct {
        targets []string
        prequisites []*action
        command command
}

func (a *action) update() (updated bool, updatedTargets []string) {
        var targets []string
        var targetsNeedUpdate bool
        var isIncommand bool
        if a.command != nil {
                if c, ok := a.command.(incommand); ok {
                        targets, targetsNeedUpdate = c.targets(a.prequisites)
                        isIncommand = true
                }
        }

        if !isIncommand {
                //fmt.Printf("targets: %v\n", a.targets)
                targets = append(targets, a.targets...)
        }

        var missingTargets, outdatedTargets []int
        var fis []os.FileInfo
        for n, s := range targets {
                if i, _ := os.Stat(s); i != nil {
                        fis = append(fis, i)
                } else {
                        fis = append(fis, nil)
                        missingTargets = append(missingTargets, n)
                }
        }

        if len(fis) != len(targets) {
                panic("internal unmatched arrayes") //errorf(-1, "internal")
        }

        updatedPreNum := 0
        prequisites := []string{}
        for _, p := range a.prequisites {
                if u, pres := p.update(); u {
                        prequisites = append(prequisites, pres...)
                        updatedPreNum++
                } else if pc, ok := p.command.(incommand); ok {
                        pres, nu := pc.targets(p.prequisites)
                        if nu { errorf(0, "requiring updating %v for %v", pres, targets) }
                        prequisites = append(prequisites, pres...)
                } else {
                        prequisites = append(prequisites, p.targets...)
                        for _, pt := range p.targets {
                                if fi, err := os.Stat(pt); err != nil {
                                        errorf(0, "`%v' not found", pt)
                                } else {
                                        for n, i := range fis {
                                                if i != nil && i.ModTime().Before(fi.ModTime()) {
                                                        outdatedTargets = append(outdatedTargets, n)
                                                }
                                        }
                                }
                        }
                }
        }

        if a.command == nil {
                for n, i := range fis {
                        if i == nil {
                                errorf(0, "`%s' not found", targets[n])
                        }
                }
                return
        }

        if 0 < updatedPreNum || targetsNeedUpdate {
                updated, updatedTargets = a.updateForcibly(targets, fis, prequisites)
        } else {
                var rr []int
                var request []string
                var requestfis []os.FileInfo

                rr = append(rr, missingTargets...)
                rr = append(rr, outdatedTargets...)
                sort.Ints(rr)

                for n, _ := range rr {
                        if n == 0 || rr[n-1] != rr[n] {
                                request = append(request, targets[rr[n]])
                                requestfis = append(requestfis, fis[rr[n]])
                        }
                }

                //fmt.Printf("targets: %v, %v, %v, %v\n", targets, request, len(a.prequisites), prequisites)
                if 0 < len(request) {
                        updated, updatedTargets = a.updateForcibly(request, requestfis, prequisites)
                }
        }

        return
}

func (a *action) updateForcibly(targets []string, tarfis []os.FileInfo, prequisites []string) (updated bool, updatedTargets []string) {
        updated = a.command.execute(targets, prequisites)

        if updated {
                var targetsNeedUpdate bool
                if c, ok := a.command.(incommand); ok {
                        updatedTargets, targetsNeedUpdate = c.targets(a.prequisites)
                        updated = !targetsNeedUpdate
                } else {
                        for _, t := range a.targets {
                                if fi, e := os.Stat(t); e != nil || fi == nil {
                                        errorf(0, "`%s' not built", t)
                                } else {
                                        updatedTargets = append(updatedTargets, t)
                                }
                        }
                }
        }
        return
}

func (a *action) clean() {
        errorf(0, "TODO: clean `%v'\n", a.targets)
}

func newAction(target string, c command, pre ...*action) *action {
        a := &action{
        command: c,
        targets: []string{ target },
        prequisites: pre,
        }
        return a
}

func newInAction(target string, c incommand, pre ...*action) *action {
        return newAction(target, c, pre...)
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
        built bool // marked as 'true' if module is built
}

var modules = map[string]*module{}

func (m *module) update() {
        //fmt.Printf("update: module: %v\n", m.name)

        if m.action == nil {
                fmt.Printf("%v: no action for module \"%v\"\n", &(m.location), m.name)
                return
        }

        if updated, _ := m.action.update(); !updated {
                fmt.Printf("smart: noting done for `%v'\n", m.name)
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
                //errorf(0, "open: %v, %v\n", err, d)
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
                        //errorf(0, "stat: %v\n", dname)
                        return
                }

                if !fun(dname, fi) {
                        continue
                }

                if fi.IsDir() {
                        if err = traverse(dname, fun); err != nil {
                                //errorf(0, "traverse: %v\n", dname)
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
                        errorf(0, "parse: `%v', %v\n", dname, err)
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
        defer func() {
                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("smart:%v: %v\n", se.number, se)
                        } else {
                                panic(e)
                        }
                }
        }()

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
