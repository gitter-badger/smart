package smart

import (
        "fmt"
        "os"
        "strings"
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
        a *action
}

func (gcc *_gcc) setupModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); panic(p.newError(0, "no module"))
        }

        if m.action == nil {
                m.action = newAction(m.name, nil)
                switch m.kind {
                case "exe":
                        m.action.command = gccNewCommand("ld")
                case "shared":
                        if !strings.HasSuffix(m.action.target, ".so") {
                                m.action.target = m.action.target + ".so"
                        }
                        m.action.command = gccNewCommand("ld", "-shared")
                case "static":
                        if !strings.HasPrefix(m.action.target, "lib") {
                                m.action.target = "lib" + m.action.target
                        }
                        if !strings.HasSuffix(m.action.target, ".a") {
                                m.action.target = m.action.target + ".a"
                        }
                        m.action.command = gccNewCommand("ar", "crs")
                default:
                        p.stepLineBack(); panic(p.newError(0, fmt.Sprintf("unknown type `%v'", m.kind)))
                }
        }
        return true
}

func (gcc *_gcc) buildModule(p *parser, args []string) bool {
        var m *module
        if m = p.module; m == nil {
                p.stepLineBack(); panic(p.newError(0, "no module"))
        }

        if m.action == nil {
                p.stepLineBack(); panic(p.newError(0, "no action for `%v'", p.module.name))
                return false
        }

        if m.action.command == nil {
                p.stepLineBack(); panic(p.newError(0, "no command for `%v'", p.module.name))
                return false
        }

        var ld *gccCommand
        if l, ok := m.action.command.(*gccCommand); !ok {
                p.stepLineBack(); panic(p.newError(0, "internal: wrong module command"))
        } else {
                ld = l
        }

        sources := p.getModuleSources()
        for _, src := range sources {
                a, asrc := newAction(src + ".o", nil), newAction(src, nil)

                var fr *filerule
                if fi, err := os.Stat(src); err != nil {
                        fr = matchFileName(src, gccSourcePatterns)
                } else {
                        fr = matchFile(fi, gccSourcePatterns)
                }

                if fr == nil {
                        panic(p.newError(0, fmt.Sprintf("unknown source `%v'", src)))
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

        return m.action != nil
}

func (gcc *_gcc) processFile(dname string, fi os.FileInfo) {
        fr := matchFile(fi, gccSourcePatterns)
        if fr == nil {
                return
        }

        if gcc.a == nil {
                gcc.a = newAction("a.out", nil)
                gcc.a.command = gccNewCommand("ld")
        }

        ld := gcc.a.command.(*gccCommand)

        a, asrc := newAction(dname + ".o", nil), newAction(dname, nil)
        switch fr.name {
        case "c":
                a.command = gccNewCommand("gcc", "-c")
                if ld.name == "ld" { ld.name = "gcc" }
        case "c++":
                a.command = gccNewCommand("g++", "-c")
                if ld.name != "g++" { ld.name = "g++" }
        }

        a.prequisites = append(a.prequisites, asrc)
        gcc.a.prequisites = append(gcc.a.prequisites, a)
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

func (c *gccCommand) execute(target string, prequisites []string) bool {
        var args []string

        if c.name == "ar" {
                args = append(c.args, target)
        } else {
                args = append([]string{ "-o", target, }, c.args...)
        }

        for _, p := range prequisites {
                //print("gcc: TODO: "+c.name+", "+target+", "+p+"\n")
                args = append(args, p)
        }
        return c.run(target, args...)
}
