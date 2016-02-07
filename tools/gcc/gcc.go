//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "fmt"
        "os"
        "strings"
        "path/filepath"
        . "github.com/duzy/smart/build"
)

func init() {
        RegisterToolset("gcc", &toolset{})
}

var (
        patterns = []*FileMatchRule{
                { "asm",    ^os.ModeType, `\.(s|S)$` },
                { "c",      ^os.ModeType, `\.(c)$` },
                { "c++",    ^os.ModeType, `\.(cpp|cxx|cc|CC|C)$` },
                { "header", ^os.ModeType, `\.(h)$` },
        }
)

func splitFieldsWithPrefix(ss, prefix string) (l []string) {
        for _, s := range strings.Fields(ss) /*strings.Split(ss, " ")*/ {
                if dont := strings.HasPrefix(s, prefix); s != "" {
                        if prefix == "-l" {
                                dont = dont || strings.ContainsAny(s, "/\\")
                                dont = dont || strings.HasSuffix(s, ".so")
                                dont = dont || strings.HasSuffix(s, ".a")
                        }
                        
                        if !dont { s = prefix + s }

                        l = append(l, s)
                }
        }
        return
}

type imported struct {
        includes, libdirs, libs []string
        prerequisites []*Action
        numAsm, numC, numCxx int
}

func (im *imported) load(ctx *Context) {
        im.includes = append(im.includes, splitFieldsWithPrefix(ctx.Call("me.includes"), "-I")...)
        im.libdirs = append(im.libdirs, splitFieldsWithPrefix(ctx.Call("me.libdirs"), "-L")...)
        im.libs = append(im.libs, splitFieldsWithPrefix(ctx.Call("me.libs"), "-l")...)
}

func (im *imported) importUsedModule(ctx *Context, m *Module) {
        importVars := func() {
                im.includes = append(im.includes, splitFieldsWithPrefix(ctx.Call("me.export.includes"), "-I")...)
                im.libdirs = append(im.libdirs, splitFieldsWithPrefix(ctx.Call("me.export.libdirs"), "-L")...)
                im.libs = append(im.libs, splitFieldsWithPrefix(ctx.Call("me.export.libs"), "-l")...)
        }
        for _, u := range m.Using {
                ctx.With(u, importVars)
                im.importUsedModule(ctx, u)
        }
        return
}

func createCompileActions(includes, sources []string) (actions []*Action, numAsm, numC, numCxx int) {
        as, gcc, gxx := NewExcmd("as"), NewExcmd("gcc"), NewExcmd("g++")
        actions = CreateSourceTransformActions(sources, func(src string) (target string, c Command) {
                var fr *FileMatchRule
                if fi, err := os.Stat(src); err != nil {
                        fr = MatchFileName(src, patterns)
                } else {
                        fr = MatchFileInfo(fi, patterns)
                }

                if fr == nil {
                        Fatal("unknown source `%v'", src)
                }

                target = src + ".o"

                switch fr.Name {
                case "asm": c = &Compile{ GccCmd{ ex:as },  includes }; numAsm++
                case "c":   c = &Compile{ GccCmd{ ex:gcc }, includes }; numC++
                case "c++": c = &Compile{ GccCmd{ ex:gxx }, includes }; numCxx++
                default: Fatal("unknown source language `%v'", src)
                }
                return
        })
        return
}

type toolset struct { BasicToolset }

func (gcc *toolset) createLinkAction(ctx *Context, out, ext string, im *imported) *Link {
        var (
                cmd = new(Link)
                m = ctx.CurrentModule()
                targetName = m.Name + ext
        )

        cmd.out, cmd.libdirs, cmd.libs = out, im.libdirs, im.libs

        m.Action = NewAction(filepath.Join(cmd.out, targetName), cmd)
        m.Action.Prerequisites = append(m.Action.Prerequisites, im.prerequisites...)

        switch {
        case 0 == im.numCxx && 0 == im.numC && 0 < im.numAsm:
                cmd.ex = NewExcmd("ld")
        case 0 == im.numCxx && 0 < im.numC:
                cmd.ex = NewExcmd("gcc")
        case 0 <  im.numCxx:
                cmd.ex = NewExcmd("g++")
        }

        if cmd.ex == nil {
                Fatal("no command (%v)", m.Name)
        }

        return cmd
}

