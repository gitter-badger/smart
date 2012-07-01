package smart

import (
        "bufio"
        "fmt"
        "io"
        "io/ioutil"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "strings"
/*
        "bytes"
        "flag"
        "os"
        "regexp"
        "runtime"
        "sort"
        */
)

/*
const (
        TargetFlagFile int = 1
        TargetFlagBuilt = 2
        TargetFlagScanned = 4
)
*/

const (
        ACTION_NOP Action = iota
        ACTION_USE
        ACTION_COMBINE
)

type Action int

func (a Action) String() string {
        switch (a) {
        case ACTION_NOP: return "nop"
        case ACTION_USE: return "use"
        case ACTION_COMBINE: return "combine"
        }
        return ""
}

var _regComment = regexp.MustCompile(`^\s*//`)
var _regMeta = regexp.MustCompile(`^\s*//\s*#smart\s+`)
var _regCall = regexp.MustCompile(`([a-z_\-]+)\s*\(\s*(([^"]|"(\\"|[^"])")+?)\s*\)\s*;?`)
var _regArg = regexp.MustCompile(`\s*(([^,"]|"(\\"|[^"])")*)(,|\s*$)`)
var targets map[string]*Target
var actions map[string]Action

func init() {
        targets = make(map[string]*Target)
        actions = map[string]Action {
                "nop": ACTION_NOP,
                "use": ACTION_USE,
                "combine": ACTION_COMBINE,
        }
}

func ResetTargets() {
        targets = make(map[string]*Target)
}

func All() map[string]*Target {
        return targets
}

// T maps name to the coresponding target.
func T(name string) *Target {
        if t, ok := targets[name]; ok {
                return t
        }
        return nil
}

type MetaInfo struct {
        Action Action
        Args []string
        Lineno int
}

func (mi *MetaInfo) String() string {
        //return fmt.Sprintf("%v(%v)", mi.Action, strings.Join(mi.Args, ","))
        return fmt.Sprintf("%v%v", mi.Action, mi.Args)
}

type NameValues struct {
        Name string
        Values []string
}

type Target struct {
        Type string
        Name string
        Depends []*Target
        Usees []*Target
        ParentUsees []*Target
        IsFile bool // file (non-dir) target
        IsDir bool // directory target
        IsIntermediate bool // opposite to 'goal'
        IsGoal bool // final target, opposite to 'intermediate'
        IsScanned bool // target is made by scan() or find()
        IsDirTarget bool // target is made by AddDir
        IsGenerated bool
        Meta []*MetaInfo
        Args []*NameValues
        Exports []*NameValues
        Variables map[string]string
}

func (t *Target) String() string {
        return t.Name
}

func (t *Target) add(l []*NameValues, name string, args ...string) ([]*NameValues, *NameValues) {
        for _, nv := range l {
                if nv.Name == name {
                        nv.Values = append(nv.Values, args...)
                        return l, nv
                }
        }

        nv := &NameValues{ name, args }
        l = append(l, nv)
        return l, nv
}

func (t *Target) join(l []*NameValues, n string) (res []string) {
        for _, nv := range l {
                if nv.Name == n {
                        for _, s := range nv.Values {
                                res = append(res, n+s)
                        }
                }
        }
        return
}

func (t *Target) joinAll(l []*NameValues) (res []string) {
        for _, nv := range l {
                if len(nv.Values) == 0 {
                        res = append(res, nv.Name)
                        continue
                }

                for _, s := range nv.Values {
                        res = append(res, nv.Name+s)
                }
        }
        return
}

func (t *Target) AddArgs(name string, args ...string) (nv *NameValues) {
        t.Args, nv = t.add(t.Args, name, args...)
        return nv
}

func (t *Target) AddExports(name string, args ...string) (nv *NameValues) {
        t.Exports, nv = t.add(t.Exports, name, args...)
        return nv
}

func (t *Target) JoinAllArgs() (res []string) {
        return t.joinAll(t.Args)
}

func (t *Target) JoinAllExports() (res []string) {
        return t.joinAll(t.Exports)
}

func (t *Target) JoinArgs(n string) []string {
        return t.join(t.Args, n)
}

func (t *Target) JoinExports(n string) []string {
        return t.join(t.Exports, n)
}

func (t *Target) joinUseesArgs(usees []*Target, n string) (res []string) {
        for _, u := range usees {
                res = append(res, u.JoinExports(n)...)
        }
        return
}

