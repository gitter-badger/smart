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
        "time"
)

// Alias of os.FileInfo, allow client to use it without import os package.
type FileInfo os.FileInfo

const (
        ACTION_NOP Action = iota
        ACTION_USE
        ACTION_COMBINE
)

var actionNames = []string{
        ACTION_NOP: "nop",
        ACTION_USE: "use",
        ACTION_COMBINE: "combine",
}

type Action int

func (a Action) String() string {
        if a < 0 || len(actionNames) <= int(a) {
                return ""
        }
        return actionNames[a]
}

var _regComment = regexp.MustCompile(`^\s*//`)
var _regMeta = regexp.MustCompile(`^\s*//\s*#smart\s+`)
var _regCall = regexp.MustCompile(`([a-z_\-]+)\s*\(\s*(([^"]|"(\\"|[^"])")+?)\s*\)\s*;?`)
var _regArg = regexp.MustCompile(`\s*(([^,"]|"(\\"|[^"])")*)(,|\s*$)`)
var targets map[string]*Target
var actions map[string]Action

const (
        Separator = string(filepath.Separator)
)

func init() {
        targets = make(map[string]*Target)
        actions = map[string]Action {
                "nop": ACTION_NOP,
                "use": ACTION_USE,
                "combine": ACTION_COMBINE,
        }
}

// All returns all the targets.
func All() map[string]*Target {
        return targets
}

// Reset reset the target map.
func Reset() map[string]*Target {
        old := targets
        targets = make(map[string]*Target)
        return old
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
        return fmt.Sprintf("%v%v", mi.Action, mi.Args)
}

type NamedValues struct {
        Name string
        Values []string
}

type Class int

func (c Class) String() string {
        var s1, s2 string
        switch {
        case c & Intermediate != 0: s1 = "Intermediate"
        case c & Final != 0: s1 = "Final"
        }
        switch {
        case c & File != 0: s2 = "File"
        case c & Dir != 0: s2 = "Dir"
        }
        return fmt.Sprintf("%s%s", s1, s2)
}

const (
        None Class = 0

        Intermediate = 1<<1 // generated target of depends 
        Final = 1<<2 // project goal target

        File = 1<<3
        Dir = 1<<4

        IntermediateFile = Intermediate | File
        IntermediateDir = Intermediate | Dir

        FinalFile = Final | File
        FinalDir = Final | Dir
)

type Target struct {
        Type string
        Name string

        Dependees []*Target // the targets this target depends on
        Dependers []*Target // the targets depends on this target
        Usees []*Target // the targets used by this target
        Users []*Target // the targets use this target

        Class Class

        IsScanned bool // target is made by scan() or find()
        IsDirTarget bool // target is made by AddDir

        Generated bool // target has already generated -- tool.Generate performed

        Meta []*MetaInfo
        Args []*NamedValues
        Exports []*NamedValues
        Variables map[string]string
}

func (t *Target) String() string {
        return t.Name
}

func (t *Target) Stat() FileInfo {
        if t.IsFile() || t.IsDir() {
                fi, _ := os.Stat(t.Name)
                return fi
        }
        return nil
}

func (t *Target) Touch() bool {
        if fi := t.Stat(); fi != nil {
                at := time.Now()
                mt := time.Now()
                if e := os.Chtimes(t.Name, at, mt); e == nil { // utime
                        return true
                }
        }
        return false
}

func (t *Target) IsDir() bool {// directory target
        return t.Class & (Dir | ^File) != 0
}

func (t *Target) IsFile() bool {// file (non-dir) target
        return t.Class & (File | ^Dir) != 0
}

func (t *Target) IsIntermediate() bool {// opposite to 'goal'
        return t.Class & (Intermediate | ^Final) != 0
}

func (t *Target) IsFinal() bool {// final target, opposite to 'intermediate'
        return t.Class & (Final | ^Intermediate) != 0
}

func (t *Target) SetVar(name, value string) {
        t.Variables[name] = value
}

func (t *Target) DelVar(name string) {
        delete(t.Variables, name)
}

func (t *Target) VarDef(name string, def string) string {
        if v, ok := t.Variables[name]; ok {
                return v
        }
        return def
}

func (t *Target) Var(name string) string {
        return t.VarDef(name, "")
}

func (t *Target) add(l []*NamedValues, name string, args ...string) ([]*NamedValues, *NamedValues) {
        for _, nv := range l {
                if nv.Name == name {
                        nv.Values = append(nv.Values, args...)
                        return l, nv
                }
        }

        nv := &NamedValues{ name, args }
        l = append(l, nv)
        return l, nv
}

