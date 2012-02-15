package smart

import (
        "path/filepath"
        "fmt"
        "strings"
)

var internals = map[string]func(p *parser, args []string) string {
        "info": internalInfo,
        "module": internalModule,
        "build": internalBuild,
        "use": internalUse,
}

func internalInfo(p *parser, args []string) string {
        fmt.Printf("%v\n", strings.Join(args, " "))
        return ""
}

func internalModule(p *parser, args []string) string {
        var name, toolsetName, kind string
        if 0 < len(args) { name = strings.TrimSpace(args[0]) }
        if 1 < len(args) { toolsetName = strings.TrimSpace(args[1]) }
        if 2 < len(args) { kind = strings.TrimSpace(args[2]) }
        if name == "" {
                p.setModule(nil)
                return ""
        }

        var toolset toolset
        if ts, ok := toolsets[toolsetName]; !ok {
                p.lineno -= 1; p.colno = p.prevColno + 1
                panic(p.newError(0, fmt.Sprintf("toolset \"%s\" not existed", toolset)))
                if ts == nil { panic(p.newError(0, "internal fatal error")) }
                // TODO: send arguments to toolset
        } else {
                toolset = ts.toolset
        }

        var m *module
        var has bool
        if m, has = modules[name]; !has {
                m = &module{
                        name: name,
                        toolset: toolset,
                        kind: kind,
                        dir: filepath.Dir(p.file),
                        location: location{ &p.file, p.lineno-1, p.prevColno+1 },
                        variables: make(map[string]*variable, 128),
                }
                modules[m.name] = m
        } else if (m.toolset != nil && toolsetName != "") && (m.kind != "" || kind != "") {
                p.lineno -= 1; p.colno = p.prevColno + 1
                fmt.Printf("%v: previous module declaration `%v'\n", &(m.location), m.name)
                panic(p.newError(0, fmt.Sprintf("module already been defined as \"%v, $v\"", m.toolset, m.kind)))
        }

        if m.toolset == nil && m.kind == "" {
                m.toolset = toolset
                m.kind = kind
        }

        p.setModule(m)
        toolset.setupModule(p, args[3:])
        return ""
}

func internalBuild(p *parser, args []string) string {
        var m *module
        if m = p.module; m == nil {
                panic(p.newError(0, "no module defined"))
        }

        var buildUsing func(mod *module) int
        buildUsing = func(mod *module) (num int) {
                for _, u := range mod.using {
                        ok := true
                        if u.toolset == nil {
                                ok = false
                        } else if l := len(u.using); 0 < l {
                                if l != buildUsing(u) { ok = false }
                        }
                        if ok && u.toolset.buildModule(p, args) {
                                num += 1
                        } else {
                                fmt.Printf("%v:%v:%v: dependency `%v' not built\n", p.file, p.lineno-1, p.prevColno+1, u.name)
                        }
                }
                return
        }

        if buildUsing(m) != len(m.using) {
                panic(p.newError(0, "not all dependencies built for `%v'", m.name))
        }

        if m.toolset == nil {
                panic(p.newError(0, "no toolset for `%v'", m.name))
        }

        if *flag_v {
                fmt.Printf("smart: build `%v'\n", m.name)
        }

        if !m.toolset.buildModule(p, args) {
                panic(p.newError(0, "failed building `%v' via `%v'", m.name, m.toolset))
        }
        return ""
}

func internalUse(p *parser, args []string) string {
        if m := p.module; m == nil {
                panic(p.newError(0, "no module defined"))
        }

        for _, a := range args {
                a = strings.TrimSpace(a)
                if m, ok := modules[a]; ok {
                        p.module.using = append(p.module.using, m)
                        m.usedBy = append(m.usedBy, p.module)
                } else {
                        m = &module{
                        name: a,
                        dir: filepath.Dir(p.file),
                        location: location{ &p.file, p.lineno, p.colno },
                        variables: make(map[string]*variable, 128),
                        usedBy: []*module{ p.module },
                        }
                        p.module.using = append(p.module.using, m)
                        modules[a] = m
                }
        }
        return ""
}
