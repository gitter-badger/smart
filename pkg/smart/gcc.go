package smart

import (
        "errors"
        "fmt"
        "strings"
        "path/filepath"
)

func init() {
        g := &gcc{}
        AddBuildTool(g)
}

type gcc struct {
        top string
        target *Target
}

type gccColl struct {
        gcc *gcc
        target *Target
}

func (gcc *gcc) SetTop(top string) {
        gcc.top = top
}

func (gcc *gcc) checkSuffix(name string) bool {
        switch {
        case strings.HasSuffix(name, ".c"): fallthrough
        case strings.HasSuffix(name, ".cpp"):
                return true
        }
        return false
}

func (gcc *gcc) ensureTarget(dir string) bool {
        if gcc.target == nil {
                var name string
                if dir == "" {
                        name = filepath.Base(gcc.top)
                } else {
                        name = filepath.Base(dir)
                }
                gcc.target = NewFileGoal(name)
        }
        return gcc.target != nil
}

func (gcc *gcc) AddDir(dir string) (t *Target) {
        if !gcc.ensureTarget(dir) {
                // TODO: error report
                return nil
        }

        switch {
        case strings.HasSuffix(dir, ".o"):
                t = NewFileGoal(filepath.Join(dir, "_.o"))
        case strings.HasSuffix(dir, ".a"):
                fallthrough
        case strings.HasSuffix(dir, ".so"):
                name := filepath.Base(dir)
                if !strings.HasPrefix(dir, "lib") {
                        name = "lib"+name
                }
                t = NewFileGoal(filepath.Join(dir, name))
        }

        scan(&gccColl{ gcc, t }, gcc.top, dir)
        //fmt.Printf("scan: %v %v\n", dir, t.Depends)

        t = gcc.target.Add(t)

        //fmt.Printf("TODO: AddDir: %v\n", t)
        return t
}

func (gcc *gcc) AddFile(dir, name string) *Target {
        if !gcc.checkSuffix(name) {
                // TODO: error report
                return nil
        }

        if !gcc.ensureTarget(dir) {
                // TODO: error report
                return nil
        }

        o := gcc.target.AddIntermediateFile(name+".o", name)
        if o == nil {
                // TODO: error report
                return nil
        }
        return o.Depends[0]
}

func (gcc *gcc) Build() error {
        t := gcc.target
        if t == nil {
                return nil
        }

        gen := func(object *Target) error {
                return gcc.generate(object)
        }

        if e, _ := generate(t.Depends, gen); e != nil {
                return e
        }

        args := []string{ "-o", t.Name, }

        for _, t := range t.Depends {
                args = append(args, t.Name)
        }

        return run("gcc", args...)
}

func (gcc *gcc) generate(object *Target) error {
        dl := len(object.Depends)
        switch (dl) {
        case 0: break
        case 1:
                d0 := object.Depends[0]
                switch {
                case strings.HasSuffix(d0.Name, ".o"):
                        // TODO: error
                case strings.HasSuffix(d0.Name, ".cc"): fallthrough
                case strings.HasSuffix(d0.Name, ".cpp"): fallthrough
                case strings.HasSuffix(d0.Name, ".cxx"): fallthrough
                case strings.HasSuffix(d0.Name, ".C"):
                        return run("g++", "-o", object.Name, "-c", d0.Name)
                case strings.HasSuffix(d0.Name, ".c"):
                        return run("gcc", "-o", object.Name, "-c", d0.Name)
                }
        default:
                var deps []string
                for _, d := range object.Depends {
                        if !strings.HasSuffix(d.Name, ".o") {
                                return errors.New("unexpected file type: "+d.String())
                        }
                        deps = append(deps, d.Name)
                }
                return run("ld", append([]string{ "-r", "-o", object.Name, }, deps...)...)
        }
        return nil
}

func (coll *gccColl) AddDir(dir string) *Target {
        fmt.Printf("TODO: gccColl.AddDir: %v\n", dir)
        return nil
}

func (coll *gccColl) AddFile(dir, name string) *Target {
        if !coll.gcc.checkSuffix(name) {
                // TODO: error report
                return nil
        }

        name = filepath.Join(dir, name)

        o := coll.target.AddIntermediateFile(name+".o", name)
        if o == nil {
                // TODO: error report
                return nil
        }
        return o.Depends[0]
}
