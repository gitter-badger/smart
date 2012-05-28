package smart

import (
        //"fmt"
        "strings"
        //"path/filepath"
)

func init() {
        g := &gcc{}
        AddBuildTool(g)
}

type gcc struct {
        //target []*Target
        //objects []*Target
}

func (gcc *gcc) NewTarget(dir, name string) (t *Target) {
        t = new(Target)
        t.Name = name
        t.IsFile = true
        return
}

func (gcc *gcc) AddFile(t *Target, s *Target) {
        if !strings.HasSuffix(s.Name, ".c") {
                return
        }

        o := new(Target)
        o.Depends = append(o.Depends, s)
        o.Name = s.Name + ".o"

        t.Depends = append(t.Depends, o)

        //fmt.Printf("add: %v, %v\n", t, o)
}

func (gcc *gcc) Build(t *Target) error {
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
