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
        { "c", ^os.ModeType, `\.(c)$` },
        { "c++", ^os.ModeType, `\.(cpp|cxx|cc|CC|C)$` },
        { "header", ^os.ModeType, `\.(h)$` },
}

type _gcc struct {
}

func (gcc *_gcc) setupModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); errorf(0, "no module")
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
                p.stepLineBack(); errorf(0, fmt.Sprintf("unknown type `%v'", m.kind))
        }

        c.mkdir = filepath.Join(out, m.name)
        m.action = newAction(filepath.Join(out, m.name, name), c)
        return true
}

func (gcc *_gcc) buildModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack();
                errorf(0, "no module")
        }

        if m.action == nil {
                //p.stepLineBack();
                errorf(0, "no action for `%v'", p.module.name)
                return false
        }

        if m.action.command == nil {
                p.stepLineBack(); errorf(0, "no command for `%v'", p.module.name)
                return false
        }

        var ld *gccCommand
        if l, ok := m.action.command.(*gccCommand); !ok {
                p.stepLineBack(); errorf(0, "internal: wrong module command")
        } else {
                ld = l
        }

        sources := p.getModuleSources()
        if len(sources) == 0 {
                p.stepLineBack(); errorf(0, "no sources for `%v'", p.module.name)
        }

        for _, src := range sources {
                a, asrc := newAction(src + ".o", nil), newAction(src, nil)

                var fr *filerule
                if fi, err := os.Stat(src); err != nil {
                        fr = matchFileName(src, gccSourcePatterns)
                } else {
                        fr = matchFile(fi, gccSourcePatterns)
                }

                if fr == nil {
                        errorf(0, fmt.Sprintf("unknown source `%v'", src))
                }

                switch fr.name {
                case "c":
                        a.command = gccNewCommand("gcc", "-c")
                        if ld.name == "ld" { ld.name = "gcc" }
                case "c++":
                        a.command = gccNewCommand("g++", "-c")
                        if ld.name != "g++" && ld.name != "ar" { ld.name = "g++" }
                }

                a.prequisites = append(a.prequisites, asrc)
                m.action.prequisites = append(m.action.prequisites, a)
        }

        //fmt.Printf("module: %v, %v, %v\n", m.name, m.action.targets, len(m.action.prequisites))

        return m.action != nil
}

type gccCommand struct {
        execCommand
        args []string
}

func gccNewCommand(name string, args ...string) *gccCommand {
        return &gccCommand{
                execCommand{ name: name, },
                args,
        }
}

func (c *gccCommand) execute(targets []string, prequisites []string) bool {
        var args []string
        var target = targets[0]

        if c.name == "ar" {
                args = append(c.args, target)
        } else {
                args = append([]string{ "-o", target, }, c.args...)
        }

        args = append(args, prequisites...)
        return c.run(target, args...)
}
