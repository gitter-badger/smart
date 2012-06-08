package smart

import (
        "errors"
        //"fmt"
        "strings"
        "path/filepath"
)

func init() {
        g := &gcc{}
        AddBuildTool(g)
}

type gcc struct {
        target *Target
}

func (gcc *gcc) Supports(dir, name string) bool {
        if strings.HasSuffix(name, ".c") {
                return true
        }
        if strings.HasSuffix(name, ".cpp") {
                return true
        }
        return false
}

func (gcc *gcc) AddFile(dir string, s *Target) {
        if gcc.target == nil {
                var name string
                if dir == "" {
                        name = filepath.Base(Top)
                } else {
                        name = filepath.Base(dir)
                }
                gcc.target = NewFileGoal(name)
        }
        gcc.target.AddIntermediateFile(s.Name+".o", s)
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
                // gcc -o out.o -c -combine a.c b.c
                args := []string{
                        "-r", "-o", object.Name,
                }
                for _, d := range object.Depends {
                        if !strings.HasSuffix(d.Name, ".o") {
                                return errors.New("unexpected file type: "+d.String())
                        }
                        args = append(args, d.Name)
                }
                return run("ld", args...)
        }
        return nil
}
