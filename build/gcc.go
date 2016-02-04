//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "fmt"
        "os"
        "strings"
        "path/filepath"
)

func init() {
        registerToolset("gcc", &_gcc{})
}

var gccSourcePatterns = []*filerule{
        { "asm", ^os.ModeType, `\.(s|S)$` },
        { "c", ^os.ModeType, `\.(c)$` },
        { "c++", ^os.ModeType, `\.(cpp|cxx|cc|CC|C)$` },
        { "header", ^os.ModeType, `\.(h)$` },
}

type _gcc struct {
}

func (gcc *_gcc) configModule(ctx *context, args []string, vars map[string]string) bool {
        if ctx.module != nil {
                return true
        }
        return false
}

func (gcc *_gcc) createActions(ctx *context, args []string) bool {
        var cmd *gccCommand
        var targetName = ctx.module.name
        switch ctx.module.kind {
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
                errorf(0, fmt.Sprintf("unknown type `%v'", ctx.module.kind))
        }
        cmd.mkdir = filepath.Join("out", ctx.module.name)
        ctx.module.action = newAction(filepath.Join("out", ctx.module.name, targetName), cmd)

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
        includes := splitFieldsWithPrefix(ctx.call("this.includes"), "-I")
        libdirs := splitFieldsWithPrefix(ctx.call("this.libdirs"), "-L")
        libs := splitFieldsWithPrefix(ctx.call("this.libs"), "-l")

        // Import includes and libs from using modules.
        var useMod func(mod *module)
        useMod = func(mod *module) {
                for _, u := range mod.using {
                        if v := strings.TrimSpace(ctx.callWith(u, "this.export.includes")); v != "" {
                                includes = append(includes, splitFieldsWithPrefix(v, "-I")...)
                        }
                        if v := strings.TrimSpace(ctx.callWith(u, "this.export.libdirs")); v != "" {
                                libdirs = append(libdirs, splitFieldsWithPrefix(v, "-L")...)
                        }
                        if v := strings.TrimSpace(ctx.callWith(u, "this.export.libs")); v != "" {
                                libs = append(libs, splitFieldsWithPrefix(v, "-l")...)
                        }                        
                        useMod(u)
                }
        }
        useMod(ctx.module)

        //fmt.Printf("libs: (%v) %v %v\n", ctx.module.name, libdirs, libs)

        cmdAs  := gccNewCommand("as",  "-c")
        cmdGcc := gccNewCommand("gcc", "-c")
        cmdGxx := gccNewCommand("g++", "-c")

        for _, c := range []*gccCommand{cmdAs, cmdGcc, cmdGxx} {
                c.args = append(c.args, includes...)
        }

        if ctx.module.kind != "static" {
                cmd.libdirs, cmd.libs = libdirs, libs
        }

        sources := ctx.module.getSources(ctx)
        if len(sources) == 0 { errorf(0, "no sources for `%v'", ctx.module.name) }
        //fmt.Printf("sources: %v: %v\n", ctx.module.name, sources)
        actions := createSourceTransformActions(sources, func(src string) (name string, c command) {
                var fr *filerule
                if fi, err := os.Stat(src); err != nil {
                        fr = matchFileName(src, gccSourcePatterns)
                } else {
                        fr = matchFileInfo(fi, gccSourcePatterns)
                }

                if fr == nil {
                        errorf(0, "unknown source `%v'", src)
                }
                
                switch fr.name {
                case "asm": c = cmdAs
                case "c":   c = cmdGcc
                        if cmd.path == "ld" { cmd.path = "gcc" }
                case "c++": c = cmdGxx
                        if cmd.path != "g++" && cmd.path != "ar" { cmd.path = "g++" }
                default:
                        errorf(0, "unknown language for source `%v'", src)
                }

                name = src + ".o"
                return
        })
        ctx.module.action.prequisites = append(ctx.module.action.prequisites, actions...)

        //fmt.Printf("module: %v, %v, %v\n", ctx.module.name, ctx.module.action.targets, len(ctx.module.action.prequisites))
        return ctx.module.action != nil
}

func (gcc *_gcc) useModule(ctx *context, m *module) bool {
        //fmt.Printf("TODO: use: %v by %v\n", m.name, ctx.module.name)
        return false
}

type gccCommand struct {
        excmd
        args []string
        libdirs, libs []string
}

func gccNewCommand(name string, args ...string) *gccCommand {
        return &gccCommand{
                excmd{ path: name, },
                args, []string{}, []string{},
        }
}

func (c *gccCommand) execute(targets []string, prequisites []string) bool {
        var args []string
        var target = targets[0]

        isar := c.path == "ar" || strings.HasSuffix(c.path, "-ar")

        if isar {
                args = append(c.args, target)
        } else {
                args = append([]string{ "-o", target, }, c.args...)
        }

        args = append(args, prequisites...)

        if !isar && 0 < len(c.libs) {
                args = append(args, append(c.libdirs, c.libs...)...)
        }

        return c.run(target, args...)
}
