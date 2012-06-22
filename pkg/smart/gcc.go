package smart

import (
        "fmt"
        "os"
        "path/filepath"
        "strings"
)

func init() {
        // ...
}

type gcc struct {
        top string
        target *Target
}

func (gcc *gcc) SetTop(top string) {
        gcc.top = top
}

func (gcc *gcc) Build() error {
        t := gcc.target
        if t == nil {
                return nil
        }

        if e, _ := Generate(gcc, []*Target{ t }); e != nil {
                return e
        }

        return nil
}

func (gcc *gcc) NewCollector(t *Target) Collector {
        return &gccCollector{ gcc, t }
}

func (gcc *gcc) Generate(t *Target) error {
        switch {
        case strings.HasSuffix(t.Name, ".o"):
                return gcc.compile(t)
        case strings.HasSuffix(t.Name, ".a"):
                return gcc.archive(t)
        }
        return gcc.link(t)
}

func (gcc *gcc) compile(t *Target) error {
        dl := len(t.Depends)
        switch (dl) {
        case 0: return NewErrorf("no depends: %v\n", t)
        case 1:
                cc, d0 := "", t.Depends[0]
                if s, ok := t.Properties["CC"]; ok && s != "" {
                        cc = s
                } else {
                        return NewErrorf("unknown file type: %v", d0.Name)
                }

                args := []string{ "-o", t.Name, }
                args = append(args, t.JoinAllArgs()...)
                args = append(args, "-c", d0.Name)
                return run(cc, args...)

        default:
                var deps []string
                for _, d := range t.Depends {
                        if !strings.HasSuffix(d.Name, ".o") {
                                return NewErrorf("unexpected file type: %v", d)
                        }
                        deps = append(deps, d.Name)
                }
                return run("ld", append([]string{ "-r", "-o", t.Name, }, deps...)...)
        }

        return nil
}

func (gcc *gcc) archive(t *Target) error {
        //fmt.Printf("archive: %v\n", t)
        
        ar := "ar"
        args := []string{ "crs", t.Name, }

        al := len(args)
        for _, d := range t.Depends {
                switch d.Type {
                case ".o":
                        args = append(args, d.String())
                default:
                        fmt.Printf("ar: ignored: %v\n", d)
                }
        }

        if len(args) - al <= 0 {
                return NewErrorf("no objects for archive: %v", t)
        }

        if s, ok := t.Properties["AR"]; ok && s != "" {
                ar = s
        }

        return run(ar, args...)
}

func (gcc *gcc) link(t *Target) error {
        //fmt.Printf("link: %v\n", t)

        ld := "ld" // the default linker is 'ld'
        args := []string{ "-o", t.Name, }

        if strings.HasSuffix(t.Name, ".so") {
                args = append(args, "-shared")
        }

        for _, d := range t.Depends {
                switch d.Type {
                case ".a": fallthrough
                case ".so":
                        args = append(args, d.JoinExports("-L")...)
                        args = append(args, d.JoinExports("-l")...)
                case ".o":
                        args = append(args, d.Name)
                default:
                        fmt.Printf("link: ignored: %v\n", d)
                }
        }

        args = append(args, t.JoinArgs("-L")...)
        args = append(args, t.JoinArgs("-l")...)

        if s, ok := t.Properties["LD"]; ok && s != "" {
                ld = s
        }

        return run(ld, args...)
}

type gccCollector struct {
        gcc *gcc
        target *Target
}

func (coll *gccCollector) ensureTarget(dir string) bool {
        if coll.target == nil {
                var name string
                if dir == "" {
                        name = filepath.Base(coll.gcc.top)
                } else {
                        name = filepath.Base(dir)
                }

                coll.target = NewFileGoal(name)

                if coll.gcc.target == nil {
                        coll.gcc.target = coll.target
                }
        }
        return coll.target != nil
}

func (coll *gccCollector) AddDir(dir string) (t *Target) {
        if !coll.ensureTarget("") {
                fmt.Fprintf(os.Stderr, "no goal in %v\n", dir)
                return nil
        }

        switch {
        case strings.HasSuffix(dir, ".o"):
                t = NewFileGoal(filepath.Join(dir, "_.o"))
                t.Type = ".o"
        case strings.HasSuffix(dir, ".a"): fallthrough
        case strings.HasSuffix(dir, ".so"):
                name := filepath.Base(dir)
                ext := filepath.Ext(name)
                if !strings.HasPrefix(dir, "lib") {
                        name = "lib"+name
                }

                t = NewFileGoal(filepath.Join(dir, name))
                t.Type = ext

                l := len(name) - len(ext)
                t.AddExports("-L", dir)
                t.AddExports("-l", name[3:l])

                coll.target.AddArgs("-I", dir)

                //fmt.Printf("TODO: -L%v -l%v\n", dir, name[3:l])
        }

        if t != nil {
                scan(coll.gcc.NewCollector(t), coll.gcc.top, dir)
                //fmt.Printf("scan: %v %v\n", dir, t.Depends)

                coll.target.Add(t)
                //fmt.Printf("TODO: AddDir: %v %v\n", t, t.Depends)
        }
        return t
}

func (coll *gccCollector) AddFile(dir, name string) *Target {
        if !coll.ensureTarget(dir) {
                fmt.Fprintf(os.Stderr, "no goal in %v\n", dir)
                return nil
        }

        cc := ""
        switch {
        default: return nil
        case strings.HasSuffix(name, ".cc"): fallthrough
        case strings.HasSuffix(name, ".cpp"): fallthrough
        case strings.HasSuffix(name, ".cxx"): fallthrough
        case strings.HasSuffix(name, ".C"): cc = "g++"
        case strings.HasSuffix(name, ".c"): cc = "gcc"
        case strings.HasSuffix(name, ".go"): cc = "gccgo"
        }

        if cc == "" {
                return nil
        }

        name = filepath.Join(dir, name)

        o := coll.target.AddIntermediateFile(name+".o", name)
        if o == nil {
                fmt.Fprintf(os.Stderr, "fatal: no intermediate: %v\n", name)
                return nil
        }

        o.Type = ".o"
        o.Properties["CC"] = cc
        coll.target.Properties["LD"] = cc

        if coll.target.Type == ".so" {
                o.AddArgs("-fPIC")
        }

        return o.Depends[0]
}
