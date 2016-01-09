/*
        Package smart builds complex project faster in the simple way.


*/
package smart

import (
        "flag"
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