func (t *Target) JoinUseesArgs(n string) []string {
        return t.joinUseesArgs(t.Usees, n)
}

func (t *Target) JoinParentUseesArgs(n string) []string {
        return t.joinUseesArgs(t.ParentUsees, n)
}

func (t *Target) Use(usee *Target) {
        for _, u := range t.Usees {
                if usee == u {
                        goto out
                }
        }
        t.Usees = append(t.Usees, usee)
out:
}

func (t *Target) Add(i interface {}) (f *Target) {
        if i != nil {
                switch d := i.(type) {
                case string:
                        f = New(d)
                case *Target:
                        f = d
                }
                if f != nil {
                        t.Depends = append(t.Depends, f)
                }
        }
        return
}

func (t *Target) AddFile(name string) *Target {
        f := t.Add(name)
        f.IsFile = true
        return f
}

func (t *Target) AddDir(name string) *Target {
        f := t.Add(name)
        f.IsDir = true
        return f
}

func (t *Target) AddIntermediate(name string, source interface{}) *Target {
        i := t.Add(name)
        i.IsIntermediate = true
        i.Add(source)
        return i
}

func (t *Target) AddIntermediateFile(name string, source interface{}) *Target {
        i := t.AddIntermediate(name, nil)
        i.IsFile = true
        switch s := source.(type) {
        case *Target: i.Add(source)
        case string: i.AddFile(s)
        }
        return i
}

func (t *Target) AddIntermediateDir(name string, source interface{}) *Target {
        i := t.AddIntermediateFile(name, source)
        i.IsFile = false
        i.IsDir = true
        return i
}

func New(name string) (t *Target) {
        t = new(Target)
        t.Name = name
        t.Variables = make(map[string]string)
        targets[name] = t
        return
}

func NewIntermediate(name string) (t *Target) {
        t = New(name)
        t.IsIntermediate = true
        return
}

func NewGoal(name string) (t *Target) {
        t = New(name)
        t.IsGoal = true
        return
}

func NewFile(name string) (t *Target) {
        t = New(name)
        t.IsFile = true
        return
}

func NewFileGoal(name string) (t *Target) {
        t = NewFile(name)
        t.IsGoal = true
        return
}

func NewFileIntermediate(name string) (t *Target) {
        t = NewFile(name)
        t.IsIntermediate = true
        return
}

func NewDir(name string) (t *Target) {
        t = New(name)
        t.IsDir = true
        return
}

func NewDirGoal(name string) (t *Target) {
        t = NewDir(name)
        t.IsGoal = true
        return
}

func NewDirIntermediate(name string) (t *Target) {
        t = NewDir(name)
        t.IsIntermediate = true
        return
}

type Collector interface {
        AddFile(dir, name string) *Target
        AddDir(dir string) *Target
}

type BuildTool interface {
        NewCollector(t *Target) Collector
        SetTop(top string)
        Generate(t *Target) error
        Goals() []*Target
}

type err struct {
        what string
}

func (e *err) Error() string {
        return e.what
}

func NewError(what string) error {
        return &err{ what }
}

func NewErrorf(what string, args ...interface{}) error {
        return &err{ fmt.Sprintf(what, args...) }
}

func Info(f string, a ...interface{}) {
        f = strings.TrimRight(f, " \t\n") + "\n"
        fmt.Fprintf(os.Stdout, f, a...)
}

func Warn(f string, a ...interface{}) {
        f = "warn: " + strings.TrimRight(f, " \t\n") + "\n"
        fmt.Fprintf(os.Stderr, f, a...)
}

func Fatal(f string, a ...interface{}) {
        f = strings.TrimRight(f, " \t\n") + "\n"
        fmt.Fprintf(os.Stderr, f, a...)
        os.Exit(-1)
}

func IsFile(name string) bool {
        if fi, e := os.Stat(name); e == nil && fi != nil {
                return fi.Mode() & os.ModeType == 0
        }
        return false
}

func IsDir(name string) bool {
        if fi, e := os.Stat(name); e == nil && fi != nil {
                return fi.IsDir()
        }
        return false
}

func ReadFile(fn string) []byte {
        if f, e := os.Open(fn); e == nil {
                defer f.Close()
                if b, e := ioutil.ReadAll(f); e == nil {
                        return b
                }
        }
        return nil
}

