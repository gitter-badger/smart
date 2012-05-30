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
        //target []*Target
        //objects []*Target
        target *Target
}

/*
func (gcc *gcc) NewTarget(dir, name string) (t *Target) {
        t = new(Target)
        t.Name = name
        t.IsFile = true
        return
}
*/

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
        t := gcc.target
        if t == nil {
                t = new(Target)
                t.IsFile = true
                if dir == "" {
                        t.Name = filepath.Base(Top)
                } else {
                        t.Name = filepath.Base(dir)
                }
                gcc.target = t
        }

        o := new(Target)
        o.Depends = append(o.Depends, s)
        o.Name = s.Name + ".o"

        t.Depends = append(t.Depends, o)

        //fmt.Printf("add: %v, %v\n", t, o)
}

func (gcc *gcc) Build() error {
        t := gcc.target

        gen := func(object *Target) error {
                return gcc.generate(object)
        }
        done := func(object *Target) {
                //fmt.Printf("compiled: %v\n", object)
        }

        if e := generate(t.Depends, gen, done); e != nil {
                return e
        }

        args := []string{ "-o", t.Name, }

        for _, t := range t.Depends {
                args = append(args, t.Name)
        }

        return run("gcc", args...)
}

func (gcc *gcc) generate(object *Target) error {
        return run("gcc", "-o", object.Name, "-c", object.Depends[0].Name)
}