func (t *Target) join(l []*NamedValues, n string) (res []string) {
        for _, nv := range l {
                if nv.Name == n {
                        for _, s := range nv.Values {
                                res = append(res, n+s)
                        }
                }
        }
        return
}

func (t *Target) joinAll(l []*NamedValues) (res []string) {
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

// 
func (t *Target) AddArgs(name string, args ...string) (nv *NamedValues) {
        t.Args, nv = t.add(t.Args, name, args...)
        return nv
}

// 
func (t *Target) AddExports(name string, args ...string) (nv *NamedValues) {
        t.Exports, nv = t.add(t.Exports, name, args...)
        return nv
}

// 
func (t *Target) JoinAllArgs() (res []string) {
        return t.joinAll(t.Args)
}

// 
func (t *Target) JoinAllExports() (res []string) {
        return t.joinAll(t.Exports)
}

// 
func (t *Target) JoinArgs(n string) []string {
        return t.join(t.Args, n)
}

// 
func (t *Target) JoinExports(n string) []string {
        return t.join(t.Exports, n)
}

// JoinUseesArgs
func (t *Target) JoinUseesArgs(n string) (res []string) {
        for _, u := range t.Usees {
                res = append(res, u.JoinExports(n)...)
        }
        return
}

// JoinDependersUseesArgs
func (t *Target) JoinDependersUseesArgs(n string) (a []string) {
        for _, der := range t.Dependers {
                a = append(a, der.JoinUseesArgs(n)...)
        }
        return
}

// Use add target to the usee list.
func (t *Target) Use(usee *Target) {
        for _, u := range t.Usees {
                if usee == u {
                        goto out
                }
        }
        t.Usees = append(t.Usees, usee)
        usee.Users = append(usee.Users, t)
out:
}

// Dep add target to the depend list.
func (t *Target) Dep(i interface {}, class Class) (o *Target) {
        switch d := i.(type) {
        case string:
                o = New(d, class)
        case *Target:
                o = d
        }

        if o != nil {
                t.Dependees = append(t.Dependees, o)
                o.Dependers = append(o.Dependers, t)
        }
        return
}

// New create new target
func New(name string, class Class) (t *Target) {
        t = new(Target)
        t.Name = name
        t.Variables = make(map[string]string)
        t.Class = class
        targets[name] = t
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

// NewError makes a new error object.
func NewError(what string) error {
        return &err{ what }
}

// NewErrorf makes a new error object.
func NewErrorf(what string, args ...interface{}) error {
        return &err{ fmt.Sprintf(what, args...) }
}

// Info reports an informative message.
func Info(f string, a ...interface{}) {
        f = strings.TrimRight(f, " \t\n") + "\n"
        fmt.Fprintf(os.Stdout, f, a...)
}

// Warn reports a warning message.
func Warn(f string, a ...interface{}) {
        f = "warn: " + strings.TrimRight(f, " \t\n") + "\n"
        fmt.Fprintf(os.Stderr, f, a...)
}

// Fatal reports a fatal error and quit all.
func Fatal(f string, a ...interface{}) {
        f = strings.TrimRight(f, " \t\n") + "\n"
        fmt.Fprintf(os.Stderr, f, a...)
        os.Exit(-1)
}

// IsFile checks if the name on a filesystem is a file.
func IsFile(name string) bool {
        if fi, e := os.Stat(name); e == nil && fi != nil {
                return fi.Mode() & os.ModeType == 0
        }
        return false
}

// IsDir checks if the name on a filesystem is a directory.
func IsDir(name string) bool {
        if fi, e := os.Stat(name); e == nil && fi != nil {
                return fi.IsDir()
        }
        return false
}

// ReadFile reads a file lay in the filesystem.
func ReadFile(fn string) []byte {
        if f, e := os.Open(fn); e == nil {
                defer f.Close()
                if b, e := ioutil.ReadAll(f); e == nil {
                        return b
                }
        }
        return nil
}

// Copy copies a file lay in the filesystem.
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

// ForEachLine iterates the file content line by line.
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

type traverseCallback func(depth int, dname string, fi FileInfo) bool

// traverse iterate each name under a directory recursively.
func traverse(depth int, d string, fun traverseCallback) (err error) {
        fd, err := os.Open(d)
        if err != nil {
                return
        }

        defer fd.Close()

        var fi FileInfo
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

// Find file via regexp.
func Find(d string, sreg string, coll Collector) error {
        re, e := regexp.Compile(sreg)
        if e != nil {
                return e
        }
        return traverse(0, d, func(depth int, dname string, fi FileInfo) bool {
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

// Graph draws dependency graph of targets.
func Graph() {
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
                } else if t.IsFinal() {
                        goals = append(goals, t)
                } else {
                        files = append(files, t)
                }
        next:
        }
}

// Command make system command.
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

type genmeta struct {
        t *Target
        g bool // tool.Generate invoked
        e error // error returned by tool.Generate
}

// generate invoke tool.Generate on targets
func generate(tool BuildTool, caller *Target, targets []*Target) (error, []*Target) {
        if len(targets) == 0 {
                return nil, nil
        }

        ch := make(chan genmeta)
        gn := len(targets)

        for _, t := range targets {
                if !t.Generated {
                        go gen(tool, caller, t, ch)
                }
        }
        
        updated := []*Target{}

        for ; 0 < gn; gn -= 1 {
                if m := <-ch; m.g && m.e == nil {
                        updated = append(updated, m.t)
                } else if m.e != nil {
                        Fatal("error: %v\n", m.e)
                }
        }

        return nil, updated
}

// gen invoke tool.Generate on a single target
func gen(tool BuildTool, caller *Target, t *Target, ch chan genmeta) {
        var err error
        var needGen = false

        if t.Generated {
                ch <- genmeta{ t, needGen, err }
                return
        }

        // check depends
        if 0 < len(t.Dependees) {
                if e, u := generate(tool, t, t.Dependees); e == nil {
                        //Info("updated: %v", u)
                        needGen = needGen || 0 < len(u)
                } else {
                        needGen, err = needGen || false, e
                }
        }

        // check for update
        fi := t.Stat()
        if fi == nil { // not an file target or file not exists
                needGen = needGen || true
        } else if caller != nil {
                // compare modification time with caller-target
                fiCaller := caller.Stat()
                if fiCaller == nil {
                        ///Info("caller: no %v", caller)
                } else if fi.ModTime().After(fiCaller.ModTime()) {
                        ///Info("%v: %v (%v)", caller, t, fi.ModTime().Sub(fiCaller.ModTime()))
                        needGen = needGen || true
                }
        }

        // invoke tool.Generate
        if needGen {
                if err = tool.Generate(t); err == nil {
                        t.Generated = true
                }
        }

        ch <- genmeta{ t, needGen, err }
}


// Generate calls tool.Generate in goroutines on each target. If any error occurs,
// it will be returned with the updated targets.
func Generate(tool BuildTool, targets []*Target) (error, []*Target) {
        return generate(tool, nil, targets)
}

// scanTargetGraph scan filesystem for targets and draw target graph
func scanTargetGraph(tool BuildTool) (e error) {
        var top string

        if top, e = os.Getwd(); e != nil {
                return
        }

        tool.SetTop(top)

        if e = Scan(tool.NewCollector(nil), top, top); e != nil {
                return
        }

        Graph() // draw dependency graph.
        return nil
}

// Build launches a build process on a tool.
func Build(tool BuildTool) (e error) {
        if e = scanTargetGraph(tool); e != nil {
                return
        }

        goals := tool.Goals()

        for _, g := range goals {
                if !g.IsFinal() {
                        e = NewErrorf("goal `%v' is not final", g)
                        return
                }
        }

        if e, _ = Generate(tool, goals); e != nil {
                return
        }

        return
}

func clean(ts []*Target) error {
        for _, t := range ts {
                if e := clean(t.Dependees); e != nil {
                        return e
                }

                if t.Stat() == nil { continue }

                if (t.IsIntermediate() || t.IsFinal()) && !t.IsScanned {
                        Info("remove %v", t)
                        if e := os.Remove(t.Name); e != nil {
                                return e
                        }
                }
        }
        return nil
}

// Clean performs the clean command line
func Clean(tool BuildTool) (e error) {
        if e = scanTargetGraph(tool); e != nil {
                return
        }

        var ts []*Target
        for _, t := range targets { ts = append(ts, t) }
        return clean(ts)
}

// CommandLine
func CommandLine(commands map[string] func(args []string) error, args []string) {
        cmd := args[0]
        args = args[1:]

        if proc, ok := commands[cmd]; ok && proc != nil {
                if e := proc(args); e != nil {
                        Fatal("smart: %v (%v %v)\n", e, cmd, args)
                        os.Exit(-1)
                }
        } else {
                Fatal("smart: '%v' not supported (%v)\n", cmd, args)
                os.Exit(-1)
        }
}
