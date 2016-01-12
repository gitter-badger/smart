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

        generalMetaFiles = []*filerule{
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
        // configModule setup the current module being processed by parser.
        // `args' and `vars' is passed in on the `$(module)' invocation.
        configModule(p *context, args []string, vars map[string]string) bool

        // createActions builds the action graph the current module
        createActions(p *context, args []string) bool

        // useModule
        useModule(p *context, m *module) bool
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

func isIA32Command(s string) bool {
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
type command interface {
        execute(targets []string, prequisites []string) bool
}

type excmd struct {
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

func (c *excmd) run(targetHint string, args ...string) bool {
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
                        errorf(0, "can't update `%v'", targetHint)
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
                                fmt.Printf("%v\n", so)
                        }
                        if se != "" {
                                fmt.Printf("------------------------------ stderr\n")
                                fmt.Printf("%v\n", se)
                        }
                        errorf(0, `failed executing "%v"`, c.path)
                }

                if c.postcall != nil {
                        c.postcall(c.stdout, c.stderr)
                }
        }

        return updated
}

// intercommand represents a intermdiate action command
type intercommand interface {
        command
        targets(prequisites []*action) (names []string, needsUpdate bool)
}

func computeInterTargets(d, sre string, prequisites []*action) (targets []string, outdates int, outdateMap map[int]int) {
        re := regexp.MustCompile(sre)
        outdateMap = map[int]int{}
        traverse(d, func(fn string, fi os.FileInfo) bool {
                if !re.MatchString(fn) { return true }
                i := len(targets)
                outdateMap[i] = 0
                targets = append(targets, fn)
                for _, p := range prequisites {
                        if pc, ok := p.command.(intercommand); ok {
                                if _, needsUpdate := pc.targets(p.prequisites); needsUpdate {
                                        outdateMap[i]++
                                }
                        } else {
                                for _, t := range p.targets {
                                        if pfi, _ := os.Stat(t); pfi == nil {
                                                errorf(0, "`%v' not found", t)
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

// action performs a command for updating targets
type action struct {
        targets []string
        prequisites []*action
        command command
}

func (a *action) update() (updated bool, updatedTargets []string) {
        var targets []string
        var targetsNeedUpdate bool
        var isIntercommand bool
        if a.command != nil {
                if c, ok := a.command.(intercommand); ok {
                        targets, targetsNeedUpdate = c.targets(a.prequisites)
                        isIntercommand = true
                }
        }

        if !isIntercommand {
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
                panic("internal unmatched arrays") //errorf(-1, "internal")
        }

        updatedPreNum := 0
        prequisites := []string{}
        for _, p := range a.prequisites {
                if u, pres := p.update(); u {
                        prequisites = append(prequisites, pres...)
                        updatedPreNum++
                } else if pc, ok := p.command.(intercommand); ok {
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

func (a *action) force(targets []string, tarfis []os.FileInfo, prequisites []string) (updated bool, updatedTargets []string) {
        updated = a.command.execute(targets, prequisites)

        if updated {
                var targetsNeedUpdate bool
                if c, ok := a.command.(intercommand); ok {
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

func createSourceTransformActions(sources []string, namecommand func(src string) (string, command)) []*action {
        var inters []*action
        if namecommand == nil {
                errorf(-1, "can't draw source rules (%v)", namecommand)
        }

        for _, src := range sources {
                aname, c := namecommand(src)
                if aname == "" { continue }
                if aname == src {
                        errorf(-1, "no intermediate name for `%v'", src)
                }

                if c == nil {
                        errorf(0, "no command for `%v'", src)
                }

                asrc := newAction(src, nil)
                a := newAction(aname, c, asrc)
                inters = append(inters, a)
        }
        return inters
}

// module is defined by a $(module) invocation in .smart script.
type module struct {
        location *location // where does it defined
        dir string
        name string
        toolset toolset
        kind string
        action *action // action for building this module
        variables map[string]*variable
        using, usedBy []*module
        built, updated bool // marked as 'true' if module is built or updated
}

func (m *module) update() {
        //fmt.Printf("update: module: %v\n", m.name)

        if m.action == nil {
                fmt.Printf("%v: no action for module \"%v\"\n", m.location, m.name)
                return
        }

        if updated, _ := m.action.update(); !updated {
                fmt.Printf("smart: noting done for `%v'\n", m.name)
        }
}

type pendedBuild struct {
        m *module
        p *context
        args []string
}

var modules = map[string]*module{}
var moduleOrderList []*module
var moduleBuildList []pendedBuild

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

type traverseCallback func(dname string, fi os.FileInfo) bool
func traverse(d string, fun traverseCallback) (err error) {
        names, err := readDirNames(d)
        if err != nil {
                //errorf(0, "readDirNames: %v, %v\n", err, d)
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

// findNumFiles finds `num' files underneath the directory `d' recursively. It's going to
// find all files if `num' is less than 1.
func findNumFiles(d string, sre string, num int) (files []string, err error) {
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
        return findNumFiles(d, sre, -1)
}

// findFile finds one file underneath the directory `d' recursively.
func findFile(d string, sre string) (file string) {
        if fs, err := findNumFiles(d, sre, 1); err == nil {
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

// matchFileInfo finds a matched filerule with FileInfo.
func matchFileInfo(fi os.FileInfo, rules []*filerule) *filerule {
        for _, g := range rules {
                if g.match(fi) { return g }
        }
        return nil
}

// matchFileName finds a matched filerule with file-name.
func matchFileName(fn string, rules []*filerule) *filerule {
        for _, g := range rules {
                if g.matchName(fn) { return g }
        }
        return nil
}

// Build builds the project with specified variables and commands.
func Build(vars map[string]string, cmds []string) {
        defer func() {
                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("smart:%v: %v\n", se.number, se)
                                os.Exit(-1)
                        } else {
                                panic(e)
                        }
                }
        }()

        var d string
        if d = *flagC; d == "" { d = "." }

        // Find and process modules.
        err := traverse(d, func(fn string, fi os.FileInfo) bool {
                fr := matchFileInfo(fi, generalMetaFiles)
                if *flagG && fr != nil { return false }
                if fi.Name() == ".smart" {
                        if _, err := parse(fn); err != nil { errorf(0, "parse: `%v', %v\n", fn, err) }
                }
                return true
        })
        if err != nil {
                fmt.Printf("error: %v\n", err)
        }

        // Build the modules
        var buildDeps func(p *context, mod *module) int
        var buildMod func(p *context, mod *module) bool
        buildMod = func(p *context, mod *module) bool {
                if buildDeps(p, mod) != len(mod.using) {
                        fmt.Printf("%v: failed building deps of `%v' (by `%v')\n", mod.location, mod.name, mod.name)
                        return false
                }
                if mod.toolset == nil {
                        //fmt.Printf("%v: no toolset for `%v'(by `%v')\n", mod.location, mod.name, mod.name)
                        fmt.Printf("%v: no toolset for `%v'\n", mod.location, mod.name)
                        return false
                }
                if !mod.built {
                        if *flagVV {
                                fmt.Printf("smart: build `%v' (%v)\n", mod.name, mod.dir)
                        }
                        p.module = mod
                        if mod.toolset.createActions(p, []string{}) {
                                mod.built = true
                        } else {
                                fmt.Printf("%v: module `%v' not built\n", mod.location, mod.name)
                        }
                }
                return mod.built
        }
        buildDeps = func(p *context, mod *module) (num int) {
                for _, u := range mod.using { if buildMod(p, u) { num++ } }
                return
        }

        var i *pendedBuild
        for 0 < len(moduleBuildList) {
                i, moduleBuildList = &moduleBuildList[0], moduleBuildList[1:]
                if !buildMod(i.p, i.m) {
                        errorf(0, "module `%v' not built", i.m.name)
                }
        }

        var updateMod, updateDeps func(mod *module)
        updateMod = func(mod *module) {
                updateDeps(mod)
                if !mod.updated {
                        fmt.Printf("smart: update `%v'...\n", mod.name)
                        mod.update()
                        mod.updated = true
                }
        }
        updateDeps = func(mod *module) {
                for _, u := range mod.using { updateMod(u) }
        }

        for _, m := range moduleOrderList { updateMod(m) }
}