func (gcc *toolset) createExe(ctx *Context, out string, im *imported) {
        cmd := gcc.createLinkAction(ctx, out, "", im)
        cmd.shared = false
}

func (gcc *toolset) createShared(ctx *Context, out string, im *imported) {
        cmd := gcc.createLinkAction(ctx, out, ".so", im)
        cmd.shared = true
}

func (gcc *toolset) createStatic(ctx *Context, out string, im *imported) {
        var (
                cmd = new(Archive)
                m = ctx.CurrentModule()
                targetName = m.Name
        )

        cmd.out = out

        if !strings.HasPrefix(targetName, "lib") { targetName = "lib" + targetName }
        if !strings.HasPrefix(targetName, ".a") { targetName = targetName + ".a" }

        m.Action = NewAction(filepath.Join(cmd.out, targetName), cmd)
        m.Action.Prerequisites = append(m.Action.Prerequisites, im.prerequisites...)

        cmd.ex = NewExcmd("ar")
}

func (gcc *toolset) CreateActions(ctx *Context) bool {
        var (
                m = ctx.CurrentModule()
                out = filepath.Join("out", m.Name)
        )

        // Add proper prefixes to includes, libdirs, libs.
        im := new(imported)
        im.load(ctx)

        // Import includes and libs from using modules.
        if 0 < len(m.Using) {
                im.importUsedModule(ctx, m)
        }

        sources := m.GetSources(ctx)
        if len(sources) == 0 { Fatal("no sources (%v)", m.Name) }

        im.prerequisites, im.numAsm, im.numC, im.numCxx = createCompileActions(im.includes, sources)

        switch m.Kind {
        case "exe":    gcc.createExe(ctx, out, im)
        case "shared": gcc.createShared(ctx, out, im)
        case "static": gcc.createStatic(ctx, out, im)
        default: Fatal(fmt.Sprintf("unknown type `%v'", m.Kind))
        }

        return m.Action != nil
}

type GccCmd struct {
        ex *Excmd
        out string
}

type Compile struct {
        GccCmd
        includes []string
}

func (c *Compile) Execute(targets []string, prerequisites []string) bool {
        if numExpected := len(targets); numExpected == 0 {
                Fatal("linking for zero")
        } else if 1 < numExpected {
                Fatal("linking multiple targets")
        }

        var (
                target = targets[0]
                args = []string{ "-o", target, "-c" }
        )

        if c.ex == nil { Fatal("nil command (%v)", target) }

        args = append(args, c.includes...)
        args = append(args, prerequisites...)
        return c.ex.Run(target, args...)
}

type Link struct {
        GccCmd
        shared bool
        libdirs, libs []string
}

func (c *Link) Execute(targets []string, prerequisites []string) bool {
        if numExpected := len(targets); numExpected == 0 {
                Fatal("linking for zero")
        } else if 1 < numExpected {
                Fatal("linking multiple targets")
        }

        var (
                target = targets[0]
                args = []string{ "-o", target }
        )

        if c.ex == nil { Fatal("nil command (%v)", target) }
        if c.shared { args = append(args, "-shared") }

        args = append(args, prerequisites...)
        args = append(args, c.libdirs...)
        args = append(args, c.libs...)

        c.ex.SetMkdir(c.out)
        return c.ex.Run(target, args...)
}

type Archive struct {
        GccCmd
}

func (c *Archive) Execute(targets []string, prerequisites []string) bool {
       if numExpected := len(targets); numExpected == 0 {
                Fatal("linking for zero")
        } else if 1 < numExpected {
                Fatal("linking multiple targets")
        }

        var (
                target = targets[0]
                args = []string{ "crs", target }
        )

        args = append(args, prerequisites...)

        c.ex.SetMkdir(c.out)
        return c.ex.Run(target, args...)
}













type gccCommand struct {
        *Excmd
        args []string
        libdirs, libs []string
}

func gccNewCommand(name string, args ...string) *gccCommand {
        return &gccCommand{
                NewExcmd(name), args, []string{}, []string{},
        }
}

func (c *gccCommand) Execute(targets []string, prerequisites []string) bool {
        var args []string
        var target = targets[0]

        isar := c.GetPath() == "ar" || strings.HasSuffix(c.GetPath(), "-ar")

        if isar {
                args = append(c.args, target)
        } else {
                args = append([]string{ "-o", target, }, c.args...)
        }

        args = append(args, prerequisites...)

        if !isar && 0 < len(c.libs) {
                args = append(args, append(c.libdirs, c.libs...)...)
        }

        return c.Run(target, args...)
}

