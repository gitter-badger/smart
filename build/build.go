//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "bytes"
        "errors"
        "fmt"
        "io"
        //"io/ioutil"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "runtime"
        "strings"
        "sort"
        "github.com/duzy/worker"
)

var (
        workdir, _ = os.Getwd()

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

        onRecipeExecutionFailure = func(err error, c *exec.Cmd) {
                /*
                if s, e := ioutil.ReadAll(c.Stderr); e != nil && 0 < len(s) {
                        fmt.Printf(os.Stderr, "%s", string(s))
                } else {
                        fmt.Printf(os.Stderr, "fail to execute `%v'", c.Path)
                } */
                fmt.Fprintf(os.Stderr, "error: `%v` %v\n", c.Path, c.Args)
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(-1)
        }
)

// toolset represents a toolchain like gcc and related utilities.
type toolset interface {
        // ConfigModule setup the current module being processed.
        // `args' and `vars' is passed in on the `$(module)' invocation.
        ConfigModule(p *Context, args Items, vars map[string]string)

        // CreateActions creates the current module's action graph
        CreateActions(p *Context) bool

        // UseModule lets a toolset decides how to use a module.
        UseModule(p *Context, o *Module) bool

        // getNamespace returns toolset namespace (internal)
        getNamespace() namespace
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

func (tt *BasicToolset) ConfigModule(ctx *Context, args Items, vars map[string]string) {
}

func (tt *BasicToolset) CreateActions(ctx *Context) bool {
        return false
}

func (tt *BasicToolset) UseModule(ctx *Context, o *Module) bool {
        return false
}

func (tt *BasicToolset) getNamespace() namespace { return nil }

func IsIA32Command(s string) bool {
        buf := new(bytes.Buffer)
        cmd := exec.Command("file", "-b", s)
        cmd.Stdout = buf
        if err := cmd.Run(); err != nil {
                message("error: %v", err)
        }
        //return strings.HasPrefix(buf.String(), "ELF 32-bit")
        return strings.Contains(buf.String(), "ELF 32-bit")
}

// A command executed by an action while updating a target.
type Command interface {
        Execute(targets []string, prerequisites []string) bool
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

                err := cmd.Run()
                if err == nil {
                        updated = true
                }

                if (*flagL /**flagV && *flagVV*/) || err != nil {
                        if err != nil { message("%v (%v)", err, c.path) }
                        if c.path != "" {
                                fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------------\n")
                                fmt.Fprintf(os.Stderr, "%v %v\n", c.path, strings.Join(args, " "))
                        }
                        so, se := c.stdout.String(), c.stderr.String()
                        if so != "" {
                                fmt.Fprintf(os.Stderr, "------------------------------------------------------------------------- stdout\n")
                                fmt.Fprintf(os.Stderr, "%v", so)
                                if !strings.HasSuffix(so, "\n") { fmt.Fprintf(os.Stderr, "\n") }
                        }
                        if se != "" {
                                fmt.Fprintf(os.Stderr, "------------------------------------------------------------------------- stderr\n")
                                fmt.Fprintf(os.Stderr, "%v", se)
                                if !strings.HasSuffix(se, "\n") { fmt.Fprintf(os.Stderr, "\n") }
                        }
                        fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------------\n")
                        if err != nil { errorf(`failed executing "%v"`, c.path) }
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
        Targets(prerequisites []*Action) (names []string, needsUpdate bool)
}

func ComputeInterTargets(d, sre string, prerequisites []*Action) (targets []string, outdates int, outdateMap map[int]int) {
        re := regexp.MustCompile(sre)
        outdateMap = map[int]int{}
        traverse(d, func(fn string, fi os.FileInfo) bool {
                if !re.MatchString(fn) { return true }
                i := len(targets)
                outdateMap[i] = 0
                targets = append(targets, fn)
                for _, p := range prerequisites {
                        if pc, ok := p.Command.(intercommand); ok {
                                if _, needsUpdate := pc.Targets(p.Prerequisites); needsUpdate {
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

func ComputeKnownInterTargets(targets []string, prerequisites []*Action) (outdates int, outdateMap map[int]int) {
        outdateMap = map[int]int{}
        for i, fn := range targets {
                fi, e := os.Stat(fn)
                if e != nil || fi == nil { // Target not existed.
                        outdateMap[i]++
                        continue
                }
                for _, p := range prerequisites {
                        if pc, ok := p.Command.(intercommand); ok {
                                if _, needsUpdate := pc.Targets(p.Prerequisites); needsUpdate {
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
        }
        return
}

// Action performs a command for updating targets
type Action struct {
        Targets []string
        Prerequisites []*Action
        Command Command
}

func (a *Action) update() (updated bool, updatedTargets []string) {
        var targets []string
        var targetsNeedUpdate bool
        var isIntercommand bool
        if a.Command != nil {
                if c, ok := a.Command.(intercommand); ok {
                        targets, targetsNeedUpdate = c.Targets(a.Prerequisites)
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
        prerequisites := []string{}
        for _, p := range a.Prerequisites {
                if u, pres := p.update(); u {
                        prerequisites = append(prerequisites, pres...)
                        updatedPreNum++
                } else if pc, ok := p.Command.(intercommand); ok {
                        pres, nu := pc.Targets(p.Prerequisites)
                        if nu { errorf("requiring updating %v for %v", pres, targets) }
                        prerequisites = append(prerequisites, pres...)
                } else {
                        prerequisites = append(prerequisites, p.Targets...)
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
                updated, updatedTargets = a.execute(targets, fis, prerequisites)
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

                //fmt.Printf("targets: %v, %v, %v, %v\n", targets, request, len(a.prerequisites), prerequisites)
                if 0 < len(request) {
                        updated, updatedTargets = a.execute(request, requestfis, prerequisites)
                }
        }

        return
}

func (a *Action) execute(targets []string, tarfis []os.FileInfo, prerequisites []string) (updated bool, updatedTargets []string) {
        if updated = a.Command.Execute(targets, prerequisites); updated {
                var targetsNeedUpdate bool
                if c, ok := a.Command.(intercommand); ok {
                        updatedTargets, targetsNeedUpdate = c.Targets(a.Prerequisites)
                        updated = !targetsNeedUpdate
                } else {
                        for _, t := range a.Targets {
                                if fi, e := os.Stat(t); e != nil || fi == nil {
                                        errorf("`%s' was not built", t)
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
                Prerequisites: pre,
        }
        return a
}

func NewAction(target string, c Command, pre ...*Action) *Action {
        return newAction(target, c, pre...)
}

func NewInterAction(target string, c intercommand, pre ...*Action) *Action {
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

type template struct {
        *namespaceEmbed
        name string
        declNodes []*node
        postNodes []*node
        post, commit *node
}

type templateToolset struct {
        *template
        BasicToolset
}

func (tt *templateToolset) getNamespace() namespace {
        return tt.template.namespaceEmbed
}

func (tt *templateToolset) ConfigModule(ctx *Context, args Items, vars map[string]string) {
        //fmt.Printf("todo: TemplateToolset.ConfigModule %v\n", args.Join(ctx, ","))
        //fmt.Printf("TemplateToolset.ConfigModule '%v'\n", ctx.m.GetName(ctx))
        for _, n := range tt.declNodes {
                if e := ctx.processNode(n); e != nil {
                        //errorf("%v", e)
                        break
                }
        }
}

func (tt *templateToolset) CreateActions(ctx *Context) bool {
        fmt.Printf("todo: TemplateToolset.CreateActions\n")
        return false
}

// Module is defined by a $(module) invocation in .smart script.
type Module struct {
        *namespaceEmbed
        Parent *Module // upper module
        Toolset toolset
        Action *Action // action for building this module
        Using, UsedBy []*Module
        Updated bool // marked as 'true' if module is updated
        Children map[string]*Module
        //defines map[string]*define
        //rules map[string]*rule
        declareLoc, commitLoc location // where does it defined and commit (could be nil)
        //x *Context // the context of the module
        l *lex // the lex scope where does it defined (could be nil)
}

func (m *Module) getNamespace(name string) (ns namespace) {
        if c, ok := m.Children[name]; ok && c != nil {
                ns = c
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

func (m *Module) Get(ctx *Context, name string) (s string) {
        if d, ok := m.defines[name]; ok && d != nil {
                s = d.value.Expand(ctx)
        }
        return
}

func (m *Module) GetName(ctx *Context) string { return m.Get(ctx, "name") }
func (m *Module) GetDir(ctx *Context) string { return m.Get(ctx, "dir") }

func (m *Module) GetSources(ctx *Context) (sources []string) {
        sources = Split(m.Get(ctx, "sources")) // Split(ctx.callWith(m.commitLoc, m, "sources"))
        for i := range sources {
                if filepath.IsAbs(sources[i]) { continue }
                sources[i] = filepath.Join(m.GetDir(ctx), sources[i])
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
                        fmt.Printf("%v:%v:%v: not all dependencies was built (%v, %v/%v)\n", s, lineno, colno, m.GetName(ctx))
                } else {
                        fmt.Printf("%v:%v:%v: not all dependencies was built\n", s, lineno, colno)
                }
                return false
        }

        if m.Action != nil {
                return true
        }

        if m.Toolset == nil {
                fmt.Printf("%v:%v:%v: nil toolset (%v)\n", s, lineno, colno, m.GetName(ctx))
                return false
        }

        if *flagVV {
                fmt.Printf("smart: config `%v' (%v)\n", m.GetName(ctx), m.GetDir(ctx))
        }

        prev := ctx.m
        ctx.m = m

        if m.Toolset.CreateActions(ctx) {
                //fmt.Printf("smart: created `%v' (%v)\n", m.Name, m.GetDir())
        } else if *flagV {
                fmt.Printf("%v:%v:%v: `%v' not built\n", s, lineno, colno, m.GetName(ctx))
        }

        ctx.m = prev
        return m.Action != nil
}

func (m *Module) update(ctx *Context) {
        if m.Action == nil {
                if *flagV {
                        s, lineno, colno := m.GetCommitLocation()
                        fmt.Printf("%v:%v:%v:warning: no action (\"%v\")\n", s, lineno, colno, m.GetName(ctx))
                }
                return
        }

        if updated, _ := m.Action.update(); !updated {
                if *flagV && *flagVV {
                        s, lineno, colno := m.GetCommitLocation()
                        fmt.Printf("%v:%v:%v:info: nothing updated (%v)\n", s, lineno, colno, m.GetName(ctx))
                }
                if *flagV && *flagVV {
                        s, lineno, colno := m.GetDeclareLocation()
                        fmt.Printf("%v:%v:%v:info: module `%v'\n", s, lineno, colno, m.GetName(ctx))
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

type checkRuleUpdater struct {
        checkRule *rule
}
func (c *checkRuleUpdater) check(ctx *Context, r *rule, m *match) bool {
        if c.checkRule != r {
                errorf("diverged check rule")
        }

        ec := &ruleExecuteContext{
                target: m.target, stem: m.stem,
        }

        for _, i := range c.checkRule.prerequisites {
                ec.prerequisites = append(ec.prerequisites, i)
        }

        //fmt.Printf("checkRuleUpdater.check: %v\n", m.target)
        if e := c.checkRule.execute(ctx, ec); e != nil {
                //fmt.Printf("checkRuleUpdater.check: %v\n", e)
                return true
        }

        return false
}
func (c *checkRuleUpdater) update(ctx *Context, r *rule, m *match) bool {
        //fmt.Printf("checkRuleUpdater.update: %v\n", m.target)
        if prev, ok := r.prev[m.target]; ok {
                return prev.update(ctx, m)
        }
        return false
}

type phonyTargetUpdater struct {
}
func (c *phonyTargetUpdater) check(ctx *Context, r *rule, m *match) bool {
        return r.ns.isPhonyTarget(ctx, m.target)
}
func (c *phonyTargetUpdater) update(ctx *Context, r *rule, m *match) bool {
        //fmt.Printf("phonyTargetUpdater.update: %v\n", m.target)

        err, matchedPrerequisites, _ := r.updatePrerequisites(ctx)
        if err != nil {
                fmt.Fprintf(os.Stderr, "%v\n", err)
                //os.Exit(-1)
                return false
        }

        needsExecute := true
        checkRules := r.ns.getRules(nodeRuleChecker, m.target)
        if 0 < len(checkRules) {
                for _, cr := range checkRules {
                        if needsExecute = cr.c.check(ctx, cr, m); needsExecute {
                                break
                        }
                }
        }

        if needsExecute {
                ec := r.newExecuteContext(ctx, m, matchedPrerequisites)

                //fmt.Printf("phonyTargetUpdater.update: %v\n", m.target)
                return r.execute(ctx, ec) == nil
        }
        return false
}

type defaultTargetUpdater struct {
}
func (c *defaultTargetUpdater) check(ctx *Context, r *rule, m *match) bool {
        if fi, err := os.Stat(m.target); err != nil {
                return true
        } else {
                if fi == nil {}
        }
        return false
}
func (c *defaultTargetUpdater) update(ctx *Context, r *rule, m *match) bool {
        //fmt.Printf("defaultTargetUpdater.update: %v\n", m.target)
        
        err, matchedPrerequisites, updatedPrerequisites := r.updatePrerequisites(ctx)
        if err != nil {
                fmt.Fprintf(os.Stderr, "%v\n", err)
                //os.Exit(-1)
                return false
        }
        
        // Check if we need to update the target
        if len(updatedPrerequisites) == 0 && !r.check(ctx, m) {
                return false
        }

        ec := r.newExecuteContext(ctx, m, matchedPrerequisites)
        
        //fmt.Printf("execute: %v\n", m.target)
        return r.execute(ctx, ec) == nil
}

type match struct {
        target string
        stem string
}

type matchrule struct {
        *match
        rule *rule 
}

func (r *rule) match(target string) (m *match, matched bool) {
        for _, t := range r.targets {
                if t == target {
                        matched, m = true, &match{
                                target: t,
                        }
                }
        }
        return
}

func (r *rule) findPrevRule(m *match) (prev *rule) {
        prev, _ = r.prev[m.target]
        return
}

func (r *rule) check(ctx *Context, m *match) (needsUpdate bool) {
        for !needsUpdate && r != nil {
                needsUpdate = r.c.check(ctx, r, m) || needsUpdate
                r = r.findPrevRule(m)
        }
        return needsUpdate
}

func (r *rule) update(ctx *Context, m *match) bool {
        return r.c.update(ctx, r, m)
}

func (r *rule) updateAll(ctx *Context) bool {
        var num = 0
        for _, t := range r.targets {
                m := &match{ target:t }
                if r.update(ctx, m) { num++ }
        }
        return 0 < num
}

func (r *rule) updatePrerequisites(ctx *Context) (err error, matchedPrerequisites, updatedPrerequisites []*matchrule) {
        for _, prerequisite := range r.prerequisites {
                if m, r := r.ns.findMatchedRule(ctx, prerequisite); m != nil && r != nil {
                        matchedPrerequisites = append(matchedPrerequisites, &matchrule{ m, r })
                } else {
                        err = errors.New(fmt.Sprintf("no rule to update '%v'\n", prerequisite))
                        return
                }
        }
        for _, mr := range matchedPrerequisites {
                if ok := mr.rule.update(ctx, mr.match); ok {
                        updatedPrerequisites = append(updatedPrerequisites, mr)
                }
        }
        return
}

func (r *rule) newExecuteContext(ctx *Context, m *match, matchedPrerequisites []*matchrule) *ruleExecuteContext {
        ec := &ruleExecuteContext{ target: m.target, stem: m.stem }

        for _, mr := range matchedPrerequisites {
                ec.prerequisites = append(ec.prerequisites, mr.target)
        }

        return ec
}

type ruleExecuteContext struct {
        target, stem string
        prerequisites []string
}

// https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html#Automatic-Variables
//   $@ The file name of the target of the rule. If the target is an archive member, then ‘$@’ is the name of the archive file. In a pattern rule that has multiple targets (see Introduction to Pattern Rules), ‘$@’ is the name of whichever target caused the rule’s recipe to be run.
//   $% The target member name, when the target is an archive member. See Archives. For example, if the target is foo.a(bar.o) then ‘$%’ is bar.o and ‘$@’ is foo.a. ‘$%’ is empty when the target is not an archive member.
//   $< The name of the first prerequisite. If the target got its recipe from an implicit rule, this will be the first prerequisite added by the implicit rule (see Implicit Rules).
//   $? The names of all the prerequisites that are newer than the target, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives).
//   $^ The names of all the prerequisites, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives). A target has only one prerequisite on each other file it depends on, no matter how many times each file is listed as a prerequisite. So if you list a prerequisite more than once for a target, the value of $^ contains just one copy of the name. This list does not contain any of the order-only prerequisites; for those see the ‘$|’ variable, below.
//   $+ This is like ‘$^’, but prerequisites listed more than once are duplicated in the order they were listed in the makefile. This is primarily useful for use in linking commands where it is meaningful to repeat library file names in a particular order.
//   $| The names of all the order-only prerequisites, with spaces between them.
//   $* The stem with which an implicit rule matches (see How Patterns Match). If the target is dir/a.foo.b and the target pattern is a.%.b then the stem is dir/foo. The stem is useful for constructing names of related files.
//      In a static pattern rule, the stem is part of the file name that matched the ‘%’ in the target pattern.
//      In an explicit rule, there is no stem; so ‘$*’ cannot be determined in that way. Instead, if the target name ends with a recognized suffix (see Old-Fashioned Suffix Rules), ‘$*’ is set to the target name minus the suffix. For example, if the target name is ‘foo.c’, then ‘$*’ is set to ‘foo’, since ‘.c’ is a suffix. GNU make does this bizarre thing only for compatibility with other implementations of make. You should generally avoid using ‘$*’ except in implicit rules or static pattern rules.
//      If the target name in an explicit rule does not end with a recognized suffix, ‘$*’ is set to the empty string for that rule.
//      
//   $(@D) $(@F) $(*D) $(*F) $(%D) $(%F) $(<D) $(<F) $(^D) $(^F) $(+D) $(+F) $(?D) $(?F)
//   
func (r *rule) execute(ctx *Context, ec *ruleExecuteContext) error {
        ns := ctx.g
        
        saveIndex, _ := ns.saveDefines(
                "@", "@D", "@F",
                "%", "%D", "%F",
                "<", "<D", "<F",
                "?", "?D", "?F",
                "^", "^D", "^F",
                "+", "+D", "+F",
                "|", "|D", "|F",
                "*", "*D", "*F")
        defer ns.restoreDefines(saveIndex)

        ns.Set(ctx, []string{ "@" },  stringitem(ec.target))
        ns.Set(ctx, []string{ "@D" }, stringitem(filepath.Dir(ec.target)))
        ns.Set(ctx, []string{ "@F" }, stringitem(filepath.Base(ec.target)))
        ns.Set(ctx, []string{ "*" },  stringitem(ec.stem))
        ns.Set(ctx, []string{ "*D" }, stringitem(filepath.Dir(ec.stem)))
        ns.Set(ctx, []string{ "*F" }, stringitem(filepath.Base(ec.stem)))

        var l, ld, lf []Item
        for n, prerequisite := range ec.prerequisites {
                d, f := filepath.Dir(prerequisite), filepath.Base(prerequisite)
                if n == 0 {
                        ns.Set(ctx, []string{ "<" },  stringitem(prerequisite))
                        ns.Set(ctx, []string{ "<D" }, stringitem(d))
                        ns.Set(ctx, []string{ "<F" }, stringitem(f))
                }
                l = append(l, stringitem(prerequisite))
                ld = append(ld, stringitem(d))
                lf = append(lf, stringitem(f))
        }
        ns.Set(ctx, []string{ "^" }, l...)
        ns.Set(ctx, []string{ "^D" }, ld...)
        ns.Set(ctx, []string{ "^F" }, lf...)

        job := new(executeRecipes)
        for _, action := range r.recipes {
                var s string
                switch a := action.(type) {
                case string: s = a
                case *node: s = a.Expand(ctx)
                }
                job.recipes = append(job.recipes, s)
        }
        /*
        if *flagJ <= 1 {
                job.Action()
        } else {
                ctx.w.Do(job)
        } */
        job.Action()
        return job.error
}

type executeRecipes struct {
        recipes []string
        error error
}
func (job *executeRecipes) Action() worker.Result {
        for _, s := range job.recipes {
                echo := true
                if s[0] == '@' {
                        s, echo = s[1:], false
                }
                if cmd := exec.Command("sh", "-c", s); cmd != nil {
                        if echo {
                                fmt.Printf("%v\n", s)
                                cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
                        }
                        if job.error = cmd.Run(); job.error != nil {
                                break
                        } else {
                                //fmt.Printf("rule: %v\n", s)
                        }
                } else {
                        errorf("nil command `sh`")
                }
        }
        return nil
}

func UpdateModules(ctx *Context, cmds ...string) {
        // Build the modules
        var i *pendedBuild
        for 0 < len(ctx.moduleBuildList) {
                i, ctx.moduleBuildList = &ctx.moduleBuildList[0], ctx.moduleBuildList[1:]
                if !i.m.createActionIfNil(i.p) {
                        errorf("nil action (%v)", i.m.GetName(ctx))
                }
        }

        var (
                updateMod, updateDeps func(m *Module)
        )
        updateMod = func(m *Module) {
                updateDeps(m)
                if !m.Updated {
                        if *flagV {
                                fmt.Printf("smart: update `%v'...\n", m.GetName(ctx))
                        }
                        m.update(ctx)
                        m.Updated = true
                }
        }
        updateDeps = func(m *Module) {
                for _, u := range m.Using { updateMod(u) }
        }

        for _, m := range ctx.moduleOrderList { updateMod(m) }
}

// Update updates the specified targets given in `cmds`.
//
// Example:
//      
//      # Updates global target 'foo.txt'
//      smart -g foo.txt
//      
//      # Updates module foo's target 'bar.txt'
//      smart foo:bar.txt
//      
//      # Updates module 'foobar'
//      smart -m foobar
//      
//      # Updates both module and global 'foobar'
//      smart foobar
// 
func Update(ctx *Context, cmds ...string) {
        ctx.w.SpawnN(*flagJ); defer ctx.w.KillAll()

        if n := len(cmds); n == 0 {
                if ctx.g.goal == "" {
                        UpdateModules(ctx, cmds...)
                } else if g, ok := ctx.g.rules[ctx.g.goal]; ok && g != nil {
                        g.updateAll(ctx)
                }
                return
        }

        for _, cmd := range cmds {
                for _, r := range ctx.g.rules {
                        if m, ok := r.match(cmd); ok {
                                r.update(ctx, m)
                        }
                }
        }
}

// Build builds the project with specified variables and commands.
func Build(vars map[string]string, cmds ...string) (ctx *Context) {
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
                if *flagGG && fr != nil { return false }
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

        Update(ctx, cmds...)
        return
}
