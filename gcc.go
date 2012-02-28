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

func (gcc *_gcc) setupModule(p *parser, args []string, vars map[string]string) bool {
        var m *module
        if m = p.module; m == nil {
                errorf(0, "no module")
        }

        out := "out"

        if m.action != nil {
                errorf(0, "module `%v' already has a action", m.name)
                return true
        }

        var c *gccCommand
        var name = m.name
        switch m.kind {
        case "exe":
                c = gccNewCommand("ld")
        case "shared":
                if !strings.HasSuffix(name, ".so") { name = name + ".so" }
                c = gccNewCommand("ld", "-shared")
        case "static":
                if !strings.HasPrefix(name, "lib") { name = "lib" + name }
                if !strings.HasSuffix(name, ".a") { name = name + ".a" }
                c = gccNewCommand("ar", "crs")
        default:
                errorf(0, fmt.Sprintf("unknown type `%v'", m.kind))
        }

        c.mkdir = filepath.Join(out, m.name)
        m.action = newAction(filepath.Join(out, m.name, name), c)
        return true
}

func (gcc *_gcc) buildModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil { errorf(0, "no module") }
        if m.action == nil { errorf(0, "no action for `%v'", p.module.name) }
        if m.action.command == nil { errorf(0, "no command for `%v'", p.module.name) }

        var ld *gccCommand
        if l, ok := m.action.command.(*gccCommand); !ok {
                errorf(0, "internal: wrong module command")
        } else {
                ld = l
        }

        sources := p.getModuleSources()
        if len(sources) == 0 { errorf(0, "no sources for `%v'", p.module.name) }

        fmt.Printf("sources: %v: %v\n", m.name, sources)

        ls := func(name, prefix string) (l []string) {
                for _, s := range strings.Split(p.call(name), " ") {
                        if strings.HasPrefix(s, prefix) { s = s[len(prefix):] }
                        if s == "" { continue }
                        l = append(l, prefix, s)
                }
                return
        }
        includes := ls("this.includes", "-I")
        libdirs := ls("this.libdirs", "-L")
        libs := ls("this.libs", "-l")

        cmdAs  := gccNewCommand("as",  "-c")
        cmdGcc := gccNewCommand("gcc", "-c")
        cmdGxx := gccNewCommand("g++", "-c")

        for _, c := range []*gccCommand{cmdAs, cmdGcc, cmdGxx} {
                c.args, c.libdirs, c.libs = append(c.args, includes...), libdirs, libs
        }

        as := drawSourceTransformActions(sources, func(src string) (name string, c command) {
                var fr *filerule
                if fi, err := os.Stat(src); err != nil {
                        fr = matchFileName(src, gccSourcePatterns)
                } else {
                        fr = matchFile(fi, gccSourcePatterns)
                }

                if fr == nil {
                        errorf(0, "unknown source `%v'", src)
                }
                
                switch fr.name {
                case "asm": c = cmdAs
                case "c":   c = cmdGcc
                        if ld.name == "ld" { ld.name = "gcc" }
                case "c++": c = cmdGxx
                        if ld.name != "g++" && ld.name != "ar" { ld.name = "g++" }
                default:
                        errorf(0, "unknown language for source `%v'", src)
                }

                name = src + ".o"
                return
        })

        m.action.prequisites = append(m.action.prequisites, as...)

        //fmt.Printf("module: %v, %v, %v\n", m.name, m.action.targets, len(m.action.prequisites))
        return m.action != nil
}

func (gcc *_gcc) useModule(p *parser, m *module) bool {
        return false
}

type gccCommand struct {
        excmd
        args []string
        libdirs, libs []string
}

func gccNewCommand(name string, args ...string) *gccCommand {
        return &gccCommand{
                excmd{ name: name, },
                args, []string{}, []string{},
        }
}

func (c *gccCommand) execute(targets []string, prequisites []string) bool {
        var args []string
        var target = targets[0]

        if c.name == "ar" || strings.HasSuffix(c.name, "-ar") {
                args = append(c.args, target)
        } else {
                args = append([]string{ "-o", target, }, c.args...)
        }

        args = append(args, prequisites...)
        if 0 < len(c.libs) {
                args = append(args, append(c.libdirs, c.libs...)...)
        }
        return c.run(target, args...)
}
