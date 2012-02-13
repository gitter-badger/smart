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
                        location: location{ &p.file, p.lineno, p.colno },
                        variables: make(map[string]*variable, 128),
                }
                modules[m.name] = m
        } else if toolsetName != "" || kind != "" {
                p.lineno -= 1; p.colno = p.prevColno + 1
                fmt.Printf("%v: previous module declaration \"%s\"", &(m.location), m.name)
                panic(p.newError(0, fmt.Sprintf("module already been defined as \"%s, $s\"", m.toolset, m.kind)))
        }

        p.setModule(m)
        return ""
}

func internalBuild(p *parser, args []string) string {
        if m := p.module; m == nil {
                panic(p.newError(0, "no module defined"))
        }

        fmt.Printf("smart: build `%v'\n", p.module.name)

        p.module.build(p, args)
        return ""
}

func internalUse(p *parser, args []string) string {
        fmt.Printf("TODO: use: %v\n", args)
        return ""
}
