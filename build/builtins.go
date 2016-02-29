//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "path/filepath"
        "strings"
        "fmt"
        "os"
)

type builtin func(ctx *Context, loc location, args []string) string

var (
        builtins = map[string]builtin {
                "toolset":      builtinToolset,
                "module":       builtinModule,
                "commit":       builtinCommit,
                "dir":          builtinDir,
                "info":         builtinInfo,
                "use":          builtinUse,

                "upper":        builtinUpper,
                "lower":        builtinLower,
                "title":        builtinTitle,

                "=":           builtinSet,
                //"!=":           builtinSetNot,
                "?=":           builtinSetQuestioned,
                "+=":           builtinSetAppend,
        }

        builtinInfoFunc = func(args ...string) {
                fmt.Printf("%v\n", strings.Join(args, ","))
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

func builtinSet(ctx *Context, loc location, args []string) (s string) {
        if num := len(args); 1 < num {
                var i []interface{}
                for _, a := range args[1:] {
                        i = append(i, strings.TrimSpace(a))
                }
                ctx.Set(strings.TrimSpace(args[0]), i...)
        }
        return
}

func builtinSetNot(ctx *Context, loc location, args []string) (s string) {
        panic("todo: $(!= name, ...)")
        return
}

func builtinSetQuestioned(ctx *Context, loc location, args []string) (s string) {
        if num := len(args); 1 < num {
                var items []interface{}
                for _, a := range args[1:] { items = append(items, a) }

                name := strings.TrimSpace(args[0])
                if i := strings.Index(name, ":"); 0 <= i {
                        prefix, parts := name[0:i], strings.Split(name[i+1:], ".")
                        if ctx.callScoped(loc, prefix, parts) == "" {
                                ctx.setScoped(prefix, parts, items...)
                        }
                } else {
                        parts := strings.Split(name, ".")
                        if ctx.getMultipart(parts) == nil {
                                ctx.setMultipart(parts, items...)
                        }
                }
        }
        return
}

func builtinSetAppend(ctx *Context, loc location, args []string) (s string) {
        if num := len(args); 1 < num {
                var items []interface{}
                for _, a := range args[1:] { items = append(items, a) }

                name := strings.TrimSpace(args[0])
                if i := strings.Index(name, ":"); 0 <= i {
                        prefix, parts := name[0:i], strings.Split(name[i+1:], ".")
                        if s := ctx.callScoped(loc, prefix, parts); s != "" {
                                items = append([]interface{}{ s }, items...)
                        }
                        ctx.setScoped(prefix, parts, items...)
                } else {
                        parts := strings.Split(name, ".")
                        if ctx.getMultipart(parts) == nil {
                                ctx.setMultipart(parts, items...)
                        }
                }
        }
        return
}

func builtinToolset(ctx *Context, loc location, args []string) (s string) {
        errorf("todo: %v", args)
        return
}

func builtinModule(ctx *Context, loc location, args []string) (s string) {
        var name, exportName, toolsetName string
        if 0 < len(args) { name = strings.TrimSpace(args[0]) }
        if 1 < len(args) { toolsetName = strings.TrimSpace(args[1]) }
        if name == "" {
                errorf("module name is required")
                return
        }
        if name == "me" {
                errorf("module name 'me' is reserved")
                return
        }

        exportName = "export"

        var toolset toolset
        if toolsetName == "" {
                // Discard empty toolset.
        } else if ts, ok := toolsets[toolsetName]; !ok {
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                fmt.Printf("%v:%v:%v: unknown toolset '%v'\n", ctx.l.scope, lineno, colno, toolsetName)
        } else {
                toolset = ts.toolset
        }

        var (
                m *Module
                has bool
        )
        if m, has = ctx.modules[name]; !has && m == nil {
                m = &Module{
                        l: nil,
                        Toolset: toolset,
                        Children: make(map[string]*Module, 2),
                        defines: make(map[string]*define, 8),
                        rules: make(map[string]*rule, 4),
                }
                ctx.modules[name] = m
                ctx.moduleOrderList = append(ctx.moduleOrderList, m)
        } else if m.l != nil /*(m.Toolset != nil && toolsetName != "") && (m.Kind != "" || kind != "")*/ {
                s := ctx.l.scope
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                fmt.Printf("%v:%v:%v: '%v' already declared\n", s, lineno, colno, name)

                s, lineno, colno = m.GetDeclareLocation()
                fmt.Printf("%v:%v:%v:warning: previous '%v'\n", s, lineno, colno, name)

                errorf("module already declared")
        }

        // Reset the current module pointer.
        upper := ctx.m
        if upper != nil {
                upper.Children[name] = m
        }

        ctx.m = m

        // Reset the lex and location (because it could be created by $(use))
        if m.l == nil {
                m.l, m.declareLoc, m.Toolset = ctx.l, loc, toolset
                if upper != nil {
                        ctx.moduleStack = append(ctx.moduleStack, upper)
                }
        }

        if x, ok := m.Children[exportName]; !ok {
                x = &Module{
                        l: m.l,
                        Parent: m,
                        Children: make(map[string]*Module),
                        defines: make(map[string]*define, 4),
                        rules: make(map[string]*rule),
                }
                m.Children[exportName] = x
        }

        if toolset != nil {
                // parsed arguments in forms like "PLATFORM=android-9"
                var a []string
                if 2 < len(args) { a = args[2:] }
                vars, rest := splitVarArgs(a)
                toolset.ConfigModule(ctx, rest, vars)
        }

        if fi, e := os.Stat(ctx.l.scope); e == nil && fi != nil && !fi.IsDir() {
                ctx.Set("me.dir", filepath.Dir(ctx.l.scope))
        } else {
                ctx.Set("me.dir", workdir)
        }
        ctx.Set("me.name", name)
        ctx.Set("me.export.name", exportName)
        return
}

func builtinCommit(ctx *Context, loc location, args []string) (s string) {
        if ctx.m == nil {
                errorf("no module defined")
                return
        }

        if *flagVV {
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                //verbose("commit `%v' (%v:%v:%v)", ctx.m.GetName(ctx), ctx.l.scope, lineno, colno)
                verbose("commit (%v:%v:%v)", ctx.l.scope, lineno, colno)
        }

        ctx.m.commitLoc = loc
        ctx.moduleBuildList = append(ctx.moduleBuildList, pendedBuild{ctx.m, ctx, args})

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

func builtinUse(ctx *Context, loc location, args []string) string {
        if ctx.m == nil { errorf("no module defined") }

        for _, a := range args {
                a = strings.TrimSpace(a)
                if m, ok := ctx.modules[a]; ok {
                        ctx.m.Using = append(ctx.m.Using, m)
                        m.UsedBy = append(m.UsedBy, ctx.m)
                        if ctx.m.Toolset != nil {
                                ctx.m.Toolset.UseModule(ctx, m)
                        }
                } else {
                        m = &Module{
                                // Use 'nil' to indicate this module is created by
                                // '$(use)' and not really declared yet.
                                l: nil,
                                UsedBy: []*Module{ ctx.m },
                                Children: make(map[string]*Module, 2),
                                defines: make(map[string]*define, 8),
                                rules: make(map[string]*rule, 4),
                        }
                        ctx.m.Using = append(ctx.m.Using, m)
                        ctx.modules[a] = m
                        if ctx.m.Toolset != nil {
                                ctx.m.Toolset.UseModule(ctx, m)
                        }
                }
        }
        return ""
}
