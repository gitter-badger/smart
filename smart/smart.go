package smart

import (
        "flag"
        "os"
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

// Main starts build from the command line.
func Main() {
        flag.Parse()

        var cmds []string
        var vars = map[string]string{}
        for _, arg := range os.Args[1:] {
                if arg[0] == '-' { continue }
                if i := strings.Index(arg, "="); 0 < i /* false at '=foo' */ {
                        vars[arg[0:i]] = arg[i+1:]
                        continue
                }
                cmds = append(cmds, arg)
        }

        if 0 == len(cmds) {
                cmds = append(cmds, "update")
        }

        Build(vars, cmds);
}
