//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "bytes"
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

var (
        toolsets = map[string]*toolsetStub{}

        generalMetaFiles = []*FileMatchRule{
                { "backup", os.ModeDir |^ os.ModeType, `[^~]*~$` },
                //{ "git", os.ModeDir |^ os.ModeType, `\.git(ignore)?` },
                { "git", os.ModeDir, `^\.git$` },
                { "git", ^os.ModeType, `^\.gitignore$` },
                { "mercurial", os.ModeDir, `^\.hg$` },
                { "subversion", os.ModeDir, `^\.svn$` },
                { "cvs", ^os.ModeType, `^CVS$` },
        }
)

// toolset represents a toolchain like gcc and related utilities.
type toolset interface {
        // ConfigModule setup the current module being processed by parser.
        // `args' and `vars' is passed in on the `$(module)' invocation.
        ConfigModule(p *Context, args []string, vars map[string]string) bool

        // CreateActions creates the module action graph
        CreateActions(p *Context) bool

        // UseModule
        UseModule(p *Context, o *Module) bool
}

type toolsetStub struct {
        name string
        toolset toolset
}

func RegisterToolset(name string, ts toolset) {
        if _, has := toolsets[name]; has {
                panic("toolset already registered: "+name)
        }

        toolsets[name] = &toolsetStub{ name:name, toolset:ts };
}

type BasicToolset struct {        
}

func (tt *BasicToolset) ConfigModule(ctx *Context, args []string, vars map[string]string) bool {
        return false
}

func (tt *BasicToolset) CreateActions(ctx *Context) bool {
        return false
}

func (tt *BasicToolset) UseModule(ctx *Context, o *Module) bool {
        return false
}

func IsIA32Command(s string) bool {
        buf := new(bytes.Buffer)
        cmd := exec.Command("file", "-b", s)
        cmd.Stdout = buf
        if err := cmd.Run(); err != nil {
                message("error: %v", err)
        }
        //message("%v", buf.String())
        //return strings.HasPrefix(buf.String(), "ELF 32-bit")
        return strings.Contains(buf.String(), "ELF 32-bit")
}

// A command executed by an action while updating a target.
type Command interface {
        Execute(targets []string, prequisites []string) bool
}

type Excmd struct {
        path string
        dir string
        mkdir string
        precall func() bool
        postcall func(so, se *bytes.Buffer)
        cmd func() bool
        ia32 bool
        stdout, stderr *bytes.Buffer
        stdin io.Reader
}

func NewExcmd(s string) *Excmd {
        return &Excmd{ path:s }
}

func (c *Excmd) GetPath() string { return c.path }
func (c *Excmd) SetPath(s string) { c.path = s }
func (c *Excmd) GetIA32() bool { return c.ia32 }
func (c *Excmd) SetIA32(v bool) { c.ia32 = v }
func (c *Excmd) SetMkdir(s string) { c.mkdir = s }
func (c *Excmd) GetMkdir() string { return c.mkdir }
func (c *Excmd) SetDir(s string) { c.dir = s }
func (c *Excmd) GetDir() string { return c.dir }
func (c *Excmd) SetStderr(s *bytes.Buffer/*io.Writer*/) { c.stderr = s }
func (c *Excmd) GetStderr() *bytes.Buffer/*io.Writer*/ { return c.stderr }
func (c *Excmd) SetStdout(s *bytes.Buffer/*io.Writer*/) { c.stdout = s }
func (c *Excmd) GetStdout() *bytes.Buffer/*io.Writer*/ { return c.stdout }
func (c *Excmd) SetStdin(s io.Reader) { c.stdin = s }
func (c *Excmd) GetStdin() io.Reader { return c.stdin }
func (c *Excmd) Run(targetHint string, args ...string) bool {
        return c.run(targetHint, args...)
}

