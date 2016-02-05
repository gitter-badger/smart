//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//

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
        flagA = flag.Bool("a", false, "auto mode")
        flagG = flag.Bool("g", true, "ignore names like \".git\", \".svn\", etc.")
        flagO = flag.String("o", "", "output directory")
        flagC = flag.String("C", "", "change directory")
        flagT = flag.String("T", "", "traverse")
        flagV = flag.Bool("v", false, "prompt command")
        flagVV = flag.Bool("V", false, "print command verbosely")
)

func GetFlagA() bool    { return *flagA }
func GetFlagG() bool    { return *flagG }
func GetFlagO() string  { return *flagO }
func GetFlagC() string  { return *flagC }
func GetFlagT() string  { return *flagT }
func GetFlagV() bool    { return *flagV }
func GetFlagVV() bool   { return *flagVV }

func SetFlagA(v bool)   { *flagA = v }
func SetFlagG(v bool)   { *flagG = v }
func SetFlagO(v string) { *flagO = v }
func SetFlagC(v string) { *flagC = v }
func SetFlagT(v string) { *flagT = v }
func SetFlagV(v bool)   { *flagV = v }
func SetFlagVV(v bool)  { *flagVV = v }

type smarterror struct {
        message string
}

func (e *smarterror) String() string {
        return e.message
}

func Fatal(f string, a ...interface{}) {
        errorf(0, f, a...)
}

func Message(s string, a ...interface{}) {
        message(s, a...)
}

func Verbose(s string, a ...interface{}) {
        verbose(s, a...)
}

// errorf throw a panic message
func errorf(num int, f string, a ...interface{}) {
        panic(&smarterror{ fmt.Sprintf(f, a...) })
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

// split split a string by space or tab
func split(str string) (items []string) {
        /*
        a := strings.Split(str, " ")
        for _, s := range a {
                if strings.TrimSpace(s) != "" {
                        items = append(items, s)
                }
        } */
        items = strings.Fields(str)
        return
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
