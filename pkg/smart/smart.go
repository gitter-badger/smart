package smart

import (
        "fmt"
        "io"
        "os"
        "os/exec"
        "path/filepath"
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

var Top string

const (
        TargetFlagFile int = 1
        TargetFlagBuilt = 2
        TargetFlagScanned = 4
)

type Target struct {
        Name string
        Depends []*Target
        IsFile bool
        IsIntermediate bool
        IsGoal bool
        IsScanned bool
}

func (t *Target) String() string {
        return t.Name
}

func (t *Target) Add(name string) *Target {
        f := new(Target)
        f.Name = name
        t.Depends = append(t.Depends, f)
        return f
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

func (t *Target) AddIntermediateFile(name string, source *Target) *Target {
        i := t.AddIntermediate(name, source)
        i.IsFile = true
        return i
}

func NewFileGoal(name string) (t *Target) {
        t = new(Target)
        t.IsFile = true
        t.IsGoal = true
        t.Name = name
        return
}

type BuildTool interface {
        Supports(dir, name string) bool
        AddFile(dir string, f *Target)
        Build() error
}

func AddBuildTool(tool BuildTool) {
}

// build scans source files under the current working directory and
// add file names to the specified build tool and then launch it's
// build method.
func build(tool BuildTool) (e error) {
        if Top == "" {
                if Top, e = os.Getwd(); e != nil {
                        return
                }
        }

        add := func(sd string, names []string) {
                for _, name := range names {
                        if strings.HasPrefix(name, ".") {
                                continue
                        }
                        if strings.HasSuffix(name, "~") {
                                continue
                        }

                        if tool.Supports(sd, name) {
                                s := new(Target)
                                s.Name = filepath.Join(sd, name)
                                s.IsFile = true
                                s.IsScanned = true
                                tool.AddFile(sd, s)
                        }
                }
        }

        read := func(d string) error {
                var sd string
                if strings.HasPrefix(d, Top) {
                        sd = d[len(Top):]
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

        if e = read(Top); e != nil && e != io.EOF {
                return
        }

        return tool.Build()
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
                                needGen = needGen || 0 < len(u)
                        } else {
                                needGen, err = false, e
                        }
                }

                if needGen {
                        err = gen(t)
                }

                ch <- meta{ t, err }
        }

        gn := len(targets)

        for _, t := range targets {
                if t.IsFile {
                        switch {
                        case t.IsScanned:
                                gn -= 1
                                continue
                        case t.IsIntermediate:
                                // TODO: Check existence of the target
                        }
                }

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
