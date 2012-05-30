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
        IsBuilt bool
        IsScanned bool
}

func (t *Target) String() string {
        return t.Name
}

type BuildTool interface {
        //NewTarget(dir, name string) (t *Target)
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

// generate calls gen in goroutines on each target, and call cb if any
// such goroutine finished. If any error occurs, it will be returned.
func generate(targets []*Target, gen func(*Target) error, cb func(*Target)) error {
        type meta struct { t *Target; e error }
        ch := make(chan meta)

        var hit func(t *Target)
        hit = func(t *Target) {
                /*
                for _, d := range t.Depends {
                        hit(d)
                }
                */
                ch <- meta{ t, gen(t) }
        }

        for _, t := range targets {
                go hit(t)
        }

        for n := 0; n < len(targets); n += 1 {
                m := <-ch
                if m.e != nil {
                        fmt.Printf("%v\n", m.e)
                } else if cb != nil {
                        cb(m.t)
                }
        }

        return nil
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
