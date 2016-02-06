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

var gccSourcePatterns = []*FileMatchRule{
        { "asm",    ^os.ModeType, `\.(s|S)$` },
        { "c",      ^os.ModeType, `\.(c)$` },
        { "c++",    ^os.ModeType, `\.(cpp|cxx|cc|CC|C)$` },
        { "header", ^os.ModeType, `\.(h)$` },
}

type toolset struct { BasicToolset }

func (gcc *toolset) CreateActions(ctx *Context) bool {
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
                        fr = MatchFileName(src, gccSourcePatterns)
                } else {
                        fr = MatchFileInfo(fi, gccSourcePatterns)
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
        m.Action.Prequisites = append(m.Action.Prequisites, actions...)

        //fmt.Printf("module: %v, %v, %v\n", m.Name, m.Action.targets, len(m.Action.prequisites))
        return m.Action != nil
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

func (c *gccCommand) Execute(targets []string, prequisites []string) bool {
        var args []string
        var target = targets[0]

        isar := c.GetPath() == "ar" || strings.HasSuffix(c.GetPath(), "-ar")

        if isar {
                args = append(c.args, target)
        } else {
                args = append([]string{ "-o", target, }, c.args...)
        }

        args = append(args, prequisites...)

        if !isar && 0 < len(c.libs) {
                args = append(args, append(c.libdirs, c.libs...)...)
        }

        return c.Run(target, args...)
}
