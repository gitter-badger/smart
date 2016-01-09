/*
        Package smart builds complex project faster in the simple way.


*/
package smart

import (
        "flag"
        "fmt"
        "strings"
)

var (
        //flag_a = flag.Bool("a", false, "automode")
        flagG = flag.Bool("g", true, "ignore names like \".git\", \".svn\", etc.")
        flagO = flag.String("o", "", "output directory")
        flagV = flag.Bool("v", false, "prompt command")
        flagC = flag.String("C", "", "change directory")
        flagT = flag.String("T", "", "traverse")
        flagVV = flag.Bool("V", false, "print command verbosely")
)

type smarterror struct {
        number int
        message string
}

func (e *smarterror) String() string {
        return fmt.Sprintf("%v (%v)", e.message, e.number)
}

// errorf throw a panic message
func errorf(num int, f string, a ...interface{}) {
        panic(&smarterror{
                number: num,
                message: fmt.Sprintf(f, a...),
        })
}

// verbose prints a message if `V' flag is enabled
func verbose(s string, a ...interface{}) {
        if *flagVV {
                message(s, a...)
        }
}

// message prints a message
func message(s string, a ...interface{}) {
        if !strings.HasPrefix(s, "smart:") {
                s = "smart: " + s
        }
        if !strings.HasSuffix(s, "\n") {
                s = s + "\n"
        }
        fmt.Printf(s, a...)
}

// splitVarArgs split arguments in the form of "NAME=value" with the others.
func splitVarArgs(args []string) (vars map[string]string, rest []string) {
        vars = make(map[string]string, 10)

        for _, arg := range args {
                if i := strings.Index(arg, "="); 0 < i /* false at '=foo' */ {
                        vars[strings.TrimSpace(arg[0:i])] = strings.TrimSpace(arg[i+1:])
                } else {
                        rest = append(rest, arg)
                }
        }

        return
}

// Main starts build from the command line.
func Main() {
        flag.Parse()

        vars, cmds := splitVarArgs(flag.Args())

        if 0 == len(cmds) {
                cmds = append(cmds, "update")
        }

        Build(vars, cmds);
}