func (c *Excmd) run(targetHint string, args ...string) bool {
        if strings.HasPrefix(c.path, "~/") {
                c.path = os.Getenv("HOME") + c.path[1:]
        }
        if c.mkdir != "" {
                if e := os.MkdirAll(c.mkdir, 0755); e != nil {
                        return false
                }
        }

        if c.stdout == nil { c.stdout = new(bytes.Buffer) }
        if c.stderr == nil { c.stderr = new(bytes.Buffer) }
        c.stdout.Reset()
        c.stderr.Reset()

        updated := false
        if c.path == "" {
                if c.cmd != nil {
                        updated = c.cmd()
                } else {
                        errorf("can't update `%v'", targetHint)
                        return false
                }
        } else {
                cmd := exec.Command(c.path, args...)
                cmd.Stdout, cmd.Stderr = c.stdout, c.stderr
                if c.stdin != nil { cmd.Stdin = c.stdin }
                if c.dir != "" { cmd.Dir = c.dir }

                if *flagV {
                        if targetHint != "" {
                                message("%v -> %v", filepath.Base(c.path), targetHint)
                        }
                } else if *flagVV {
                        fmt.Printf("%v\n", strings.Join(cmd.Args, " "))
                }

                if c.precall != nil && c.precall() == false {
                        return false
                }

                if c.ia32 && runtime.GOOS == "linux" {
                        switch runtime.GOARCH {
                        case "amd64":
                                cmd = exec.Command("linux32", append([]string{ cmd.Path }, args...)...)
                                cmd.Stdout, cmd.Stderr = c.stdout, c.stderr
                                if c.stdin != nil { cmd.Stdin = c.stdin }
                                //fmt.Printf("%v\n", strings.Join(cmd.Args, " "))
                        }
                }

                if err := cmd.Run(); err == nil {
                        updated = true
                } else {
                        message("%v", err)
                        if c.path != "" {
                                message(`"%v %v"`, c.path, strings.Join(args, " "))
                        }
                        so, se := c.stdout.String(), c.stderr.String()
                        if so != "" {
                                fmt.Printf("------------------------------ stdout\n")
                                fmt.Printf("%v", so)
                                if !strings.HasSuffix(so, "\n") { fmt.Printf("\n") }
                        }
                        if se != "" {
                                fmt.Printf("------------------------------ stderr\n")
                                fmt.Printf("%v", se)
                                if !strings.HasSuffix(se, "\n") { fmt.Printf("\n") }
                        }
                        fmt.Printf("-------------------------------------\n")
                        errorf(`failed executing "%v"`, c.path)
                }

                if c.postcall != nil {
                        c.postcall(c.stdout, c.stderr)
                }
        }

        return updated
}

// intercommand represents a intermdiate action command
type intercommand interface {
        Command
        targets(prequisites []*Action) (names []string, needsUpdate bool)
}

func ComputeInterTargets(d, sre string, prequisites []*Action) (targets []string, outdates int, outdateMap map[int]int) {
        re := regexp.MustCompile(sre)
        outdateMap = map[int]int{}
        traverse(d, func(fn string, fi os.FileInfo) bool {
                if !re.MatchString(fn) { return true }
                i := len(targets)
                outdateMap[i] = 0
                targets = append(targets, fn)
                for _, p := range prequisites {
                        if pc, ok := p.Command.(intercommand); ok {
                                if _, needsUpdate := pc.targets(p.Prequisites); needsUpdate {
                                        outdateMap[i]++
                                }
                        } else {
                                for _, t := range p.Targets {
                                        if pfi, _ := os.Stat(t); pfi == nil {
                                                errorf("`%v' not found", t)
                                        } else if fi.ModTime().Before(pfi.ModTime()) {
                                                outdateMap[i]++
                                        }
                                }
                        }
                }
                outdates += outdateMap[i]
                return true
        })
        return
}

// Action performs a command for updating targets
type Action struct {
        Targets []string
        Prequisites []*Action
        Command Command
}

