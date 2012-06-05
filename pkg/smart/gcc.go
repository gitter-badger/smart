package smart

import (
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
        if len(object.Depends) == 0 { return nil }
        return run("gcc", "-o", object.Name, "-c", object.Depends[0].Name)
}