func CopyFile(s, d string) (err error) {
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

func ForEachLine(filename string, fun func(lineno int, line []byte) bool) error {
        f, e := os.Open(filename)
        if e != nil {
                return e
        }

        defer f.Close()

        lineno := 0
        b := bufio.NewReader(f)
outfor: for {
                var line []byte
        readfor:for {
                        l, isPrefix, e := b.ReadLine()
                        switch {
                        case e == io.EOF: break outfor
                        case e != nil: return e
                        default:
                                line = append(line, l...)
                                if !isPrefix { break readfor }
                        }
                }

                lineno += 1

                if !fun(lineno, line) {
                        break
                }
        }

        return nil
}

func meta(name string) (info []*MetaInfo) {
        ForEachLine(name, func(lineno int, line []byte) bool {
                if !_regComment.Match(line) { return false }

                if loc := _regMeta.FindIndex(line); loc != nil {
                        line = line[loc[1]:]
                        //fmt.Printf("%s:%d:%d:TODO: %v (%v)\n", name, lineno, loc[1], string(line), loc)
                }

                if ma := _regCall.FindAllSubmatch(line, -1); ma != nil {
                        for _, m := range ma {
                                //fmt.Printf("%s:%d:TODO: (%d) '%v' '%v'\n", name, lineno, len(m), string(m[1]), string(m[2]))
                                fn, action, ok := string(m[1]), ACTION_NOP, false
                                if action, ok = actions[fn]; !ok {
                                        continue
                                }

                                mi := &MetaInfo{
                                        Action:action,
                                        Lineno:lineno,
                                }

                                if aa := _regArg.FindAllSubmatch(m[2], -1); aa != nil {
                                        for _, a := range aa {
                                                //fmt.Printf("%s:%d:TODO: %v '%v'\n", name, lineno, fn, string(a[1]))
                                                mi.Args = append(mi.Args, string(a[1]))
                                        }
                                }

                                info = append(info, mi)
                        }
                }
                return true
        })
        return
}

// scan scans source files under the current working directory and
// add file names to the specified build tool.
func Scan(coll Collector, top, dir string) (e error) {
        add := func(sd string, names []string) {
                for _, name := range names {
                        if strings.HasPrefix(name, ".") {
                                continue
                        }
                        if strings.HasSuffix(name, "~") {
                                continue
                        }

                        dname := filepath.Join(sd, name)

                        if fi, e := os.Stat(dname); fi == nil || e != nil {
                                fmt.Printf("error: %v\n", e); continue
                        } else {
                                if fi.IsDir() {
                                        if s := coll.AddDir(dname); s != nil {
                                                s.IsDirTarget = true
                                                targets[dname] = s
                                        }
                                } else if s := coll.AddFile(sd, name); s != nil {
                                        s.IsScanned = true
                                        s.Meta = meta(dname)
                                        //fmt.Printf("meta: %s: %v\n", dname, s.Meta)
                                        targets[dname] = s
                                }
                        }
                }
        }

        read := func(d string) error {
                var sd string
                if strings.HasPrefix(d, top) {
                        sd = d[len(top):]
                } else {
                        sd = d
                }

                fd, e := os.Open(d)
                if e != nil {
                        return e
                }

                var names []string
                for {
                        names, e = fd.Readdirnames(50)
                        if 0 < len(names) {
                                add(sd, names)
                        }
                        if e != nil {
                                return e
                        }
                }

                return nil
        }

        if e = read(dir); e == io.EOF {
                e = nil
        }

        return
}

type traverseCallback func(depth int, dname string, fi os.FileInfo) bool

// traverse iterate each name under a directory recursively.
func traverse(depth int, d string, fun traverseCallback) (err error) {
        fd, err := os.Open(d)
        if err != nil {
                return
        }

        defer fd.Close()

        var fi os.FileInfo
readloop:
        for {
                names, e := fd.Readdirnames(50)
                switch {
                case e == io.EOF: break readloop
                case e != nil: err = e; break readloop
                }

        nameloop:
                for _, name := range names {
                        dname := filepath.Join(d, name)
                        if fi, err = os.Stat(dname); err != nil || fi == nil {
                                return
                        }

                        if !fun(depth, dname, fi) {
                                break readloop
                        }

                        if fi.IsDir() {
                                if err = traverse(depth+1, dname, fun); err != nil {
                                        return
                                }
                                continue nameloop
                        }
                }
        }
        return
}

// find 
func Find(d string, sre string, coll Collector) error {
        re, e := regexp.Compile(sre)
        if e != nil {
                return e
        }
        return traverse(0, d, func(depth int, dname string, fi os.FileInfo) bool {
                if re.MatchString(dname) {
                        if !fi.IsDir() {
                                t := coll.AddFile(filepath.Dir(dname), filepath.Base(dname))
                                if t != nil {
                                        t.IsScanned = true
                                }
                        }
                }
                return true
        })
}

// graph draws dependency graph of targets.
func Graph() {
        //fmt.Printf("scanned: %v\n", targets)

        var dirs []*Target
        var files []*Target
        var goals []*Target
        for _, t := range targets {
                //fmt.Printf("scan: %v -> %v (%v, %v)\n", k, t, t.IsFile, t.IsDirTarget)
                if t.IsDirTarget {
                        for _, d := range dirs {
                                if d == t { goto next }
                        }
                        dirs = append(dirs, t)
                } else if t.IsGoal {
                        goals = append(goals, t)
                } else {
                        files = append(files, t)
                }
        next:
        }

        /*
        fmt.Printf("dirs: %v\n", dirs)
        fmt.Printf("files: %v\n", files)
        fmt.Printf("goals: %v\n", goals)
        */

        var apply func(ts []*Target)
        apply = func(ts []*Target) {
                for _, t := range ts {
                        apply(t.Depends)
                        for _, d := range t.Depends {
                                d.ParentUsees = append(d.ParentUsees, t.Usees...)
                        }
                }
        }
        apply(goals)
}

// 
func Command(name string, args ...string) *exec.Cmd {
        p := exec.Command(name, args...)
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        return p
}

// Run executes the command specified by cmd with arguments by args.
func Run(cmd string, args ...string) error {
        return RunInDir(cmd, "", args...)
}

func RunInDir(cmd, dir string, args ...string) error {
        fmt.Printf("%s\n", cmd + " " + strings.Join(args, " "))
        p := Command(cmd, args...)
        if dir != "" { p.Dir = dir }
        p.Start()
        return p.Wait()
}

func Run32(cmd string, args ...string) error {
        return Run32InDir(cmd, "", args...)
}

func Run32InDir(cmd, dir string, args ...string) error {
        fmt.Printf("%s\n", filepath.Base(cmd) + " " + strings.Join(args, " "))
        args = append([]string{ cmd }, args...)
        p := Command("linux32", args...)
        if dir != "" { p.Dir = dir }
        p.Start()
        return p.Wait()
}

// Generate calls tool.Generate in goroutines on each target. If any error occurs,
// it will be returned with the updated targets.
func Generate(tool BuildTool, targets []*Target) (error, []*Target) {
        if len(targets) == 0 {
                return nil, nil
        }

        type meta struct { t *Target; e error }
        ch := make(chan meta)

        gen := func(t *Target) {
                //fmt.Printf("gen: %v (%v)\n", t, t.IsGenerated)

                if t.IsGenerated {
                        ch <- meta{ t, nil }
                        return
                }

                var err error
                var needGen = false

                if 0 < len(t.Depends) {
                        if e, u := Generate(tool, t.Depends); e == nil {
                                needGen = needGen || 0 < len(u)
                        } else {
                                needGen, err = needGen || false, e
                        }
                }

                if t.IsFile || t.IsDir {
                        switch {
                        case t.IsScanned:
                                needGen = needGen || false
                        default:
                                if _, e := os.Stat(t.Name); e != nil {
                                        needGen = needGen || true
                                } else {
                                        needGen = needGen || false
                                }
                        }
                }

                //fmt.Printf("gen: %v (%v, %v)\n", t, t.IsGenerated, needGen)

                if needGen {
                        if err = tool.Generate(t); err == nil {
                                t.IsGenerated = true
                        }
                }

                ch <- meta{ t, err }
        }

        gn := len(targets)

        for _, t := range targets {
                if !t.IsGenerated {
                        go gen(t)
                }
        }

        updated := make([]*Target, gn)

        for ; 0 < gn; gn -= 1 {
                if m := <-ch; m.e == nil {
                        updated = append(updated, m.t)
                } else {
                        Fatal("error: %v\n", m.e)
                }
        }

        return nil, updated
}

// Build launches a build process on a tool.
func Build(tool BuildTool) (e error) {
        var top string

        if top, e = os.Getwd(); e != nil {
                return
        }

        tool.SetTop(top)

        if e = Scan(tool.NewCollector(nil), top, top); e != nil {
                return
        }

        Graph() // draw dependency graph.

        if e, _ = Generate(tool, tool.Goals()); e != nil {
                return
        }

        return
}
