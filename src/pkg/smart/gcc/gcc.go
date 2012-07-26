package gcc

import (
        ".."
        "path/filepath"
        "strings"
)

func init() {
        // ...
}

func New() (t *gcc) {
        t = &gcc{}
        return
}

type gcc struct {
        top string
        target *smart.Target
}

func (gcc *gcc) SetTop(top string) {
        gcc.top = top
}

func (gcc *gcc) Goals() []*smart.Target {
        return []*smart.Target{ gcc.target }
}

func (gcc *gcc) NewCollector(t *smart.Target) smart.Collector {
        return &gccCollector{ gcc, t }
}

func (gcc *gcc) Generate(t *smart.Target) error {
        switch {
        case strings.HasSuffix(t.Name, ".o"):
                return gcc.compile(t)
        case strings.HasSuffix(t.Name, ".a"):
                return gcc.archive(t)

                /**
                 *  TODO: think about this ignores.
                 *
                 *  Avoid linking these files.
                 */
        case strings.HasSuffix(t.Name, ".c"): fallthrough
        case strings.HasSuffix(t.Name, ".C"): fallthrough
        case strings.HasSuffix(t.Name, ".cc"):fallthrough
        case strings.HasSuffix(t.Name, ".cpp"): fallthrough
        case strings.HasSuffix(t.Name, ".cxx"):
                return nil
        }
        return gcc.link(t)
}

func (gcc *gcc) compile(t *smart.Target) error {
        cc, args := "cc", []string{ "-o", t.Name, }

        dl := len(t.Depends)
        switch (dl) {
        case 0: return smart.NewErrorf("no depends: %v\n", t)
        case 1:
                d0 := t.Depends[0]
                if cc = t.VarDef("CC", cc); cc == "" {
                        return smart.NewErrorf("unknown file type: %v", d0.Name)
                }

                args = append(args, t.JoinAllArgs()...)
                args = append(args, t.JoinUseesArgs("-I")...)
                args = append(args, t.JoinParentUseesArgs("-I")...)
                args = append(args, "-c", d0.Name)

        default:
                cc, args = "ld", append(args, "-r")
                for _, d := range t.Depends {
                        if !strings.HasSuffix(d.Name, ".o") {
                                return smart.NewErrorf("unexpected file type: %v", d)
                        }
                        args = append(args, d.Name)
                }
        }

        return smart.Run(cc, args...)
}

func (gcc *gcc) archive(t *smart.Target) error {
        ar, args := "ar", []string{ "crs", t.Name, }

        al := len(args)
        for _, d := range t.Depends {
                switch d.Type {
                case ".o":
                        args = append(args, d.String())
                default:
                        smart.Warn("ar: ignored: %v\n", d)
                }
        }

        if len(args) - al <= 0 {
                return smart.NewErrorf("no objects for archive: %v", t)
        }

        if ar = t.VarDef("AR", ar); ar == "" {
                return smart.NewErrorf("empty AR command for: %v", t)
        }

        return smart.Run(ar, args...)
}

func (gcc *gcc) link(t *smart.Target) error {
        ld := "ld" // the default linker is 'ld'
        args := []string{ "-o", t.Name, }

        if strings.HasSuffix(t.Name, ".so") {
                args = append(args, "-shared")
        }

        //smart.Info("link: %v (%v)\n", t.Name, gcc.target)

        for _, d := range t.Depends {
                switch d.Type {
                case ".a": fallthrough
                case ".so":
                        //args = append(args, d.JoinExports("-L")...)
                        //args = append(args, d.JoinExports("-l")...)
                case ".o":
                        args = append(args, d.Name)
                default:
                        smart.Info("link: ignored: %v\n", d)
                }
        }

        args = append(args, t.JoinArgs("-Wl,-rpath=")...)
        args = append(args, t.JoinArgs("-L")...)
        args = append(args, t.JoinArgs("-l")...)
        args = append(args, t.JoinUseesArgs("-Wl,-rpath=")...)
        args = append(args, t.JoinUseesArgs("-L")...)
        args = append(args, t.JoinUseesArgs("-l")...)

        if ld = t.VarDef("LD", ld); ld == "" {
                return smart.NewErrorf("empty LD command for: %v", t)
        }

        return smart.Run(ld, args...)
}