func (gcc *toolset) createAny(ctx *Context) {
        var (
                m = ctx.CurrentModule()
                targetName = m.Name
                cmd *gccCommand
        )
        switch m.Kind {
        case "exe":
                cmd = gccNewCommand("ld")
        case "shared":
                if !strings.HasSuffix(targetName, ".so") { targetName = targetName + ".so" }
                cmd = gccNewCommand("ld", "-shared")
        case "static":
                if !strings.HasPrefix(targetName, "lib") { targetName = "lib" + targetName }
                if !strings.HasSuffix(targetName, ".a")  { targetName = targetName + ".a" }
                cmd = gccNewCommand("ar", "crs")
        default:
                Fatal(fmt.Sprintf("unknown type `%v'", m.Kind))
        }
        cmd.SetMkdir(filepath.Join("out", m.Name))

        m.Action = NewAction(filepath.Join("out", m.Name, targetName), cmd)

        // Add proper prefixes to includes, libdirs, libs.
        splitFieldsWithPrefix := func(ss, prefix string) (l []string) {
                for _, s := range strings.Split(ss, " ") {
                        if dont := strings.HasPrefix(s, prefix); s != "" {
                                if prefix == "-l" {
                                        dont = dont || strings.ContainsAny(s, "/\\")
                                        dont = dont || strings.HasSuffix(s, ".so")
                                        dont = dont || strings.HasSuffix(s, ".a")
                                }

                                if !dont { s = prefix + s }

                                l = append(l, s)
                        }
                }
                return
        }
        includes := splitFieldsWithPrefix(ctx.Call("includes"), "-I")
        libdirs := splitFieldsWithPrefix(ctx.Call("libdirs"), "-L")
        libs := splitFieldsWithPrefix(ctx.Call("libs"), "-l")

        // Import includes and libs from using modules.
        var useMod func(mod *Module)
        useMod = func(mod *Module) {
                for _, u := range mod.Using {
                        if v := strings.TrimSpace(ctx.CallWith(u, "export.includes")); v != "" {
                                includes = append(includes, splitFieldsWithPrefix(v, "-I")...)
                        }
                        if v := strings.TrimSpace(ctx.CallWith(u, "export.libdirs")); v != "" {
                                libdirs = append(libdirs, splitFieldsWithPrefix(v, "-L")...)
                        }
                        if v := strings.TrimSpace(ctx.CallWith(u, "export.libs")); v != "" {
                                libs = append(libs, splitFieldsWithPrefix(v, "-l")...)
                        }                        
                        useMod(u)
                }
        }
        useMod(m)

        //fmt.Printf("libs: (%v) %v %v\n", m.Name, libdirs, libs)

        cmdAs  := gccNewCommand("as",  "-c")
        cmdGcc := gccNewCommand("gcc", "-c")
        cmdGxx := gccNewCommand("g++", "-c")

        for _, c := range []*gccCommand{cmdAs, cmdGcc, cmdGxx} {
                c.args = append(c.args, includes...)
        }

        if m.Kind != "static" {
                cmd.libdirs, cmd.libs = libdirs, libs
        }

        sources := m.GetSources(ctx)
        if len(sources) == 0 { Fatal("no sources for `%v'", m.Name) }

        actions := CreateSourceTransformActions(sources, func(src string) (name string, c Command) {
                var fr *FileMatchRule
                if fi, err := os.Stat(src); err != nil {
                        fr = MatchFileName(src, patterns)
                } else {
                        fr = MatchFileInfo(fi, patterns)
                }

                if fr == nil {
                        Fatal("unknown source `%v'", src)
                }
                
                switch fr.Name {
                case "asm": c = cmdAs
                case "c":   c = cmdGcc
                        if cmd.GetPath() == "ld" { cmd.SetPath("gcc") }
                case "c++": c = cmdGxx
                        if s := cmd.GetPath(); s != "g++" && s != "ar" { cmd.SetPath("g++") }
                default:
                        Fatal("unknown language for source `%v'", src)
                }

                name = src + ".o"
                return
        })
        m.Action.Prerequisites = append(m.Action.Prerequisites, actions...)
}
