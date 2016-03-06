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

type builtin func(ctx *Context, loc location, args Items) Items

var (
        builtins = map[string]builtin {
                "template":     builtinTemplate,
                "module":       builtinModule,
                "commit":       builtinCommit,
                "dir":          builtinDir,
                "info":         builtinInfo,
                "use":          builtinUse,

                "upper":        builtinUpper,
                "lower":        builtinLower,
                "title":        builtinTitle,

                "when":         builtinWhen,
                "unless":       builtinUnless,
                "let":          builtinLet,

                "expr":         builtinExpr,

                "=":            builtinSet,
                //"!=":         builtinSetNot,
                "?=":           builtinSetQuestioned,
                "+=":           builtinSetAppend,
        }

        builtinInfoFunc = func(ctx *Context, args Items) {
                var as []string
                for _, a := range args {
                        as = append(as, a.Expand(ctx))
                }
                fmt.Printf("%v\n", strings.Join(as, ","))
        }
)

func builtinDir(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(filepath.Dir(ctx.ItemString(a))))
        }
        return
}

func builtinInfo(ctx *Context, loc location, args Items) (is Items) {
        if builtinInfoFunc != nil {
                builtinInfoFunc(ctx, args)
        }
        return
}

func builtinUpper(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToUpper(ctx.ItemString(a))))
        }
        return
}

func builtinLower(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToLower(ctx.ItemString(a))))
        }
        return
}

func builtinTitle(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToTitle(ctx.ItemString(a))))
        }
        return
}

func builtinSet(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                ctx.Set(strings.TrimSpace(args[0].Expand(ctx)), args[1:]...)
        }
        return
}

func builtinSetNot(ctx *Context, loc location, args Items) (is Items) {
        panic("todo: $(!= name, ...)")
        return
}

func builtinSetQuestioned(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                name := strings.TrimSpace(args[0].Expand(ctx))
                if i := strings.Index(name, ":"); 0 <= i {
                        prefix, parts := name[0:i], strings.Split(name[i+1:], ".")
                        if ii := ctx.callScoped(loc, prefix, parts); ii.IsEmpty(ctx) {
                                ctx.setScoped(prefix, parts, args[1:]...)
                        }
                } else {
                        parts := strings.Split(name, ".")
                        if ctx.getMultipart(parts) == nil {
                                ctx.setMultipart(parts, args[1:]...)
                        }
                }
        }
        return
}

func builtinSetAppend(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                name := strings.TrimSpace(args[0].Expand(ctx))
                if i := strings.Index(name, ":"); 0 <= i {
                        prefix, parts := name[0:i], strings.Split(name[i+1:], ".")
                        ctx.setScoped(prefix, parts, ctx.callScoped(loc, prefix, parts).Concat(ctx, args[1:]...)...)
                } else {
                        parts := strings.Split(name, ".")
                        if ctx.getMultipart(parts) == nil {
                                ctx.setMultipart(parts, args[1:]...)
                        }
                }
        }
        return
}

func builtinWhen(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

func builtinUnless(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

func builtinLet(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

// builtinExpr evaluates a math expression.
func builtinExpr(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

func builtinTemplate(ctx *Context, loc location, args Items) (is Items) {
        if ctx.t != nil {
                errorf("template already defined (%v)", args)
        } else {
                if ctx.m != nil {
                        s, lineno, colno := ctx.m.GetDeclareLocation()
                        fmt.Printf("%v:%v:%v:warning: declare template in module\n", s, lineno, colno)

                        lineno, colno = ctx.l.caculateLocationLineColumn(loc)
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: ", ctx.l.scope, lineno, colno)

                        errorf("declare template inside module")
                        return
                }

                name := strings.TrimSpace(args[0].Expand(ctx))
                if name == "" {
                        lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: empty template name", ctx.l.scope, lineno, colno)
                        errorf("empty template name")
                        return
                }

                if t, ok := ctx.templates[name]; ok && t != nil {
                        //lineno, colno := ctx.l.caculateLocationLineColumn(t.loc)
                        //fmt.Fprintf(os.Stderr, "%v:%v:%v: %s already declared", ctx.l.scope, lineno, colno, ctx.t.name)
                        errorf("template '%s' already declared", name)
                        return
                }

                ctx.t = &template{
                        name:name,
                }
        }
        return
}

func builtinModule(ctx *Context, loc location, args Items) (is Items) {
        var name, exportName, toolsetName string
        if 0 < len(args) { name = strings.TrimSpace(args[0].Expand(ctx)) }
        if 1 < len(args) { toolsetName = strings.TrimSpace(args[1].Expand(ctx)) }
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
                /*
                var a []string
                if 2 < len(args) { a = args[2:] }
                vars, rest := splitVarArgs(a) */
                var rest Items
                vars := make(map[string]string, 4)
                for _, a := range args[2:] {
                        s := a.Expand(ctx)
                        if i := strings.Index(s, "="); 0 < i /* false if '=foo' */ {
                                vars[strings.TrimSpace(s[0:i])] = strings.TrimSpace(s[i+1:])
                        } else {
                                rest = append(rest, a)
                        }
                }
                toolset.ConfigModule(ctx, rest, vars)
        }

        if fi, e := os.Stat(ctx.l.scope); e == nil && fi != nil && !fi.IsDir() {
                ctx.Set("me.dir", stringitem(filepath.Dir(ctx.l.scope)))
        } else {
                ctx.Set("me.dir", stringitem(workdir))
        }
        ctx.Set("me.name", stringitem(name))
        ctx.Set("me.export.name", stringitem(exportName))
        return
}

func builtinCommit(ctx *Context, loc location, args Items) (is Items) {
        if ctx.t != nil {
                if ctx.m != nil {
                        errorf("declared template inside module")
                        return
                }
                if t, ok := ctx.templates[ctx.t.name]; ok && t != nil {
                        errorf("template '%s' already declared", ctx.t.name)
                        return
                }

                t := ctx.t
                ctx.templates[t.name] = t
                ctx.t = nil
                return
        }

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

func builtinUse(ctx *Context, loc location, args Items) (is Items) {
        if ctx.m == nil { errorf("no module defined") }

        for _, a := range args {
                s := strings.TrimSpace(a.Expand(ctx))
                if m, ok := ctx.modules[s]; ok {
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
                        ctx.modules[s] = m
                        if ctx.m.Toolset != nil {
                                ctx.m.Toolset.UseModule(ctx, m)
                        }
                }
        }
        return
}
