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
        m := &module{
                name: args[0],
                toolset: args[1],
                kind: args[2],
                dir: filepath.Dir(p.file),
                location: location{ &p.file, p.lineno, p.colno },
                variables: make(map[string]*variable, 128),
        }

        p.setModule(m)

        fmt.Printf("TODO: module: %v\n", args)
        return ""
}

func internalBuild(p *parser, args []string) string {
        fmt.Printf("TODO: build: %v\n", args)
        return ""
}

func internalUse(p *parser, args []string) string {
        fmt.Printf("TODO: use: %v\n", args)
        return ""
}