func (a *Action) update() (updated bool, updatedTargets []string) {
        var targets []string
        var targetsNeedUpdate bool
        var isIntercommand bool
        if a.Command != nil {
                if c, ok := a.Command.(intercommand); ok {
                        targets, targetsNeedUpdate = c.targets(a.Prequisites)
                        isIntercommand = true
                }
        }

        if !isIntercommand {
                //fmt.Printf("targets: %v\n", a.targets)
                targets = append(targets, a.Targets...)
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
                panic("internal unmatched arrays") //errorf(-1, "internal")
        }

        updatedPreNum := 0
        prequisites := []string{}
        for _, p := range a.Prequisites {
                if u, pres := p.update(); u {
                        prequisites = append(prequisites, pres...)
                        updatedPreNum++
                } else if pc, ok := p.Command.(intercommand); ok {
                        pres, nu := pc.targets(p.Prequisites)
                        if nu { errorf("requiring updating %v for %v", pres, targets) }
                        prequisites = append(prequisites, pres...)
                } else {
                        prequisites = append(prequisites, p.Targets...)
                        for _, pt := range p.Targets {
                                if fi, err := os.Stat(pt); err != nil {
                                        errorf("`%v' not found", pt)
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

        if a.Command == nil {
                for n, i := range fis {
                        if i == nil {
                                errorf("`%s' not found", targets[n])
                        }
                }
                return
        }

        if 0 < updatedPreNum || targetsNeedUpdate {
                updated, updatedTargets = a.force(targets, fis, prequisites)
        } else {
                var rr []int
                var request []string
                var requestfis []os.FileInfo

                rr = append(rr, missingTargets...)
                rr = append(rr, outdatedTargets...)
                sort.Ints(rr)

                for n := range rr {
                        if n == 0 || rr[n-1] != rr[n] {
                                request = append(request, targets[rr[n]])
                                requestfis = append(requestfis, fis[rr[n]])
                        }
                }

                //fmt.Printf("targets: %v, %v, %v, %v\n", targets, request, len(a.prequisites), prequisites)
                if 0 < len(request) {
                        updated, updatedTargets = a.force(request, requestfis, prequisites)
                }
        }

        return
}

func (a *Action) force(targets []string, tarfis []os.FileInfo, prequisites []string) (updated bool, updatedTargets []string) {
        updated = a.Command.Execute(targets, prequisites)

        if updated {
                var targetsNeedUpdate bool
                if c, ok := a.Command.(intercommand); ok {
                        updatedTargets, targetsNeedUpdate = c.targets(a.Prequisites)
                        updated = !targetsNeedUpdate
                } else {
                        for _, t := range a.Targets {
                                if fi, e := os.Stat(t); e != nil || fi == nil {
                                        errorf("`%s' not built", t)
                                } else {
                                        updatedTargets = append(updatedTargets, t)
                                }
                        }
                }
        }
        return
}

func (a *Action) clean() {
        errorf("TODO: clean `%v'\n", a.Targets)
}

func newAction(target string, c Command, pre ...*Action) *Action {
        a := &Action{
                Command: c,
                Targets: []string{ target },
                Prequisites: pre,
        }
        return a
}

func NewAction(target string, c Command, pre ...*Action) *Action {
        return newAction(target, c, pre...)
}

func CreateSourceTransformActions(sources []string, namecommand func(src string) (string, Command)) []*Action {
        var inters []*Action
        if namecommand == nil {
                errorf("can't draw source rules (%v)", namecommand)
        }

        for _, src := range sources {
                aname, c := namecommand(src)
                if aname == "" { continue }
                if aname == src {
                        errorf("no intermediate name for `%v'", src)
                }

                if c == nil {
                        errorf("no command for `%v'", src)
                }

                asrc := newAction(src, nil)
                a := newAction(aname, c, asrc)
                inters = append(inters, a)
        }
        return inters
}

// Module is defined by a $(module) invocation in .smart script.
type Module struct {
        Parent *Module // upper module
        Name, Kind string
        Toolset toolset
        Action *Action // action for building this module
        Using, UsedBy []*Module
        Updated bool // marked as 'true' if module is updated
        defines map[string]*define
        declareLoc, commitLoc location // where does it defined and commit (could be nil)
        l *lex // the lex scope where does it defined (could be nil)
}

func (m *Module) GetDir() (s string) {
        if m.l != nil {
                s = filepath.Dir(m.l.scope)
        }
        return
}
func (m *Module) GetDeclareLocation() (s string, lineno, colno int) {
        if l := m.l; l != nil {
                lineno, colno = l.caculateLocationLineColumn(m.declareLoc)
                s = l.scope
        }
        return
}
func (m *Module) GetCommitLocation() (s string, lineno, colno int) {
        if l := m.l; l != nil {
                lineno, colno = l.caculateLocationLineColumn(m.commitLoc)
                s = l.scope
        }
        return
}

func (m *Module) GetSources(ctx *Context) (sources []string) {
        sources = split(ctx.callWith(m.commitLoc, m, "sources"))
        for i := range sources {
                if sources[i][0] == '/' { continue }
                sources[i] = filepath.Join(m.GetDir(), sources[i])
        }
        return
}

func (m *Module) createActionIfNil(ctx *Context) bool {
        s, lineno, colno := m.GetCommitLocation()

        numUsing := len(m.Using)
        for _, u := range m.Using {
                if u.createActionIfNil(ctx) { numUsing-- }
        }
        if 0 < numUsing {
                if *flagV || *flagVV {
                        fmt.Printf("%v:%v:%v: not all dependencies was built (%v, %v/%v)\n", s, lineno, colno, m.Name)
                } else {
                        fmt.Printf("%v:%v:%v: not all dependencies was built\n", s, lineno, colno)
                }
                return false
        }

        if m.Action != nil {
                return true
        }

        if m.Toolset == nil {
                fmt.Printf("%v:%v:%v: no such toolset (%v)\n", s, lineno, colno, m.Name)
                return false
        }

        if *flagVV {
                fmt.Printf("smart: config `%v' (%v)\n", m.Name, m.GetDir())
        }

        prev := ctx.m
        ctx.m = m

        if m.Toolset.CreateActions(ctx) {
                // ...
        } else if *flagV {
                fmt.Printf("%v:%v:%v: `%v' not built\n", s, lineno, colno, m.Name)
        }

        ctx.m = prev
        return m.Action != nil
}

func (m *Module) update() {
        if m.Action == nil {
                if *flagV {
                        s, lineno, colno := m.GetCommitLocation()
                        fmt.Printf("%v:%v:%v:warning: no action (\"%v\")\n", s, lineno, colno, m.Name)
                }
                return
        }

        if updated, _ := m.Action.update(); !updated {
                if *flagV {
                        s, lineno, colno := m.GetCommitLocation()
                        fmt.Printf("%v:%v:%v:warning: `%v' - nothing updated\n", s, lineno, colno, m.Name)

                        s, lineno, colno = m.GetDeclareLocation()
                        fmt.Printf("%v:%v:%v:info: `%v'\n", s, lineno, colno, m.Name)
                }
        }
}

type FileMatchRule struct {
        Name string
        Mode os.FileMode
        Rule string // *regexp.Regexp
}

func (r *FileMatchRule) match(fi os.FileInfo) bool {
        re := regexp.MustCompile(r.Rule)
        if fi.Mode() & r.Mode != 0 && re.MatchString(fi.Name()) {
                return true
        }
        return false
}

func (r *FileMatchRule) matchName(fn string) bool {
        re := regexp.MustCompile(r.Rule)
        if re.MatchString(fn) {
                return true
        }
        return false
}

func ReadDirNames(dirname string) ([]string, error) {
        return readDirNames(dirname)
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	fd, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}

	defer fd.Close()

	names, err := fd.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	sort.Strings(names)
	return names, nil
}

func FindFiles(d string, sre string) ([]string, error) {
        return findFiles(d, sre)
}

func FindFile(d string, sre string) (string) {
        return findFile(d, sre)
}

func Traverse(d string, fun traverseFunc) (error) {
        return traverse(d, fun)
}

func CopyFile(s, d string) (err error) {
        return copyFile(s, d)
}

type traverseFunc func(dname string, fi os.FileInfo) bool
func traverse(d string, fun traverseFunc) (err error) {
        names, err := readDirNames(d)
        if err != nil {
                //errorf("readDirNames: %v, %v\n", err, d)
                return
        }

        var fi os.FileInfo
        for _, name := range names {
                dname := filepath.Join(d, name)
                //fmt.Printf("traverse: %s\n", dname)

                fi, err = os.Stat(dname)
                if err != nil {
                        //errorf("stat: %v\n", dname)
                        return
                }

                if !fun(dname, fi) {
                        continue
                }

                if fi.IsDir() {
                        if err = traverse(dname, fun); err != nil {
                                //errorf("traverse: %v\n", dname)
                                return
                        }
                        continue
                }
        }
        return
}

// findNFiles finds N files underneath the directory `d' recursively. It's going to
// find all files if N is less than 1.
func findNFiles(d string, sre string, num int) (files []string, err error) {
        re := regexp.MustCompile(sre)
        err = traverse(d, func(dname string, fi os.FileInfo) bool {
                if re.MatchString(dname) {
                        files = append(files, dname)
                        if num--; num == 0 { return false }
                }
                return true
        })
        return
}

// findFiles finds all files underneath the directory `d' recursively.
func findFiles(d string, sre string) (files []string, err error) {
        return findNFiles(d, sre, -1)
}

// findFile finds one file underneath the directory `d' recursively.
func findFile(d string, sre string) (file string) {
        if fs, err := findNFiles(d, sre, 1); err == nil {
                if 0 < len(fs) { file = fs[0] }
        }
        return
}

// copyFile copies a file from `s' to `d'.
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

func MatchFileInfo(fi os.FileInfo, rules []*FileMatchRule) *FileMatchRule {
        return matchFileInfo(fi, rules)
}
func MatchFileName(fn string, rules []*FileMatchRule) *FileMatchRule {
        return matchFileName(fn, rules)
}

// matchFileInfo finds a matched FileMatchRule with FileInfo.
func matchFileInfo(fi os.FileInfo, rules []*FileMatchRule) *FileMatchRule {
        for _, g := range rules {
                if g.match(fi) { return g }
        }
        return nil
}

// matchFileName finds a matched FileMatchRule with file-name.
func matchFileName(fn string, rules []*FileMatchRule) *FileMatchRule {
        for _, g := range rules {
                if g.matchName(fn) { return g }
        }
        return nil
}

// Build builds the project with specified variables and commands.
func Build(vars map[string]string, cmds []string) (ctx *Context) {
        defer func() {
                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("smart: %v\n", se.message)
                                os.Exit(-1)
                        } else {
                                panic(e)
                        }
                }
        }()

        var (
                d string
                err error
        )
        if d = *flagC; d == "" { d = "." }

        s := []byte{} // TODO: needs init script
        ctx, err = NewContext("init", s, vars)
        if err != nil {
                fmt.Printf("smart: %v\n", err)
                return
        }

        // Find and process modules.
        err = traverse(d, func(fn string, fi os.FileInfo) bool {
                fr := matchFileInfo(fi, generalMetaFiles)
                if *flagG && fr != nil { return false }
                if fi.Name() == ".smart" {
                        if err := ctx.include(fn); err != nil {
                                errorf("include: `%v', %v\n", fn, err)
                        }
                }
                return true
        })
        if err != nil {
                fmt.Printf("error: %v\n", err)
        }

        // Build the modules
        var i *pendedBuild
        for 0 < len(ctx.moduleBuildList) {
                i, ctx.moduleBuildList = &ctx.moduleBuildList[0], ctx.moduleBuildList[1:]
                if !i.m.createActionIfNil(i.p) {
                        errorf("module `%v' not built", i.m.Name)
                }
        }

        var updateMod, updateDeps func(mod *Module)
        updateMod = func(mod *Module) {
                updateDeps(mod)
                if !mod.Updated {
                        if *flagV {
                                fmt.Printf("smart: update `%v'...\n", mod.Name)
                        }
                        mod.update()
                        mod.Updated = true
                }
        }
        updateDeps = func(mod *Module) {
                for _, u := range mod.Using { updateMod(u) }
        }

        for _, m := range ctx.moduleOrderList { updateMod(m) }
        return
}