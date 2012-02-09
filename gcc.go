package smart

import (
        "os"
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

func (gcc *_gcc) processFile(dname string, fi os.FileInfo) {
        fr := matchFile(fi, gccSourcePatterns)
        if fr == nil {
                return
        }

        if gcc.a == nil {
                gcc.a = makeAction("a.out")
                gcc.a.command = gccNewCommand("ld")
        }

        ld := gcc.a.command.(*gccCommand)

        a, asrc := makeAction(dname + ".o"), makeAction(dname)
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

func (gcc *_gcc) updateAll() {
        gcc.a.update()
}

func (gcc *_gcc) cleanAll() {
        gcc.a.clean()
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
        args := append([]string{ "-o", target, }, c.args...)
        for _, p := range prequisites {
                //print("gcc: TODO: "+c.name+", "+target+", "+p+"\n")
                args = append(args, p)
        }
        return c.run(args...)
}
