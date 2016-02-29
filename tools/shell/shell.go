//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        "fmt"
        "strings"
        "path/filepath"
        . "github.com/duzy/smart/build"
)

func init() {
        RegisterToolset("shell", &toolset{})
}

type toolset struct { BasicToolset }

func (shell *toolset) ConfigModule(ctx *Context, args []string, vars map[string]string) {
        var (
                m = ctx.CurrentModule()
                commandPath string
        )

        if num := len(args); 1 == num {
                commandPath = strings.TrimSpace(args[0])
                ctx.Set("me.path", commandPath)
        } else if 1 < num {
                commandPath = strings.Join(args, " ")
                commandPath = strings.Join(Split(commandPath), " ")
                ctx.Set("me.path", commandPath)
        } else {
                s, l, c := m.GetDeclareLocation()
                fmt.Fprintf(os.Stderr, "no commands", s, l, c)
        }
}

func (shell *toolset) CreateActions(ctx *Context) bool {
        ac := &command{
                args: Split(ctx.Call("me.args")),
                targets: Split(ctx.Call("me.targets")),
        }

        m := ctx.CurrentModule()
        d := m.GetDir(ctx)

        for _, s := range Split(ctx.Call("me.path")) {
                c := NewExcmd(s)
                c.SetDir(d)
                ac.cmds = append(ac.cmds, c)
        }

        m.Action = NewInterAction(m.GetName(ctx), ac)

        for i, s := range ac.targets {
                if !filepath.IsAbs(s) {
                        ac.targets[i] = filepath.Join(d, s)
                }
        }

        for _, s := range Split(ctx.Call("me.depends")) {
                if !filepath.IsAbs(s) {
                        s = filepath.Join(d, s)
                }
                m.Action.Prerequisites = append(m.Action.Prerequisites, NewAction(s, nil))
        }

        // TODO: handle m.Using

        return true
}

func (shell *toolset) UseModule(ctx *Context, o *Module) bool {
        return false
}

type command struct {
        cmds []*Excmd
        args, targets []string
}

func (c *command) Targets(prerequisites []*Action) (targets []string, needsUpdate bool) {
        outdates, _ := ComputeKnownInterTargets(c.targets, prerequisites)
        targets, needsUpdate = c.targets, 0 < outdates
        return
}

func (c *command) Execute(targets []string, prerequisites []string) bool {
        for _, ex := range c.cmds {
                var (
                        a []string
                        m = make(map[string]int)
                )
                for _, t := range targets { m[filepath.Base(t)]++ }
                for s, _ := range m { a = append(a, s) }
                if !ex.Run(fmt.Sprintf("%v", a), c.args...) {
                        return false
                }
        }
        return 0 < len(c.cmds)
}
