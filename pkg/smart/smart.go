package smart

import (
        "bufio"
        "fmt"
        "io"
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

var actions map[string]Action
var regComment = regexp.MustCompile(`^\s*//`)
var regMeta = regexp.MustCompile(`^\s*//\s*#smart\s+`)
var regCall = regexp.MustCompile(`([a-z_\-]+)\s*\(\s*(([^"]|"(\\"|[^"])")+?)\s*\)\s*;?`)
var regArg = regexp.MustCompile(`\s*(([^,"]|"(\\"|[^"])")*)(,|\s*$)`)

func init() {
        actions = map[string]Action {
                "use": ACTION_USE,
                "combine": ACTION_COMBINE,
        }
}

type MetaInfo struct {
        Action Action
        Args []string
}

func (mi *MetaInfo) String() string {
        return fmt.Sprintf("%v(%v)", mi.Action, strings.Join(mi.Args, ","))
}

type Target struct {
        Name string
        Depends []*Target
        IsFile bool
        IsIntermediate bool
        IsGoal bool
        IsScanned bool
        Meta []*MetaInfo
}

func (t *Target) String() string {
        return t.Name
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

func (t *Target) AddIntermediate(name string, source *Target) *Target {
        i := t.Add(name)
        i.IsIntermediate = true
        if source != nil {
                i.Depends = append(i.Depends, source)
        }
        return i
}

func (t *Target) AddIntermediateFile(name string, source interface{}) *Target {
        if source == nil {
                i := t.AddIntermediate(name, nil)
                i.IsFile = true
                return i
        }
        switch s := source.(type) {
        case *Target:
                i := t.AddIntermediate(name, s)
                i.IsFile = true
                return i
        case string:
                i := t.AddIntermediate(name, nil)
                i.IsFile = true
                i.AddFile(s)
                return i
        }
        return nil
}

func New(name string) (t *Target) {
        t = new(Target)
        t.Name = name
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

type Collector interface {
        AddFile(dir, name string) *Target
        AddDir(dir string) *Target
}

type BuildTool interface {
        Collector
        SetTop(top string)
        Build() error
}

func AddBuildTool(tool BuildTool) {
}

func meta(name string) (info []*MetaInfo) {
        f, e := os.Open(name)
        if e != nil {
                return
        }

        defer f.Close()

        lineno := 0
        b := bufio.NewReader(f)
outfor: for {
                var line []byte
                for {
                        l, isPrefix, e := b.ReadLine()
                        if e == io.EOF { break outfor }
                        if l[0] != '/' { break outfor }
                        
                        line = append(line, l...)
                        if !isPrefix { break }
                }

                lineno += 1

                if !regComment.Match(line) { break }

                if loc := regMeta.FindIndex(line); loc != nil {
                        line = line[loc[1]:]
                        //fmt.Printf("%s:%d:%d:TODO: %v (%v)\n", name, lineno, loc[1], string(line), loc)
                }

                if ma := regCall.FindAllSubmatch(line, -1); ma != nil {
                        for _, m := range ma {
                                //fmt.Printf("%s:%d:TODO: (%d) '%v' '%v'\n", name, lineno, len(m), string(m[1]), string(m[2]))
                                fn, action, ok := string(m[1]), ACTION_NOP, false
                                if action, ok = actions[fn]; !ok {
                                        continue
                                }

                                mi := &MetaInfo{ Action:action }
                                if aa := regArg.FindAllSubmatch(m[2], -1); aa != nil {
                                        for _, a := range aa {
                                                //fmt.Printf("%s:%d:TODO: %v '%v'\n", name, lineno, fn, string(a[1]))
                                                mi.Args = append(mi.Args, string(a[1]))
                                        }
                                }
                                info = append(info, mi)
                        }
                }
        }
        return
}

// scan scans source files under the current working directory and
// add file names to the specified build tool.
func scan(coll Collector, top, dir string) (e error) {
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
                                        coll.AddDir(dname)
                                        continue
                                }
                        }

                        if s := coll.AddFile(sd, name); s != nil {
                                s.IsScanned = true
                                s.Meta = meta(dname)
                                if s.Meta != nil {
                                        fmt.Printf("TODO: %s: %v\n", dname, s.Meta)
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

        if e = read(dir); e != nil && e != io.EOF {
                return
        }

        e = nil
        return
}

// generate calls gen in goroutines on each target. If any error occurs,
// it will be returned with the updated targets.
func generate(targets []*Target, gen func(*Target) error) (error, []*Target) {
        if len(targets) == 0 {
                return nil, nil
        }

        type meta struct { t *Target; e error }
        ch := make(chan meta)

        g := func(t *Target) {
                var err error
                needGen := true

                if 0 < len(t.Depends) {
                        if e, u := generate(t.Depends, gen); e == nil {
                                //fmt.Printf("%v: %v, %v\n", t, t.Depends, u)
                                needGen = needGen || 0 < len(u)
                        } else {
                                needGen, err = false, e
                        }
                }

                if t.IsFile {
                        switch {
                        case t.IsScanned:
                                needGen = false
                        case t.IsIntermediate:
                                if _, e := os.Stat(t.Name); e != nil {
                                        needGen = true
                                }
                        }
                }

                if needGen {
                        err = gen(t)
                }

                ch <- meta{ t, err }
        }

        gn := len(targets)

        for _, t := range targets {
                go g(t)
        }

        updated := make([]*Target, gn)

        for ; 0 < gn; gn -= 1 {
                if m := <-ch; m.e == nil {
                        updated = append(updated, m.t)
                } else {
                        fmt.Printf("%v\n", m.e)
                }
        }

        return nil, updated
}

// run executes the command specified by cmd with arguments by args.
func run(cmd string, args ...string) error {
        fmt.Printf("%s\n", cmd + " " + strings.Join(args, " "))
        p := exec.Command(cmd, args...)
        p.Stdout = os.Stdout
        p.Stderr = os.Stderr
        p.Start()
        return p.Wait()
}

func Build() error {
        // TODO...
        return nil
}