type gccCollector struct {
        gcc *gcc
        target *smart.Target
}

func (coll *gccCollector) ensureTarget(dir string) bool {
        if coll.target == nil {
                var name string
                if dir == "" {
                        name = filepath.Base(coll.gcc.top)
                } else {
                        name = filepath.Base(dir)
                }

                coll.target = smart.New(name, smart.FinalFile)

                if coll.gcc.target == nil {
                        coll.gcc.target = coll.target
                }
        }
        return coll.target != nil
}

func (coll *gccCollector) AddDir(dir string) (t *smart.Target) {
        if !coll.ensureTarget("") {
                smart.Fatal("no goal in %v\n", dir)
                return nil
        }

        switch {
        case strings.HasSuffix(dir, ".o"):
                t = smart.New(filepath.Join(dir, "_.o"), smart.FinalFile)
                t.Type = ".o"
        case strings.HasSuffix(dir, ".a"): fallthrough
        case strings.HasSuffix(dir, ".so"):
                name := filepath.Base(dir)
                ext := filepath.Ext(name)
                if !strings.HasPrefix(dir, "lib") {
                        name = "lib"+name
                }

                t = smart.New(filepath.Join(dir, name), smart.FinalFile)
                t.Type = ext

                l := len(name) - len(ext)
                t.AddExports("-I", dir)
                t.AddExports("-L", dir)
                t.AddExports("-l", name[3:l])
                if strings.HasSuffix(dir, ".so") {
                        rpath := dir //filepath.Join(coll.gcc.top, dir)
                        t.AddExports("-Wl,-rpath=", rpath)
                }

                coll.target.Use(t)
        }

        if t != nil {
                smart.Scan(coll.gcc.NewCollector(t), coll.gcc.top, dir)
                //fmt.Printf("scan: %v %v\n", dir, t.Depends)

                coll.target.Dep(t, smart.None)
                //fmt.Printf("TODO: AddDir: %v %v\n", t, t.Depends)
        }
        return t
}

func (coll *gccCollector) AddFile(dir, name string) *smart.Target {
        if !coll.ensureTarget(dir) {
                smart.Fatal("no goal in %v\n", dir)
                return nil
        }

        cc := ""
        switch {
        default: return nil
        case strings.HasSuffix(name, ".cc"):  fallthrough
        case strings.HasSuffix(name, ".cpp"): fallthrough
        case strings.HasSuffix(name, ".cxx"): fallthrough
        case strings.HasSuffix(name, ".C"):   cc = "g++"
        case strings.HasSuffix(name, ".c"):   cc = "gcc"
        case strings.HasSuffix(name, ".go"):  cc = "gccgo"
        }

        if cc == "" {
                return nil
        }

        name = filepath.Join(dir, name)

        //smart.Info("File: %v (%v)\n", name, coll.target)

        o := coll.target.Dep(name+".o", smart.IntermediateFile)
        o.Dep(name, smart.File)
        if o == nil {
                smart.Fatal("fatal: no intermediate: %v\n", name)
                return nil
        }

        o.Type = ".o"
        o.SetVar("CC", cc)
        coll.target.SetVar("LD", cc)

        if coll.target.Type == ".so" {
                o.AddArgs("-fPIC")
        }

        return o.Depends[0]
}

func build(args []string) error {
        tool := New()
        return smart.Build(tool)
}

func clean(args []string) error {
        return nil
}

func CommandLine(args []string) {
        var commands = map[string] func(args []string) error {
                "build": build,
                "clean": clean,
        }

        smart.CommandLine(commands, args)
}
