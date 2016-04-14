//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "path/filepath"
        "strings"
        "fmt"
)

type builtin func(ctx *Context, loc location, args Items) Items

var (
        builtins = map[string]builtin {
                "dir":          builtinDir,
                "info":         builtinInfo,

                "upper":        builtinUpper,
                "lower":        builtinLower,
                "title":        builtinTitle,

                "when":         builtinWhen,
                "unless":       builtinUnless,
                "let":          builtinLet,
                "set":          builtinSet,

                "expr":         builtinExpr,

                "=":            builtinSetEqual,
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

func SetBuiltinInfoFunc(f func(ctx *Context, args Items)) func(ctx *Context, args Items) {
        previous := builtinInfoFunc
        builtinInfoFunc = f
        return previous
}

func builtinDir(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(filepath.Dir(a.Expand(ctx))))
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
                is = append(is, stringitem(strings.ToUpper(a.Expand(ctx))))
        }
        return
}

func builtinLower(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToLower(a.Expand(ctx))))
        }
        return
}

func builtinTitle(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToTitle(a.Expand(ctx))))
        }
        return
}

func builtinSet(ctx *Context, loc location, args Items) (is Items) {
        return builtinSetEqual(ctx, loc, args)
}

func builtinSetEqual(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                name := strings.TrimSpace(args[0].Expand(ctx))
                hasPrefix, prefix, parts := ctx.expandNameString(name)
                ctx.setWithDetails(hasPrefix, prefix, parts, args[1:]...)
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
                hasPrefix, prefix, parts := ctx.expandNameString(name)
                if d := ctx.getDefineWithDetails(hasPrefix, prefix, parts); d == nil || d.value.IsEmpty(ctx) {
                        ctx.setWithDetails(hasPrefix, prefix, parts, args[1:]...)
                }
        }
        return
}

func builtinSetAppend(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                name := strings.TrimSpace(args[0].Expand(ctx))
                hasPrefix, prefix, parts := ctx.expandNameString(name)
                if d := ctx.getDefineWithDetails(hasPrefix, prefix, parts); d == nil {
                        ctx.setWithDetails(hasPrefix, prefix, parts, args[1:]...)
                } else {
                        d.value = append(d.value, args...)
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
