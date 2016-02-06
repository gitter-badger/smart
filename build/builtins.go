//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "path/filepath"
        "fmt"
        "strings"
)

type builtin func(ctx *Context, loc location, args []string) string

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

func builtinDir(ctx *Context, loc location, args []string) string {
        var ds []string
        for _, a := range args {
                ds = append(ds, filepath.Dir(a))
        }
        return strings.Join(ds, " ")
}

func builtinInfo(ctx *Context, loc location, args []string) (s string) {
        if builtinInfoFunc != nil {
                builtinInfoFunc(args...)
        }
        return
}

func builtinUpper(ctx *Context, loc location, args []string) string {
        for i, s := range args {
                args[i] = strings.ToUpper(s)
        }
        return strings.Join(args, " ")
}

func builtinLower(ctx *Context, loc location, args []string) string {
        for i, s := range args {
                args[i] = strings.ToLower(s)
        }
        return strings.Join(args, " ")
}

func builtinTitle(ctx *Context, loc location, args []string) string {
        for i, s := range args {
                args[i] = strings.ToTitle(s)
        }
        return strings.Join(args, " ")
}

func builtinModule(ctx *Context, loc location, args []string) (s string) {
        var name, toolsetName, kind string
        if 0 < len(args) { name = strings.TrimSpace(args[0]) }
        if 1 < len(args) { toolsetName = strings.TrimSpace(args[1]) }
        if 2 < len(args) { kind = strings.TrimSpace(args[2]) }
        if name == "" {
                errorf("module name is required")
                return
        }
        if name == "me" {
                errorf("module name 'me' is reserved")
                return
        }

        var toolset toolset
        if ts, ok := toolsets[toolsetName]; !ok {
                //ctx.lineno -= 1; ctx.colno = ctx.prevColno + 1
                errorf("toolset `%v' unknown", toolsetName)
                if ts == nil { errorf("builtin fatal error") }
                // TODO: send arguments to toolset
        } else {
                toolset = ts.toolset
        }

        var m *Module
        var has bool
        if m, has = modules[name]; !has {
                m = &Module{
                        l: ctx.l,
                        Name: name,
                        Toolset: toolset,
                        Kind: kind,
                        location: loc, defines: make(map[string]*define, 32),
                }
                modules[m.Name] = m
                moduleOrderList = append(moduleOrderList, m)
        } else if (m.Toolset != nil && toolsetName != "") && (m.Kind != "" || kind != "") {
                //ctx.lineno -= 1; ctx.colno = ctx.prevColno + 1
                fmt.Printf("%v: previous module declaration `%v'\n", &(m.location), m.Name)
                errorf(fmt.Sprintf("module already been defined as \"%v, $v\"", m.Toolset, m.Kind))
        }

        if m.Toolset == nil && m.Kind == "" {
                m.Toolset = toolset
                m.Kind = kind
        }

        m.l = ctx.l // reset the lex (it could be created by $(use))
        if ctx.m != nil {
                ctx.moduleStack = append(ctx.moduleStack, ctx.m)
        }
        ctx.m = m

        // parsed arguments in forms like "PLATFORM=android-9"
        var a []string
        if 2 < len(args) { a = args[2:] }
        vars, rest := splitVarArgs(a)
        toolset.ConfigModule(ctx, m, rest, vars)
        return
}

func builtinCommit(ctx *Context, loc location, args []string) (s string) {
        if ctx.m == nil {
                errorf("no module defined")
                return
        }

        verbose("commit `%v' (%v)", ctx.m.Name, ctx.m.GetDir())

        moduleBuildList = append(moduleBuildList, pendedBuild{ctx.m, ctx, args})

        i := len(ctx.moduleStack)-1
        if 0 <= i {
                up := ctx.moduleStack[i]
                ctx.m.Parent = up
                ctx.moduleStack, ctx.m = ctx.moduleStack[0:i], up
        } else {
                ctx.m = nil
        }

        return
}

func builtinBuild(ctx *Context, loc location, args []string) string {
        errorf("use $(commit) instead")
        return ""
}

func builtinUse(ctx *Context, loc location, args []string) string {
        if ctx.m == nil { errorf("no module defined") }
        if ctx.m.Toolset == nil { errorf("no toolset for `%v'", ctx.m.Name) }

        for _, a := range args {
                a = strings.TrimSpace(a)
                if m, ok := modules[a]; ok {
                        ctx.m.Using = append(ctx.m.Using, m)
                        m.UsedBy = append(m.UsedBy, ctx.m)
                        ctx.m.Toolset.UseModule(ctx, ctx.m, m)
                } else {
                        m = &Module{
                                Name: a,
                                UsedBy: []*Module{ ctx.m },
                                defines: make(map[string]*define, 32),
                        }
                        ctx.m.Using = append(ctx.m.Using, m)
                        modules[a] = m
                        ctx.m.Toolset.UseModule(ctx, ctx.m, m)
                }
        }
        return ""
}
