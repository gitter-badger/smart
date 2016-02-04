//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "path/filepath"
        "fmt"
        "strings"
)

type builtin func(ctx *context, args []string) string

var (
        builtins = map[string]builtin {
                "module":       builtinModule,
                "commit":       builtinCommit,
                "build":        builtinBuild,
                "dir":          builtinDir,
                "info":         builtinInfo,
                "use":          builtinUse,

                "upper":        builtinUpper,
                "lower":        builtinLower,
                "title":        builtinTitle,
        }

        builtinInfoFunc = func(args ...string) {
                fmt.Printf("%v\n", strings.Join(args, " "))
        }
)

func builtinDir(ctx *context, args []string) string {
        var ds []string
        for _, a := range args {
                ds = append(ds, filepath.Dir(a))
        }
        return strings.Join(ds, " ")
}

func builtinInfo(ctx *context, args []string) (s string) {
        if builtinInfoFunc != nil {
                builtinInfoFunc(args...)
        }
        return
}

func builtinUpper(ctx *context, args []string) string {
        for i, s := range args {
                args[i] = strings.ToUpper(s)
        }
        return strings.Join(args, " ")
}

func builtinLower(ctx *context, args []string) string {
        for i, s := range args {
                args[i] = strings.ToLower(s)
        }
        return strings.Join(args, " ")
}

func builtinTitle(ctx *context, args []string) string {
        for i, s := range args {
                args[i] = strings.ToTitle(s)
        }
        return strings.Join(args, " ")
}

func builtinModule(ctx *context, args []string) string {
        var name, toolsetName, kind string
        if 0 < len(args) { name = strings.TrimSpace(args[0]) }
        if 1 < len(args) { toolsetName = strings.TrimSpace(args[1]) }
        if 2 < len(args) { kind = strings.TrimSpace(args[2]) }
        if name == "" {
                ctx.setModule(nil)
                return ""
        }

        var toolset toolset
        if ts, ok := toolsets[toolsetName]; !ok {
                //ctx.lineno -= 1; ctx.colno = ctx.prevColno + 1
                errorf(0, "toolset `%v' unknown", toolsetName)
                if ts == nil { errorf(0, "builtin fatal error") }
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
                        dir: filepath.Dir(ctx.l.scope),
                        location: ctx.l.location(),
                        defines: make(map[string]*define, 32),
                }
                modules[m.name] = m
                moduleOrderList = append(moduleOrderList, m)
        } else if (m.toolset != nil && toolsetName != "") && (m.kind != "" || kind != "") {
                //ctx.lineno -= 1; ctx.colno = ctx.prevColno + 1
                fmt.Printf("%v: previous module declaration `%v'\n", &(m.location), m.name)
                errorf(0, fmt.Sprintf("module already been defined as \"%v, $v\"", m.toolset, m.kind))
        }

        if m.toolset == nil && m.kind == "" {
                m.toolset = toolset
                m.kind = kind
        }

        m.dir = filepath.Dir(ctx.l.scope)
        ctx.setModule(m)

        // parsed arguments in forms like "PLATFORM=android-9"
        var a []string
        if 2 < len(args) { a = args[2:] }
        vars, rest := splitVarArgs(a)

        fmt.Printf("vars: %v\n", vars)

        toolset.configModule(ctx, rest, vars)
        return ""
}

func builtinBuild(ctx *context, args []string) string {
        panic("use $(commit) instead")
}

func builtinCommit(ctx *context, args []string) string {
        var m *module
        if m = ctx.module; m == nil { errorf(0, "no module defined") }

        verbose("pending `%v' (%v)", m.name, m.dir)

        moduleBuildList = append(moduleBuildList, pendedBuild{m, ctx, args})
        return ""
}

func builtinUse(ctx *context, args []string) string {
        if ctx.module == nil { errorf(0, "no module defined") }
        if ctx.module.toolset == nil { errorf(0, "no toolset for `%v'", ctx.module.name) }

        for _, a := range args {
                a = strings.TrimSpace(a)
                if m, ok := modules[a]; ok {
                        ctx.module.using = append(ctx.module.using, m)
                        m.usedBy = append(m.usedBy, ctx.module)
                        ctx.module.toolset.useModule(ctx, m)
                } else {
                        m = &module{
                                name: a,
                                dir: filepath.Dir(ctx.l.scope),
                                location: ctx.l.location(),
                                defines: make(map[string]*define, 32),
                                usedBy: []*module{ ctx.module },
                        }
                        ctx.module.using = append(ctx.module.using, m)
                        modules[a] = m
                        ctx.module.toolset.useModule(ctx, m)
                }
        }
        return ""
}
